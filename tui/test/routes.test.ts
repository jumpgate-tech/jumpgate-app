import { expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { check, goRoutes } from "../scripts/check-routes.ts";
import { ROUTES } from "../src/api/routes.ts";
import { samples } from "./samples.ts";
import { shapeErrors } from "./shape.ts";

const dir = join(import.meta.dir, "fixtures");
const files = readdirSync(dir).filter((f) => f.endsWith(".json"));

test("there is a sample for every fixture, and a fixture for every sample", () => {
  expect(files.map((f) => f.replace(/\.json$/, "")).sort()).toEqual(Object.keys(samples).sort());
});

for (const f of files) {
  test(`fixture ${f} matches its TS type`, () => {
    const name = f.replace(/\.json$/, "");
    expect(shapeErrors(JSON.parse(readFileSync(join(dir, f), "utf8")), samples[name])).toEqual([]);
  });
}

test("shapeErrors catches a renamed field and a changed type", () => {
  expect(shapeErrors({ b: 1 }, { a: 1 })).toEqual(["$.a: missing in the Go fixture"]);
  expect(shapeErrors({ a: "1" }, { a: 1 })).toEqual(["$.a: fixture is string, TS sample is number"]);
  expect(shapeErrors({ a: null }, { a: { b: 1 } })).toEqual([]);
});

test("every ROUTES entry is a Go route", () => {
  const { missing } = check(ROUTES, goRoutes(join(import.meta.dir, "..", "..", "internal", "server")));
  expect(missing).toEqual([]);
});

test("a renamed Go route fails the check", () => {
  expect(check(["GET /api/vault"], new Set(["GET /api/vault2"])).missing).toEqual(["GET /api/vault"]);
});

test("check-routes exits 0", () => {
  const r = Bun.spawnSync(["bun", "run", "scripts/check-routes.ts"], { cwd: join(import.meta.dir, "..") });
  expect(r.exitCode).toBe(0);
});
