import { afterEach, expect, test } from "bun:test";
import { JobsScreen } from "../../src/screens/jobs.ts";
import { key } from "../../src/shell/keys.ts";
import { json, sseResponse } from "../fake.ts";
import { fixture, mount, say, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const routes = {
  "GET /api/jobs/alpha/j1": () => json(fixture("jobdetail")),
  "GET /api/jobs/alpha/j1/output/stream": () => sseResponse([`data: ${JSON.stringify(fixture("joboutput"))}\n\n`], true),
  "POST /api/jobs/alpha/j1/decide": () => json({ ...fixture("jobview"), state: "running", question: undefined }),
  "POST /api/jobs/alpha/j1/adopt": () => json(fixture("jobview")),
};

const open = async () => {
  m = await mount((app) => [new JobsScreen(app)], { routes });
  await m.waitFor("alpha/j1");
  await m.press("return");
  return m.waitFor("imported block");
};

test("the inbox lists jobs; Enter shows steps, the question and live output", async () => {
  const f = await open();
  expect(f).toContain("[done   ] Stop");
  expect(f).toContain("[asking ] Swap binary");
  expect(f).toContain("Swap the binary now?");
  expect(f).toContain("1 Go - swap and restart");
  expect(f).toContain("geth imported block");
});

test("a approves after a confirm that shows the step text, with the api.JobDecision shape", async () => {
  await open();
  const c = await m!.press("a");
  expect(c).toContain("Swap the binary now?");
  expect(c).toContain("Go");
  await m!.press("return"); // Enter alone never answers
  expect(m!.fake.requests.some((r) => r.path.endsWith("/decide"))).toBe(false);
  await say(m!, "go");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path.endsWith("/decide")));
  const req = m!.fake.requests.find((r) => r.path.endsWith("/decide"))!;
  expect(JSON.parse(req.body)).toEqual({ decision: "q1", option: "go" });
  expect(Object.keys(JSON.parse(req.body)).sort()).toEqual(Object.keys(fixture("jobdecision")).sort());
});

test("x refuses with the abort option; n at the confirm sends nothing", async () => {
  await open();
  await m!.press("x", "escape");
  expect(m!.fake.requests.some((r) => r.path.endsWith("/decide"))).toBe(false);
  await m!.press("x");
  await say(m!, "abort");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path.endsWith("/decide")));
  expect(JSON.parse(m!.fake.requests.find((r) => r.path.endsWith("/decide"))!.body)).toEqual({ decision: "q1", option: "abort" });
});

test("A adopts after a confirm", async () => {
  await open();
  await m!.press(key("a", { seq: "A", shift: true }));
  await say(m!, "j1");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path.endsWith("/adopt")));
});
