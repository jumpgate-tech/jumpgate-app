import { useCallback } from "react";
import { Navigate, Route, Routes, useLocation, useParams } from "react-router-dom";
import { LegacyScreen } from "./components/LegacyScreen";
import { Panel } from "./screens/Panel/Panel";
import { Security } from "./screens/Security/Security";
import { Diagnostics } from "./screens/Diagnostics/Diagnostics";
import { Settings } from "./screens/Settings/Settings";
import { Targets } from "./screens/Targets/Targets";
import { Analytics } from "./screens/Analytics/Analytics";
import { Machine } from "./screens/Machine/Machine";
import { renderRPC } from "./rpc";

const NAV = [
  { to: "#/rpc", nav: "rpc", label: "RPC" },
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

// Bridge wrappers: memoize the legacy render so LegacyScreen's effect is stable.
function Bridge({ render }: { render: (el: HTMLElement) => (() => void) | void }) {
  return <LegacyScreen render={render} />;
}
function LegacyRPC() {
  return <Bridge render={useCallback((el: HTMLElement) => renderRPC(el), [])} />;
}
function RedirectToMachine() {
  const { id } = useParams();
  return <Navigate to={`/machine/${encodeURIComponent(id!)}`} replace />;
}

export function App() {
  const { pathname } = useLocation();
  const screen = pathname.split("/").filter(Boolean)[0] ?? "home";
  const nav = activeNav(screen);
  return (
    <div className="shell">
      <header className="topbar">
        <a className="brand" href="#/">
          valve-node-app
        </a>
        <nav className="nav">
          {NAV.map((n) => (
            <a key={n.nav} href={n.to} className={n.nav === nav ? "active" : undefined} data-nav={n.nav}>
              {n.label}
            </a>
          ))}
        </nav>
      </header>
      <main id="content" className="content">
        <Routes>
          <Route path="/" element={<Panel />} />
          <Route path="/panel" element={<Panel />} />
          <Route path="/rpc" element={<LegacyRPC />} />
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
    </div>
  );
}
