package api

// CustodyWarning is printed before a controller server is trusted by another
// box (spec D10): what an encrypted controller key does and does not protect.
const CustodyWarning = "root on this controller server, with the key unlocked, can sign routine intents for every box that trusts it. A gate upgrade, which replaces a box's gate, also needs an approval signature unless that box's policy.json makes it routine. Encryption protects the key at rest, in backups and against disk theft; it does not protect it from live root."
