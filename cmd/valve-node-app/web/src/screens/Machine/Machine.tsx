// #/machine/:id — one page per machine. The four routes that used to be one
// screen each (#/setup, #/dash, #/logs, #/services) are SECTIONS of this
// page: a one-line status that expands to the full detail on click. No tabs.
//
// Port of machine.ts's renderMachine. This is deliberately subtraction, not a
// rewrite: each section's detail IS the already-converted React section
// component (SetupWizard / DashboardSection / LogsSection / ServicesSection),
// mounted the first time the operator expands it. Those components own their
// own SSE stream/timers (via their hooks); React's own mount/unmount runs
// their cleanup, so navigating away from this page closes every open stream.
//
// LAZY-MOUNT / KEEP-MOUNTED — mirrors machine.ts exactly. A section is
// mounted (its component enters the tree, so its stream starts) only the
// first time it is expanded. Once mounted it STAYS mounted: collapsing a
// section only hides its body (the React equivalent of machine.ts toggling
// `bodyEl.hidden`), so its stream keeps running and its scroll/log state
// survives a collapse. A section never opened has no component in the tree
// and therefore no stream — the whole point of the lazy mount.
//
// The reused section components still render their own <h1>/<Footer> inside a
// section, exactly as the legacy screens did inside machine.ts's section
// bodies; stripping those so the page reads as one document is the follow-up
// machine.ts's own comment calls out, not this increment.
import { useState, type ReactNode } from "react";
import { Navigate, useParams } from "react-router-dom";
import type * as api from "../../api";
import { useCatalog, useTargets } from "../../hooks/target";
import { Badge } from "../../components/Badge";
import { Footer } from "../../components/Footer";
import { SetupWizard } from "./SetupWizard";
import { DashboardSection } from "./DashboardSection";
import { LogsSection } from "./LogsSection";
import { ServicesSection } from "./ServicesSection";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

interface SectionDef {
  key: string;
  title: string;
  // status is the one-line summary shown on the collapsed row. BEST-EFFORT
  // from data already in hand (the target's wire config + the catalog) —
  // never a deep measured verdict, matching machine.ts's own caveat.
  status: (t: api.Target, catalog: api.Catalog) => ReactNode;
  Component: (props: { targetId: string }) => ReactNode;
}

// SECTIONS is the collapse itself: the four former routes, in the order the
// direction doc puts them (set up, then watch, then read, then the devnet a
// machine can always host).
const SECTIONS: SectionDef[] = [
  {
    key: "setup",
    title: "Setup",
    status: (t) =>
      t.wire ? <Badge text="set up" kind="ok" /> : <Badge text="not set up" kind="neutral" />,
    Component: SetupWizard,
  },
  {
    key: "dashboard",
    title: "Dashboard",
    status: (t) => (
      <span className="muted small">
        {t.wire
          ? "sync, peers, storage and endpoints — live"
          : "available once this machine is set up"}
      </span>
    ),
    Component: DashboardSection,
  },
  {
    key: "logs",
    title: "Logs",
    status: (t) => (
      <span className="muted small">
        {t.wire ? "live tail and error feed" : "available once this machine is set up"}
      </span>
    ),
    Component: LogsSection,
  },
  {
    key: "services",
    title: "Devnet",
    // Offered on EVERY machine, set up or not — a devnet is a container, so
    // unlike a node it needs no Linux host and no root.
    status: () => (
      <span className="muted small">throwaway chain — always available on this machine</span>
    ),
    Component: ServicesSection,
  },
];

export function Machine() {
  const { id } = useParams<{ id: string }>();
  const targetId = id ?? "";

  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const initLoading = targetsQuery.isLoading || catalogQuery.isLoading;
  const initErr = targetsQuery.error ?? catalogQuery.error;

  const target = targetsQuery.data?.find((t) => t.id === targetId);

  // `opened` tracks which sections have EVER been expanded (→ mounted, → their
  // stream started) and `expanded` which are currently visible. A section
  // leaves `expanded` on collapse but never leaves `opened`, so its component
  // stays in the tree — the keep-mounted half of machine.ts's behavior.
  const [opened, setOpened] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  function toggle(key: string): void {
    const willOpen = !expanded[key];
    setExpanded((e) => ({ ...e, [key]: willOpen }));
    if (willOpen) setOpened((o) => ({ ...o, [key]: true }));
  }

  return (
    <>
      <h1>{targetId}</h1>
      <div>
        {initLoading ? (
          <p className="muted">Loading…</p>
        ) : initErr ? (
          <p className="error">Failed to load machine: {formatError(initErr)}</p>
        ) : !target ? (
          // Not a machine we know — the id in the hash is stale (removed, or a
          // bad deep link). Bounce to the list rather than render an empty
          // page, mirroring machine.ts's `location.hash = "#/targets"`.
          <Navigate to="/targets" replace />
        ) : (
          <MachineShell
            target={target}
            catalog={catalogQuery.data!}
            targetId={targetId}
            opened={opened}
            expanded={expanded}
            onToggle={toggle}
          />
        )}
      </div>
    </>
  );
}

function MachineShell({
  target,
  catalog,
  targetId,
  opened,
  expanded,
  onToggle,
}: {
  target: api.Target;
  catalog: api.Catalog;
  targetId: string;
  opened: Record<string, boolean>;
  expanded: Record<string, boolean>;
  onToggle: (key: string) => void;
}) {
  const modeLabel = target.mode === "local" ? "this machine" : "SSH";
  const location =
    target.mode === "ssh" && target.ssh ? `${target.ssh.User}@${target.ssh.Host}` : modeLabel;

  return (
    <>
      <p className="muted">{location}</p>
      <p>
        <HeaderStatus target={target} catalog={catalog} />
      </p>
      <div className="machine-sections">
        {SECTIONS.map((s) => (
          <SectionRow
            key={s.key}
            def={s}
            target={target}
            catalog={catalog}
            targetId={targetId}
            mounted={!!opened[s.key]}
            open={!!expanded[s.key]}
            onToggle={() => onToggle(s.key)}
          />
        ))}
      </div>
      <Footer />
    </>
  );
}

// HeaderStatus mirrors targets.ts's card status line and machine.ts's
// headerStatus: the network + client badges once set up, or "not set up"
// before. Same best-effort caveat — it is the wire config, not live health.
function HeaderStatus({ target, catalog }: { target: api.Target; catalog: api.Catalog }) {
  const wire = target.wire;
  if (!wire) return <Badge text="not set up" kind="neutral" />;
  const net = catalog.networks.find((n) => n.ChainID === wire.ChainID);
  const netName = net ? net.Name : `chain ${wire.ChainID}`;
  return (
    <>
      <Badge text={netName} kind="ok" /> <Badge text={wire.ExecID} kind="neutral" />{" "}
      <Badge text={wire.BeaconID} kind="neutral" />
      {wire.Archive && (
        <>
          {" "}
          <Badge text="archive" kind="warn" />
        </>
      )}
    </>
  );
}

function SectionRow({
  def,
  target,
  catalog,
  targetId,
  mounted,
  open,
  onToggle,
}: {
  def: SectionDef;
  target: api.Target;
  catalog: api.Catalog;
  targetId: string;
  mounted: boolean;
  open: boolean;
  onToggle: () => void;
}) {
  const { Component } = def;
  return (
    <section className={open ? "card machine-section open" : "card machine-section"} data-section-card={def.key}>
      <button
        type="button"
        className="machine-section-head"
        onClick={onToggle}
        aria-expanded={open}
      >
        <span className="machine-section-title">{def.title}</span>
        <span className="machine-section-status">{def.status(target, catalog)}</span>
        <span className="machine-section-caret" aria-hidden="true">
          ▸
        </span>
      </button>
      <div className="machine-section-body" data-section-body={def.key} hidden={!open}>
        {mounted && <Component targetId={targetId} />}
      </div>
    </section>
  );
}
