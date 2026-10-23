// A gateway's module cards (gateway placement, e.g. private send): the same
// generic card the Machine screen renders, from modulesModel. Whether a module
// applies comes only from /api/catalog's Applies rows, across the chains this
// gateway serves. A toggle saves and re-provisions the gateway; progress is
// the placement machine's setup stream.
import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as api from "../../api";
import type { GatewayView } from "../../api";
import { useCatalog } from "../../hooks/target";
import { ModuleCardView } from "../Machine/ModulesSection";
import { gatewayModuleCards } from "../Machine/modulesModel";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function GatewayModules({ gw }: { gw: GatewayView }) {
  const qc = useQueryClient();
  const catalogQuery = useCatalog();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const streamRef = useRef<(() => void) | null>(null);

  useEffect(() => () => streamRef.current?.(), []);

  if (!catalogQuery.data) return null;
  const cards = gatewayModuleCards(
    catalogQuery.data,
    (gw.networks ?? []).map((n) => n.chainId),
    (gw.config.Modules ?? []).map((m) => m.id),
  ).filter((c) => c.state !== "hidden");
  if (cards.length === 0) return null;

  async function toggle(id: string, enabled: boolean): Promise<void> {
    setBusy(true);
    setErr(null);
    setLines([]);
    try {
      const res = await api.setGatewayModule(gw.id, id, enabled);
      streamRef.current?.();
      streamRef.current = api.streamSetup(res.targetId, (ev) => {
        const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : "";
        if (line) setLines((prev) => [...prev, line].slice(-5));
        if (ev.err || ev.done) void qc.invalidateQueries({ queryKey: ["gateways"] });
      });
    } catch (e) {
      setErr(message(e));
    } finally {
      setBusy(false);
      void qc.invalidateQueries({ queryKey: ["gateways"] });
    }
  }

  return (
    <div className="gateway-modules">
      <h3>Modules</h3>
      {err && <p className="error">{err}</p>}
      {cards.map((c) => (
        <ModuleCardView key={c.id} card={c} busy={busy} onToggle={() => void toggle(c.id, c.state !== "on")} />
      ))}
      {lines.length > 0 && <pre className="small">{lines.join("\n")}</pre>}
    </div>
  );
}
