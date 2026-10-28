// start: create the renderer, mount the screens, connect, run until quit.
import { createCliRenderer, type CliRenderer } from "@opentui/core";
import { discover } from "../api/discover.ts";
import type { Args } from "../main.ts";
import { VERSION } from "../main.ts";
import { ConnectScreen } from "../screens/connect.ts";
import { FleetScreen } from "../screens/fleet.ts";
import { JobsScreen } from "../screens/jobs.ts";
import { KeysScreen } from "../screens/keys.ts";
import { HostsScreen } from "../screens/hosts.ts";
import { ModulesScreen } from "../screens/modules.ts";
import { ServicesScreen } from "../screens/services.ts";
import { VPNScreen } from "../screens/vpn.ts";
import { NodeScreen } from "../screens/node.ts";
import { VaultScreen } from "../screens/vault.ts";
import { App } from "./app.ts";

/** mountApp registers the slice's screens on renderer. */
export function mountApp(renderer: CliRenderer, args: Args, version = VERSION): App {
  const app = new App(renderer, { myVersion: version });
  app.add(new ConnectScreen(app, { discover: () => discover(process.env, args.socket) }));
  app.add(new FleetScreen(app));
  app.add(new HostsScreen(app));
  app.add(new ModulesScreen(app));
  app.add(new ServicesScreen(app));
  app.add(new VPNScreen(app));
  app.add(new JobsScreen(app));
  app.add(new VaultScreen(app));
  app.add(new KeysScreen(app));
  app.add(new NodeScreen(app), false);
  return app;
}

export async function start(args: Args): Promise<number> {
  const renderer = await createCliRenderer({ exitOnCtrlC: false });
  const app = mountApp(renderer, args);
  app.go("connect");
  await app.done;
  return 0;
}
