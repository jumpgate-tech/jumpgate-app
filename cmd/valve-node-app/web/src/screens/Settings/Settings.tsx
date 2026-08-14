// #/settings — AI provider selection + key, the provider keys that fill the
// ${NAME} slots in RPC URLs, plus the reference RPC base (advanced, rarely
// touched). Port of settings.ts.
//
// Every field is a controlled input backed by useState rather than read from
// the DOM at save time, which settings.ts's own provider-key rows already did
// (typed text held in `providerTyped` so a rejected save wouldn't take it
// down with the next render) but its single ai-key and ref-rpc-base fields
// did not: those two were read straight off the DOM node, so an UNRELATED
// re-render — e.g. picking a different AI provider — silently blanked
// whatever had just been typed into them while leaving `keyTouched` set,
// meaning a save right after would send an empty key. Controlled inputs
// close that gap by construction; both fields now persist the same way the
// provider-key rows always did.
import { useState } from "react";
import type * as api from "../../api";
import { Footer } from "../../components/Footer";
import { useSettings, usePutSettings } from "../../hooks/settings";
import { keyNames } from "./settingsModel";
import { ProviderKeyRow } from "./ProviderKeyRow";

const PROVIDERS: { value: api.AIProvider; label: string }[] = [
  { value: "", label: "None" },
  { value: "gemini", label: "Gemini" },
  { value: "groq", label: "Groq" },
  { value: "ollama", label: "Ollama" },
];

export function Settings() {
  const settingsQuery = useSettings();
  const putSettings = usePutSettings();
  const current = settingsQuery.data;
  // providerKeysSet NAMES the ${...} placeholders that have a key stored —
  // never the values. Coalesced to [] in case an older binary behind a newer
  // bundle sends the field as JSON null.
  const providerKeysSet = Array.isArray(current?.providerKeysSet) ? current.providerKeysSet : [];

  // null = untouched this session; fall back to the loaded/saved value.
  const [providerChoice, setProviderChoice] = useState<api.AIProvider | null>(null);
  const [refRpcBase, setRefRpcBase] = useState<string | null>(null);
  // null = untouched this session; fall back to the loaded value. A missing
  // updateCheckEnabled (older binary) reads as "on", matching the server.
  const [updateCheckEnabled, setUpdateCheckEnabled] = useState<boolean | null>(null);

  const [aiKeyValue, setAiKeyValue] = useState("");
  const [keyTouched, setKeyTouched] = useState(false);

  // Provider-key rows touched this session, and what was typed into them.
  // Only touched names are ever sent — the server reads "" as "forget this
  // key", and GET never echoes a stored value back for us to resend.
  const [providerTouched, setProviderTouched] = useState<Set<string>>(new Set());
  const [providerTyped, setProviderTyped] = useState<Map<string, string>>(new Map());

  // The add-row: a name typed in for a slot with no key yet, so no row for
  // it exists to appear in.
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyValue, setNewKeyValue] = useState("");

  const [saved, setSaved] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  function touchProviderKey(name: string, value: string) {
    setProviderTouched((prev) => new Set(prev).add(name));
    setProviderTyped((prev) => new Map(prev).set(name, value));
    setSaved(false);
  }

  async function handleSave() {
    if (!current) return;
    const body: api.PutSettingsRequest = {
      aiProvider: providerChoice ?? current.aiProvider,
      refRpcBase: (refRpcBase ?? current.refRpcBase).trim(),
    };
    // Only send the update toggle when it was touched this session, so a save
    // for an unrelated field leaves the stored value alone.
    if (updateCheckEnabled !== null) body.updateCheckEnabled = updateCheckEnabled;
    // Only send aiKey if the user actually touched the field this session —
    // omitting it preserves whatever key is already stored server-side.
    if (keyTouched) body.aiKey = aiKeyValue;

    // Same rule per provider-key name: an untouched name is left out
    // entirely, because sending it empty means "forget it".
    const providerKeys: Record<string, string> = {};
    for (const name of providerTouched) providerKeys[name] = providerTyped.get(name) ?? "";
    // The add-row is a name and a value together; an empty name is just an
    // untouched row. The name is sent as typed — the server is the authority
    // on what a placeholder name may be, and its complaint is the one worth
    // showing, so nothing is silently dropped or rewritten here.
    const typedName = newKeyName.trim();
    if (typedName) providerKeys[typedName] = newKeyValue;
    if (Object.keys(providerKeys).length > 0) body.providerKeys = providerKeys;

    setSaveError(null);
    setSaved(false);
    try {
      await putSettings.mutateAsync(body);
      setKeyTouched(false);
      setAiKeyValue("");
      setProviderTouched(new Set());
      setProviderTyped(new Map());
      setNewKeyName("");
      setNewKeyValue("");
      setRefRpcBase(null);
      setUpdateCheckEnabled(null);
      setSaved(true);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <>
      <h1>Settings</h1>
      <div>
        {settingsQuery.isLoading ? (
          <p className="muted">Loading…</p>
        ) : settingsQuery.isError ? (
          <p className="error">Failed to load settings: {String(settingsQuery.error)}</p>
        ) : current ? (
          <form className="card" onSubmit={(e) => e.preventDefault()}>
            <label>
              AI provider
              <select
                value={providerChoice ?? current.aiProvider}
                onChange={(e) => {
                  setProviderChoice(e.target.value as api.AIProvider);
                  setSaved(false);
                }}
              >
                {PROVIDERS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              API key
              <input
                type="password"
                autoComplete="off"
                placeholder={current.aiKeySet ? "•••••••• (leave blank to keep)" : "no key set"}
                value={aiKeyValue}
                onChange={(e) => {
                  setAiKeyValue(e.target.value);
                  setKeyTouched(true);
                  setSaved(false);
                }}
              />
            </label>
            {current.aiKeySet && (
              <button
                className="btn btn-ghost"
                type="button"
                onClick={() => {
                  setAiKeyValue("");
                  setKeyTouched(true);
                  setSaved(false);
                }}
              >
                Clear saved key
              </button>
            )}
            <p className="muted small">
              Keys stay on this machine — they&apos;re written to ~/.valve-node-app/config.json (mode 0600)
              and only sent to the provider you pick, never anywhere else.
            </p>

            <section className="pk-section">
              <h2>Provider keys</h2>
              <p className="muted small">
                Some RPC endpoints carry an account key in the URL, which the chain feed writes as a slot
                like <code>{"${INFURA_API_KEY}"}</code>. An endpoint whose slot has no key is rejected before
                it is dialled, naming the slot it needs — fill that slot here and the endpoint becomes a
                candidate again. Stored on this machine only, and never sent back to this page.
              </p>
              {keyNames(current.providerKeysSet).map((name) => (
                <ProviderKeyRow
                  key={name}
                  name={name}
                  isSet={providerKeysSet.includes(name)}
                  value={providerTyped.get(name) ?? ""}
                  onChange={(value) => touchProviderKey(name, value)}
                  onClear={() => touchProviderKey(name, "")}
                />
              ))}
              <div className="pk-row pk-new">
                <label>
                  Add a key for another slot
                  <input
                    type="text"
                    autoComplete="off"
                    spellCheck={false}
                    placeholder="INFURA_API_KEY"
                    value={newKeyName}
                    onChange={(e) => {
                      setNewKeyName(e.target.value);
                      setSaved(false);
                    }}
                  />
                </label>
                <label>
                  <span className="muted small">its key</span>
                  <input
                    type="password"
                    autoComplete="off"
                    placeholder="no key set"
                    value={newKeyValue}
                    onChange={(e) => {
                      setNewKeyValue(e.target.value);
                      setSaved(false);
                    }}
                  />
                </label>
                <p className="muted small">Use the exact name the rejection quotes. Letters, digits and underscores only.</p>
              </div>
            </section>

            <section className="pk-section">
              <h2>Updates</h2>
              <label className="check">
                <input
                  type="checkbox"
                  checked={updateCheckEnabled ?? current.updateCheckEnabled ?? true}
                  onChange={(e) => {
                    setUpdateCheckEnabled(e.target.checked);
                    setSaved(false);
                  }}
                />
                Check for updates automatically
              </label>
              <p className="muted small">
                When on, the app asks GitHub once every few hours whether a newer release exists and shows a
                notice if so. It never installs anything on its own. Turn it off to stop the app reaching
                GitHub.
              </p>
            </section>

            <details className="advanced">
              <summary>Advanced</summary>
              <label>
                Reference RPC base
                <input
                  type="text"
                  value={refRpcBase ?? current.refRpcBase}
                  onChange={(e) => {
                    setRefRpcBase(e.target.value);
                    setSaved(false);
                  }}
                />
              </label>
              <p className="muted small">
                Used to compute head-lag on the dashboard. Leave the default unless you have your own
                reference endpoint.
              </p>
            </details>
            {saveError && <p className="error">{saveError}</p>}
            {saved && <p className="ok">Saved.</p>}
            <button
              className="btn btn-primary"
              type="button"
              disabled={putSettings.isPending}
              onClick={() => void handleSave()}
            >
              {putSettings.isPending ? "Saving…" : "Save"}
            </button>
          </form>
        ) : null}
      </div>
      <Footer />
    </>
  );
}
