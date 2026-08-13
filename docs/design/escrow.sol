// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title JumpgateEscrow — prepaid escrow with signed-voucher settlement
/// @notice One operator, one stablecoin, one domain. The operator picks the
///         chain and the token at deploy time. Deploy one instance per domain.
///
/// This contract is the on-chain half of docs/design/metered-api-keys.md,
/// section 5. The flow is:
///   1. The customer prepays into escrow. This is the on-chain anchor.
///   2. The operator meters usage off-chain and collects signed vouchers.
///   3. The operator settles on a cadence. It submits the latest voucher.
///
/// The chain never sees how many requests the operator served. Two rules bound
/// the loss instead:
///   - Escrow ceiling. The customer can only lose what it deposited.
///   - Signed monotonic total. The operator can only draw what the customer
///     signed for, so it can never draw more than it served.
///
/// The customer exits on a timelock. The operator keeps a window to submit the
/// final voucher before the funds leave.
///
/// Status: design sketch. Audit it before you put real value behind it. It is
/// intentionally thin — no owner, no upgrade path, no pause. To change the
/// operator or the token, deploy a fresh instance.

interface IERC20 {
    function transfer(address to, uint256 value) external returns (bool);
    function transferFrom(address from, address to, uint256 value) external returns (bool);
}

contract JumpgateEscrow {
    // ---- fixed at deploy: the operator sets these once, per domain ----
    IERC20  public immutable token;         // the stablecoin
    address public immutable operator;      // serves eRPC and draws earnings
    uint256 public immutable withdrawDelay; // seconds the customer waits to exit

    // ---- per-customer state ----
    mapping(address => uint256) public balance;        // escrowed, not yet drawn
    mapping(address => uint256) public claimed;        // monotonic total drawn
    mapping(address => uint256) public withdrawableAt; // 0 means no exit pending

    // ---- EIP-712 voucher domain (binds a voucher to this chain + contract) ----
    // The domain separator carries the chain id and this address. A voucher is
    // therefore valid on one chain and one contract only. This is what makes
    // "one instance per domain" safe: a voucher can never replay across domains.
    bytes32 private immutable _domainSeparator;
    bytes32 private constant _VOUCHER_TYPEHASH =
        keccak256("Voucher(address customer,uint256 cumulative)");

    event Deposited(address indexed customer, uint256 amount, uint256 balance);
    event Claimed(address indexed customer, uint256 amount, uint256 drawn);
    event WithdrawRequested(address indexed customer, uint256 readyAt);
    event Withdrawn(address indexed customer, uint256 amount);

    error NotOperator();
    error StaleVoucher();
    error BadSignature();
    error NoExitPending();
    error ExitNotReady();
    error TransferFailed();

    constructor(IERC20 token_, address operator_, uint256 withdrawDelay_) {
        token = token_;
        operator = operator_;
        withdrawDelay = withdrawDelay_;
        _domainSeparator = keccak256(abi.encode(
            keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
            keccak256("JumpgateEscrow"),
            keccak256("1"),
            block.chainid,
            address(this)
        ));
    }

    /// The customer prepays. The customer approves this contract first.
    /// A gasless top-up can wrap this with EIP-3009 `transferWithAuthorization`
    /// on the token, as section 5.2 describes. The core stays a plain pull.
    function deposit(uint256 amount) external {
        _pull(msg.sender, amount);
        uint256 b = balance[msg.sender] + amount;
        balance[msg.sender] = b;
        delete withdrawableAt[msg.sender]; // a fresh deposit cancels a pending exit
        emit Deposited(msg.sender, amount, b);
    }

    /// The operator settles. It submits the customer's latest signed voucher.
    /// `cumulative` is the total value the customer authorizes over the whole
    /// relationship, not the amount of one draw. The contract pays only the new
    /// part, and caps that part at the escrow balance.
    ///
    /// A short balance does not burn the voucher. `claimed` only advances by
    /// what the contract paid, so the operator can resubmit the same voucher
    /// after the customer tops up.
    function claim(address customer, uint256 cumulative, uint8 v, bytes32 r, bytes32 s) external {
        if (msg.sender != operator) revert NotOperator();
        if (cumulative <= claimed[customer]) revert StaleVoucher();
        if (_recover(customer, cumulative, v, r, s) != customer) revert BadSignature();

        uint256 owed = cumulative - claimed[customer];
        uint256 bal = balance[customer];
        uint256 pay = owed < bal ? owed : bal; // never draw past the escrow
        claimed[customer] += pay;
        balance[customer] = bal - pay;
        _push(operator, pay);
        emit Claimed(customer, pay, claimed[customer]);
    }

    /// The customer starts the exit clock. The operator can still settle until
    /// the clock runs out.
    function requestWithdraw() external {
        uint256 readyAt = block.timestamp + withdrawDelay;
        withdrawableAt[msg.sender] = readyAt;
        emit WithdrawRequested(msg.sender, readyAt);
    }

    /// After the delay, the customer takes back the balance the operator did
    /// not draw.
    function withdraw() external {
        uint256 readyAt = withdrawableAt[msg.sender];
        if (readyAt == 0) revert NoExitPending();
        if (block.timestamp < readyAt) revert ExitNotReady();
        uint256 amount = balance[msg.sender];
        delete balance[msg.sender];
        delete withdrawableAt[msg.sender];
        _push(msg.sender, amount);
        emit Withdrawn(msg.sender, amount);
    }

    // ---- EIP-712 recover ----
    function _recover(address customer, uint256 cumulative, uint8 v, bytes32 r, bytes32 s)
        private view returns (address)
    {
        bytes32 structHash = keccak256(abi.encode(_VOUCHER_TYPEHASH, customer, cumulative));
        bytes32 digest = keccak256(abi.encodePacked("\x19\x01", _domainSeparator, structHash));
        return ecrecover(digest, v, r, s);
    }

    // ---- token plumbing: tolerate stablecoins that return no bool (e.g. USDT) ----
    function _pull(address from, uint256 amount) private {
        (bool ok, bytes memory ret) = address(token).call(
            abi.encodeCall(IERC20.transferFrom, (from, address(this), amount))
        );
        if (!ok || (ret.length != 0 && !abi.decode(ret, (bool)))) revert TransferFailed();
    }

    function _push(address to, uint256 amount) private {
        (bool ok, bytes memory ret) = address(token).call(
            abi.encodeCall(IERC20.transfer, (to, amount))
        );
        if (!ok || (ret.length != 0 && !abi.decode(ret, (bool)))) revert TransferFailed();
    }
}
