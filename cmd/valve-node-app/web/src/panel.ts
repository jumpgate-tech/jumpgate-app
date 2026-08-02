// The Easy-Button panel: a single card (list → network → endpoint) that
// replaces the old capability-detected home as the default landing. This
// module is the scaffold — the icon sprite, the view-state machine, and a
// loading skeleton. Real list/network/endpoint content and the action
// handler land in later tasks (see panelModel.ts for the pure helpers they
// build on).
import "./panel.css";
import * as api from "./api";
import { onAction, escapeHtml } from "./ui";

// Inline SVG sprite (currentColor stroke) — cross-platform, no SF Symbols.
const SPRITE = `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
  <symbol id="p-power" viewBox="0 0 24 24"><line x1="12" y1="3.5" x2="12" y2="11.5"/><path d="M7.5 7a7 7 0 1 0 9 0"/></symbol>
  <symbol id="p-globe" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.5 2.4 2.5 14.6 0 17M12 3.5c-2.5 2.4-2.5 14.6 0 17"/></symbol>
  <symbol id="p-ws" viewBox="0 0 24 24"><path d="M4 9h13l-3.5-3.5M20 15H7l3.5 3.5"/></symbol>
  <symbol id="p-archive" viewBox="0 0 24 24"><path d="M12 3 3 7.5l9 4.5 9-4.5L12 3ZM3 12l9 4.5 9-4.5M3 16.5 12 21l9-4.5"/></symbol>
  <symbol id="p-trace" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5.5"/><path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3"/></symbol>
  <symbol id="p-lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9.5" rx="2.2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></symbol>
  <symbol id="p-pencil" viewBox="0 0 24 24"><path d="M14 5.5l4.5 4.5M4 20l1.2-4.4L16 4.8a2 2 0 0 1 2.8 0l.4.4a2 2 0 0 1 0 2.8L8.4 18.8 4 20Z"/></symbol>
  <symbol id="p-trash" viewBox="0 0 24 24"><path d="M4 6.5h16M9.5 6.5V5a2 2 0 0 1 2-2h1a2 2 0 0 1 2 2v1.5M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6"/></symbol>
  <symbol id="p-copy" viewBox="0 0 24 24"><rect x="9" y="9" width="11" height="11" rx="2.2"/><path d="M5 15V5a2 2 0 0 1 2-2h8"/></symbol>
  <symbol id="p-scale" viewBox="0 0 24 24"><path d="M12 3v18M7 21h10M12 5 5 8m7-3 7 3M5 8l-3 6a3 3 0 0 0 6 0L5 8Zm14 0-3 6a3 3 0 0 0 6 0l-3-6Z"/></symbol>
  <symbol id="p-refresh" viewBox="0 0 24 24"><path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"/></symbol>
  <symbol id="p-chevR" viewBox="0 0 24 24"><path d="M9.5 5.5l6.5 6.5-6.5 6.5"/></symbol>
  <symbol id="p-chevL" viewBox="0 0 24 24"><path d="M14.5 5.5 8 12l6.5 6.5"/></symbol>
  <symbol id="p-plus" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></symbol>
</defs></svg>`;
const ic = (id: string) => `<svg class="p-i"><use href="#p-${id}"/></svg>`;
// Not called yet — renderList/renderNetwork/renderEndpoint start using it in
// Tasks 6-10. Referencing it here keeps tsc's noUnusedLocals quiet until then.
void ic;

type View = { name: "list" } | { name: "network"; chainId: number } | { name: "endpoint"; chainId: number; upstreamId: string };

export function renderPanel(root: HTMLElement): () => void {
  let gw: api.GatewayView | null = null;
  let view: View = { name: "list" };
  let err: string | null = null;
  let poll: number | null = null;

  root.innerHTML = SPRITE + `<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>`;
  const card = root.querySelector<HTMLElement>("#p-card")!;

  async function load(): Promise<void> {
    try {
      const res = await api.getGateways();
      gw = primaryGateway(res.gateways);
      err = null;
    } catch (e) {
      err = message(e);
    }
    render();
  }
  function render(): void {
    card.innerHTML = renderView();
  }
  function renderView(): string {
    if (err) return bandError(err);
    if (view.name === "network") return renderNetwork(gw, view.chainId);
    if (view.name === "endpoint") return renderEndpoint(gw, view.chainId, view.upstreamId);
    return renderList(gw);
  }

  onAction(card, (action, el) => {
    void handleAction(action, el);
  });
  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    /* filled in Tasks 6-10 */
    void action;
    void el;
  }

  void load();
  return () => {
    if (poll) window.clearInterval(poll);
  };
}

// primaryGateway: the panel manages one gateway — the local one, else the first.
function primaryGateway(gws: api.GatewayView[] | null): api.GatewayView | null {
  if (!gws || gws.length === 0) return null;
  return gws.find((g) => g.placement.targetId === "local") ?? gws[0];
}
function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
function bandError(m: string): string {
  return `<div class="p-band" style="padding:16px;color:var(--red)">${escapeHtml(m)}</div>`;
}
function renderList(_gw: api.GatewayView | null): string {
  return `<div class="p-band" style="padding:16px">Panel loading…</div>`;
}
function renderNetwork(_gw: api.GatewayView | null, _c: number): string {
  return "";
}
function renderEndpoint(_gw: api.GatewayView | null, _c: number, _u: string): string {
  return "";
}
