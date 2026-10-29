// guide: the first-run checklist, derived from the live state only. Each
// step says whether it is done, why it matters in one sentence, and which
// existing action does it (a screen, and the key to press there).
import type { ControllerView, Fleet, ModulesView, VaultView } from "../api/types.ts";
import { boxHealth } from "./statusbar.ts";

export interface GuideFacts {
  connected: boolean;
  vault?: VaultView;
  controller?: ControllerView;
  fleet?: Fleet;
  /** Each paired box's modules, once read; missing means not checked yet. */
  modules?: Record<string, ModulesView>;
  platform: string;
}

export interface Step {
  id: string;
  title: string;
  done: boolean;
  optional: boolean;
  why: string;
  /** how is what to do, in words, naming the screen and its key. */
  how: string;
  /** go is the screen to open, and the key to press there (an existing action). */
  go?: { screen: string; key?: string };
}

/** The links that go through a paired gate (the API's names for them). */
const GATE_LINKS: Record<string, true> = { agent: true, agent_local: true };
const paired = (link: string, gate?: string) => GATE_LINKS[link] === true || !!gate;

export function guideSteps(f: GuideFacts): Step[] {
  const v = f.vault;
  const rows = f.fleet?.rows ?? [];
  const steps: Step[] = [];
  const add = (s: Omit<Step, "optional">, optional = false) => steps.push({ ...s, optional });

  add({
    id: "server",
    title: "Server running",
    done: f.connected,
    why: "The dashboard, the CLI and every gate talk to this one server.",
    how: f.connected ? "Running." : "Connect: press s to start it.",
    go: { screen: "connect", key: f.connected ? undefined : "s" },
  });

  const vs = v?.state;
  const vaultHow =
    vs === "unlocked" ? "Unlocked."
    : vs === "locked" ? "Vault: press u and type the passphrase."
    : vs === "none" ? "Vault: press i to create it (init)."
    : vs === "unusable" ? "Vault: its file cannot be used; press R to restore a backup."
    : "Connect to the server first.";
  add({
    id: "vault",
    title: "Vault created and unlocked",
    done: vs === "unlocked",
    why: "It holds the controller key and the tokens, encrypted with your passphrase.",
    how: vaultHow,
    go: { screen: "vault", key: vs === "locked" ? "u" : vs === "none" ? "i" : vs === "unusable" ? "R" : undefined },
  });

  const pass = v?.slots.filter((s) => s === "passphrase").length ?? 0;
  add({
    id: "recovery",
    title: "Recovery code added",
    done: pass >= 2,
    why: "If the passphrase is lost, the recovery code is the only way back into the vault.",
    how: pass >= 2 ? "Added." : vs === "none" || !v ? "Create the vault first." : "Vault: press k; the code prints once on the terminal.",
    go: { screen: "vault", key: v && vs !== "none" && pass < 2 ? "k" : undefined },
  });

  const c = f.controller;
  const keyDone = c?.state === "ok" || (c?.state === "locked" && !!c.recorded);
  add({
    id: "key",
    title: "Controller key present",
    done: keyDone,
    why: "It signs every command; gates run only what this key signed.",
    how: keyDone ? "Present." : c?.state === "missing" ? "Keys: press c to create it." : c ? `Keys: the key is ${c.state}; see the reason there.` : "Connect to the server first.",
    go: { screen: "keys", key: c?.state === "missing" ? "c" : undefined },
  });

  add({
    id: "box",
    title: "First box added",
    done: rows.length > 0,
    why: "A box is a machine that runs your node and its gate.",
    how: rows.length ? "Added." : "Hosts: press a to add one over SSH, or i for an invite to run on the box.",
    go: { screen: "hosts", key: rows.length ? undefined : "a" },
  });

  const pairedRows = rows.filter((r) => paired(r.link, r.agent));
  const healthy = pairedRows.some((r) => boxHealth(r) === "ok");
  add({
    id: "paired",
    title: "Box paired and healthy",
    done: healthy,
    why: "The gate must answer, signed, before jumpgate can run anything on the box.",
    how: healthy ? "Paired and healthy."
      : !rows.length ? "Add a box first."
      : !pairedRows.length ? "Hosts: press a and give the box's installed address to pair it."
      : "Alerts: see what is wrong with the box.",
    go: { screen: !rows.length || !pairedRows.length ? "hosts" : "alerts" },
  });

  const sealed = !!v?.slots.some((s) => s === "systemd-creds" || s === "keystore");
  const linux = f.platform === "linux";
  add(
    {
      id: "seal",
      title: "Vault sealed for auto-unlock",
      done: sealed,
      why: "The service can then unlock the vault on boot without you.",
      how: sealed ? "Sealed." : linux ? "Vault: press s to seal it with the TPM." : "On the Linux service: jumpgate vault seal.",
      go: linux && !sealed ? { screen: "vault", key: "s" } : { screen: "vault" },
    },
    true,
  );

  const mods = Object.values(f.modules ?? {});
  const anyOn = mods.some((m) => m.modules.some((x) => x.active));
  const watch = mods.some((m) => m.modules.some((x) => x.id === "watch" && x.active));
  const checked = f.modules !== undefined;
  add(
    {
      id: "module",
      title: "A module turned on",
      done: anyOn,
      why: "Modules add services to a box, such as a bootnode or a message board.",
      how: anyOn ? "On." : checked ? "Modules: open a box and press space on a module." : "Not checked yet.",
      go: { screen: "modules" },
    },
    true,
  );
  add(
    {
      id: "alerts",
      title: "Alerts watching a box",
      done: watch,
      why: "The watch module checks the box and raises alerts here.",
      how: watch ? "On." : checked ? "Modules: open a box and turn on watch." : "Not checked yet.",
      go: { screen: "modules" },
    },
    true,
  );
  return steps;
}

/** progress counts the required steps. */
export function guideProgress(steps: Step[]): { done: number; total: number } {
  const req = steps.filter((s) => !s.optional);
  return { done: req.filter((s) => s.done).length, total: req.length };
}

/** nextStep is the first required step not done, then the first optional one. */
export function nextStep(steps: Step[]): Step | undefined {
  return steps.find((s) => !s.done && !s.optional) ?? steps.find((s) => !s.done);
}
