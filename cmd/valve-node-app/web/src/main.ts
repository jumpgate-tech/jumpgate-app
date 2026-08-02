// Hash-routed SPA shell: #/ (panel), #/panel, #/targets, #/rpc,
// #/analytics/<gid>, #/machine/<id>, #/security/<id>, #/diag/<id>,
// #/settings. No framework — each screen module renders into a shared content
// element and returns a cleanup function (closes any EventSource / timers)
// that main.ts calls before routing away.
//
// The empty hash (and any unrecognised route) lands on the Easy-Button panel
// (panel.ts), which replaced the old capability-detected home (home.ts, still
// present but no longer routed to) as the default landing.
//
// #/machine/<id> is the collapsed per-machine page (setup + dashboard + logs +
// devnet as expandable sections). The four routes it replaced — #/setup,
// #/dash, #/logs, #/services — are kept alive as REDIRECTS to it so existing
// deep links still land somewhere, preserving the id.
import "./style.css";
import { initTheme } from "./theme";
import { renderAnalytics } from "./analytics";
import { renderDiagnostics } from "./diag";
import { renderMachine } from "./machine";
import { renderPanel } from "./panel";
import { renderSecurity } from "./security";
import { renderSettings } from "./settings";
import { renderShell } from "./ui";
import { renderRPC } from "./rpc";
import { renderTargets } from "./targets";

type Cleanup = () => void;

// Stamp the resolved light/dark/system theme onto <html> before the shell
// renders, so the first paint is already in the right theme (no dark flash).
initTheme();

const appRoot = document.querySelector<HTMLDivElement>("#app")!;
const { contentEl, setActiveNav } = renderShell(appRoot);

let currentCleanup: Cleanup | null = null;

interface Route {
  screen: string;
  id?: string;
}

function parseHash(): Route {
  const hash = location.hash.replace(/^#\/?/, "");
  const parts = hash.split("/").filter(Boolean);
  if (parts.length === 0) return { screen: "home" };
  const [screen, rawId] = parts;
  if (
    screen === "machine" ||
    // The four former per-machine routes are still parsed (id-scoped) so the
    // switch below can redirect them to #/machine/<id> with the id intact.
    screen === "setup" ||
    screen === "dash" ||
    screen === "logs" ||
    screen === "security" ||
    screen === "diag" ||
    screen === "services" ||
    // analytics is scoped to a GATEWAY, not a machine — the id here is a
    // gateway id, which is why it lands on #/rpc rather than #/targets when
    // it is missing.
    screen === "analytics"
  ) {
    return { screen, id: rawId ? decodeURIComponent(rawId) : undefined };
  }
  return { screen: screen ?? "targets" };
}

// mount gives a screen a brand-new child element of contentEl to render
// into and discards the previous one. Screens attach their delegated click
// listeners (via onAction) to the root element they're passed, not to
// contentEl itself — so a fresh node per screen means those listeners are
// discarded with the old node on every navigation instead of stacking up
// on the page-lifetime contentEl.
function mount(render: (root: HTMLElement) => Cleanup): Cleanup {
  const screenEl = document.createElement("div");
  contentEl.replaceChildren(screenEl);
  return render(screenEl);
}

function route(): void {
  if (currentCleanup) {
    try {
      currentCleanup();
    } catch {
      // A screen's cleanup failing must not block navigating away from it.
    }
    currentCleanup = null;
  }

  const { screen, id } = parseHash();
  setActiveNav(screen);

  switch (screen) {
    case "machine":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderMachine(root, id));
      break;
    // The per-machine routes collapsed into #/machine/<id>. They still parse
    // (see parseHash) purely so an old deep link can be redirected with its id
    // preserved — each is now a section of the machine page, not a screen.
    case "setup":
    case "dash":
    case "logs":
    case "services":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      location.hash = `#/machine/${encodeURIComponent(id)}`;
      return;
    case "security":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderSecurity(root, id));
      break;
    case "diag":
      if (!id) {
        location.hash = "#/targets";
        return;
      }
      currentCleanup = mount((root) => renderDiagnostics(root, id));
      break;
    case "analytics":
      if (!id) {
        location.hash = "#/rpc";
        return;
      }
      currentCleanup = mount((root) => renderAnalytics(root, id));
      break;
    case "rpc":
      // No id: eRPC is a layer over the whole fleet, not a machine's service,
      // so its screen is not scoped to a target.
      currentCleanup = mount((root) => renderRPC(root));
      break;
    case "settings":
      currentCleanup = mount((root) => renderSettings(root));
      break;
    case "targets":
      currentCleanup = mount((root) => renderTargets(root));
      break;
    case "panel":
      currentCleanup = mount((root) => renderPanel(root));
      break;
    // The empty hash and anything unrecognised land on the Easy-Button panel,
    // which is now the default landing (it replaces the old capability-detected
    // home for that role).
    case "home":
    default:
      currentCleanup = mount((root) => renderPanel(root));
      break;
  }
}

window.addEventListener("hashchange", route);
route();
