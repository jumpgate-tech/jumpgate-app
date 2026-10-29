// Alerts (K): what the fleet says is wrong now, worst first.
import { alertLine, alerts } from "../shell/alerts.ts";
import type { App, Screen } from "../shell/app.ts";

export class AlertsScreen implements Screen {
  readonly id = "alerts";
  readonly title = "Alerts";
  constructor(private readonly app: App) {}
  view(): string[] {
    const f = this.app.state.fleet;
    if (!f) return ["n/a: no fleet yet"];
    const l = alerts(f, new Date()).map(alertLine);
    return [...(this.app.state.fleetStale ? ["~ the fleet stream is reconnecting: these may be stale", ""] : []), ...(l.length ? l : ["No alerts."])];
  }
}
