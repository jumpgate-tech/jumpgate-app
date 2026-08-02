import { useCallback } from "react";
import { Navigate, Route, Routes, useLocation, useParams } from "react-router-dom";
import { LegacyScreen } from "./components/LegacyScreen";
import { renderPanel } from "./panel";
import { renderRPC } from "./rpc";
import { renderTargets } from "./targets";
import { renderSettings } from "./settings";
import { renderMachine } from "./machine";
import { renderSecurity } from "./security";
import { renderDiagnostics } from "./diag";
import { renderAnalytics } from "./analytics";

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
function LegacyPanel() {
  return <Bridge render={useCallback((el: HTMLElement) => renderPanel(el), [])} />;
}
function LegacyRPC() {
  return <Bridge render={useCallback((el: HTMLElement) => renderRPC(el), [])} />;
}
function LegacyTargets() {
  return <Bridge render={useCallback((el: HTMLElement) => renderTargets(el), [])} />;
}
function LegacySettings() {
  return <Bridge render={useCallback((el: HTMLElement) => renderSettings(el), [])} />;
}
function LegacyMachine() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderMachine(el, id!), [id])} />;
}
function LegacySecurity() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderSecurity(el, id!), [id])} />;
}
function LegacyDiag() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderDiagnostics(el, id!), [id])} />;
}
function LegacyAnalytics() {
  const { id } = useParams();
  return <Bridge render={useCallback((el: HTMLElement) => renderAnalytics(el, id!), [id])} />;
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
          <Route path="/" element={<LegacyPanel />} />
          <Route path="/panel" element={<LegacyPanel />} />
          <Route path="/rpc" element={<LegacyRPC />} />
          <Route path="/targets" element={<LegacyTargets />} />
          <Route path="/settings" element={<LegacySettings />} />
          <Route path="/machine/:id" element={<LegacyMachine />} />
          <Route path="/machine" element={<Navigate to="/targets" replace />} />
          <Route path="/security/:id" element={<LegacySecurity />} />
          <Route path="/diag/:id" element={<LegacyDiag />} />
          <Route path="/analytics/:id" element={<LegacyAnalytics />} />
          <Route path="/analytics" element={<Navigate to="/rpc" replace />} />
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={p} path={`/${p}/:id`} element={<RedirectToMachine />} />
          ))}
          {["setup", "dash", "logs", "services"].map((p) => (
            <Route key={`${p}-noid`} path={`/${p}`} element={<Navigate to="/targets" replace />} />
          ))}
          <Route path="*" element={<LegacyPanel />} />
        </Routes>
      </main>
    </div>
  );
}
