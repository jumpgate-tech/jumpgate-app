import { expect, test } from "bun:test";
import { checkSSHArgv, localSSH } from "../src/sshargv.ts";

const good = [
  "ssh", "-p", "22", "-i", "/home/u/.ssh/id",
  "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=/home/u/.jumpgate/confirmed_hosts",
  "-o", "GlobalKnownHostsFile=none", "-o", "ProxyJump=none", "-o", "ProxyCommand=none",
  "-l", "root", "--", "10.0.0.2",
];

test("the server's shape passes", () => {
  expect(checkSSHArgv(good)).toBeNull();
  expect(checkSSHArgv(good.filter((_, i) => i !== 3 && i !== 4))).toBeNull(); // no -i
  expect(checkSSHArgv(["/usr/bin/ssh", ...good.slice(1)])).toBeNull();
});

const replace = (i: number, v: string) => good.map((x, j) => (j === i ? v : x));
const bad: [string, string[]][] = [
  ["ProxyCommand to a shell", replace(14, "ProxyCommand=sh -c id")],
  ["-oProxyCommand glued", [...good.slice(0, 17), "-oProxyCommand=sh", ...good.slice(17)]],
  ["-oLocalCommand", [...good.slice(0, 5), "-oLocalCommand=id", ...good.slice(5)]],
  ["an extra -o", [...good.slice(0, 17), "-o", "PermitLocalCommand=yes", ...good.slice(17)]],
  ["a missing --", good.filter((x) => x !== "--")],
  ["a different argv[0]", replace(0, "/tmp/evil")],
  ["sh as argv[0]", replace(0, "sh")],
  ["a bad port", replace(2, "022")],
  ["a host starting with -", replace(18, "-oProxyCommand=id")],
  ["an element after the host", [...good, "id"]],
  ["a newline", replace(16, "root\nid")],
  ["a $ in the key path", replace(4, "/tmp/$HOME")],
  ["a quote in known_hosts", replace(8, `UserKnownHostsFile=/a'b`)],
  ["no known_hosts", replace(8, "UserKnownHostsFile=")],
  ["StrictHostKeyChecking=no", replace(6, "StrictHostKeyChecking=no")],
];
for (const [name, argv] of bad) test(`refuses ${name}`, () => expect(checkSSHArgv(argv)).not.toBeNull());

test("the local ssh runs by name, never the server's argv[0]", () => {
  expect(localSSH(["/tmp/evil", "-p", "22"], "linux")).toEqual(["ssh", "-p", "22"]);
  expect(localSSH(["C:\\x\\ssh.exe", "-p", "22"], "win32")).toEqual(["ssh.exe", "-p", "22"]);
});
