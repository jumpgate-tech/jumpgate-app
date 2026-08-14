import { Navigate, Route, Routes, useLocation, useParams } from "react-router-dom";
import { Panel } from "./screens/Panel/Panel";
import { Rpc } from "./screens/Rpc/Rpc";
import { Security } from "./screens/Security/Security";
import { Diagnostics } from "./screens/Diagnostics/Diagnostics";
import { Settings } from "./screens/Settings/Settings";
import { PrivateAccess } from "./screens/PrivateAccess/PrivateAccess";
import { Targets } from "./screens/Targets/Targets";
import { Analytics } from "./screens/Analytics/Analytics";
import { Machine } from "./screens/Machine/Machine";
import { UpdateBanner } from "./components/UpdateBanner";

const NAV = [
  { to: "#/rpc", nav: "rpc", label: "RPC" },
  { to: "#/vpn", nav: "vpn", label: "Private access" },
  { to: "#/targets", nav: "targets", label: "Machines" },
  { to: "#/settings", nav: "settings", label: "Settings" },
];

export function activeNav(screen: string): string {
  return screen === "machine"
    ? "targets"
    : screen === "home" || screen === "panel"
      ? "rpc"
      : screen;
}

// TRAY_TABS is the desktop (tray) app's bottom navigation. The tray drops the
// browser topbar, so without this the panel was the ONLY reachable screen —
// no way to add a remote machine, choose gateway placement, or reach Settings.
// The panel is "Home"; the rest of the app hangs off the other tabs.
const TRAY_TABS = [
  { to: "#/", key: "home", label: "Home" },
  { to: "#/rpc", key: "rpc", label: "RPC" },
  { to: "#/targets", key: "targets", label: "Machines" },
  { to: "#/vpn", key: "vpn", label: "Private" },
  { to: "#/settings", key: "settings", label: "Settings" },
];

// trayActive marks the current tab. Unlike activeNav (which folds home/panel
// into "rpc" for the browser topbar), the tray has a dedicated Home tab for the
// panel, so home/panel light Home rather than RPC.
export function trayActive(screen: string): string {
  if (screen === "home" || screen === "panel") return "home";
  return activeNav(screen);
}

function RedirectToMachine() {
  const { id } = useParams();
  return <Navigate to={`/machine/${encodeURIComponent(id!)}`} replace />;
}

export function App() {
  const { pathname } = useLocation();
  const screen = pathname.split("/").filter(Boolean)[0] ?? "home";
  const nav = activeNav(screen);
  // The tray/tiny-app build sets this flag before the app loads (webview.Init).
  // There the panel IS the whole window, so the multi-screen topbar nav — which
  // only makes sense for browser-based fleet management — is dropped.
  const tray = typeof window !== "undefined" && Boolean((window as { __VALVE_TRAY__?: boolean }).__VALVE_TRAY__);
  // The panel fills the tray window (padding: 0); the other screens are normal
  // pages that need a gutter, so they get one back.
  const trayPage = tray && screen !== "home" && screen !== "panel";
  return (
    <div className={`shell${tray ? " shell-tray" : ""}${trayPage ? " shell-tray-page" : ""}`}>
      {tray ? null : (
        <header className="topbar">
          <a className="brand" href="#/">
            Jumpgate
          </a>
          <nav className="nav">
            {NAV.map((n) => (
              <a key={n.nav} href={n.to} className={n.nav === nav ? "active" : undefined} data-nav={n.nav}>
                {n.label}
              </a>
            ))}
          </nav>
        </header>
      )}
      <main id="content" className="content">
        <UpdateBanner />
        <Routes>
          <Route path="/" element={<Panel />} />
          <Route path="/panel" element={<Panel />} />
          <Route path="/rpc" element={<Rpc />} />
          <Route path="/vpn" element={<PrivateAccess />} />
          <Route path="/targets" element={<Targets />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/machine/:id" element={<Machine />} />
          <Route path="/machine" element={<Navigate to="/targets" replace />} />
          <Route path="/security/:id" element={<Security />} />
          <Route path="/security" element={<Navigate to="/targets" replace />} />
          <Route path="/diag/:id" element={<Diagnostics />} />
          <Route path="/diag" element={<Navigate to="/targets" replace />} />
          <Route path="/analytics/:id" element={<Analytics />} />
          <Route path="/analytics" element={<Navigate to="/rpc" replace />} />
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={p} path={`/${p}/:id`} element={<RedirectToMachine />} />
          ))}
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={`${p}-noid`} path={`/${p}`} element={<Navigate to="/targets" replace />} />
          ))}
          <Route path="*" element={<Panel />} />
        </Routes>
      </main>
      {tray ? (
        <nav className="tray-tabs" aria-label="Sections">
          {TRAY_TABS.map((t) => {
            const active = trayActive(screen) === t.key;
            return (
              <a
                key={t.key}
                href={t.to}
                className={active ? "active" : undefined}
                data-nav={t.key}
                aria-current={active ? "page" : undefined}
              >
                {t.label}
              </a>
            );
          })}
        </nav>
      ) : null}
    </div>
  );
}
