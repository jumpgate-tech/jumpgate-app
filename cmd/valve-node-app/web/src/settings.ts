// #/settings — AI provider selection + key, the provider keys that fill the
// ${NAME} slots in RPC URLs, plus the reference RPC base (advanced, rarely
// touched).
import * as api from "./api";
import { dropdown, escapeHtml, footer, onAction, wireDropdowns } from "./ui";

const PROVIDERS: { value: api.AIProvider; label: string }[] = [
  { value: "", label: "None" },
  { value: "gemini", label: "Gemini" },
  { value: "groq", label: "Groq" },
  { value: "ollama", label: "Ollama" },
];

// VALVE_API_KEY gets a row whether or not one is stored: it is the slot the
// app itself always wants filled, and a row that only appears once a key
// exists is a row nobody can use to enter their first one.
const VALVE_KEY = "VALVE_API_KEY";

// providerKeyNote is the one line under a row saying what filling it buys.
// Only VALVE_API_KEY has anything special to say — a key ships for it, so the
// row is optional. Nothing is claimed about that key's limits.
function providerKeyNote(name: string): string {
  if (name === VALVE_KEY) {
    return `Optional — a key ships with the app and is used when this is empty. Enter your own account's key to use that instead.`;
  }
  return `Fills the <code>\${${escapeHtml(name)}}</code> slot wherever an endpoint URL carries one.`;
}

export function renderSettings(root: HTMLElement): () => void {
  let disposed = false;
  let keyTouched = false;
  let saving = false;
  let error: string | null = null;
  let saved = false;
  let current: api.Settings | null = null;
  // The provider the user has picked this session (null = untouched, use the
  // saved value). Held separately from `current` so a change before Save
  // doesn't look persisted.
  let providerChoice: api.AIProvider | null = null;
  // Provider-key rows the operator touched this session, and what they typed.
  // Only touched names are sent — an untouched name must not be sent at all,
  // since the server reads "" as "forget this key" and GET never echoes the
  // value back for us to resend. The typed text is held here rather than left
  // in the DOM because every state change re-renders the form: without it, a
  // save that the server rejects would take the key the operator just typed
  // down with it.
  const providerTouched = new Set<string>();
  const providerTyped = new Map<string, string>();
  // The add-row: a name the operator types in because the slot has no key yet,
  // so there is no row for it to appear in.
  let newKeyName = "";
  let newKeyValue = "";

  root.innerHTML = `<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${footer()}`;
  const body = root.querySelector<HTMLElement>("#settings-body")!;

  onAction(root, (action, target) => {
    if (action === "save") void save();
    if (action === "clear-key") {
      if (!current) return;
      keyTouched = true;
      const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
      if (keyInput) keyInput.value = "";
      render(current);
    }
    if (action === "clear-provider-key") {
      const name = target.dataset.key;
      if (!current || !name) return;
      // Same shape as clear-key: mark it touched and blank the field, so save
      // sends "" for this name and only this name.
      providerTouched.add(name);
      providerTyped.set(name, "");
      saved = false;
      render(current);
    }
  });

  wireDropdowns(root, (id, value) => {
    if (id !== "ai-provider" || !current) return;
    providerChoice = value as api.AIProvider;
    saved = false;
    render(current);
  });

  load();

  async function load(): Promise<void> {
    try {
      const settings = await api.getSettings();
      if (disposed) return;
      current = settings;
      render(settings);
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load settings: ${escapeHtml(String(err))}</p>`;
    }
  }

  // keyNames is every row to draw: the placeholders the server says hold a key,
  // plus VALVE_API_KEY whether or not it does. providerKeysSet is a promised
  // array and still coalesced — a nil slice from an older binary would
  // serialise as JSON null and take the screen down on .filter.
  function keyNames(settings: api.Settings): string[] {
    const set = Array.isArray(settings.providerKeysSet) ? settings.providerKeysSet : [];
    const rest = set.filter((n) => n !== VALVE_KEY).sort();
    return [VALVE_KEY, ...rest];
  }

  function providerKeyRow(name: string, isSet: boolean): string {
    const safe = escapeHtml(name);
    return `
      <div class="pk-row">
        <label>
          <code>${safe}</code>
          <input class="provider-key" data-key="${safe}" type="password" autocomplete="off"
                 placeholder="${isSet ? "•••••••• (leave blank to keep)" : "no key set"}" />
        </label>
        <p class="muted small">${providerKeyNote(name)}</p>
        ${isSet ? `<button class="btn btn-ghost" type="button" data-action="clear-provider-key" data-key="${safe}">Clear saved key</button>` : ""}
      </div>`;
  }

  function render(settings: api.Settings): void {
    const provider = providerChoice ?? settings.aiProvider;
    const set = Array.isArray(settings.providerKeysSet) ? settings.providerKeysSet : [];
    const rows = keyNames(settings)
      .map((name) => providerKeyRow(name, set.includes(name)))
      .join("");

    body.innerHTML = `
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          ${dropdown(
            "ai-provider",
            PROVIDERS.map((p) => ({ value: p.value, label: p.label })),
            provider,
          )}
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${settings.aiKeySet ? "•••••••• (leave blank to keep)" : "no key set"}" autocomplete="off" />
        </label>
        ${settings.aiKeySet ? `<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>` : ""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node-app/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>

        <section class="pk-section">
          <h2>Provider keys</h2>
          <p class="muted small">Some RPC endpoints carry an account key in the URL, which the chain feed
            writes as a slot like <code>\${INFURA_API_KEY}</code>. An endpoint whose slot has no key is
            rejected before it is dialled, naming the slot it needs — fill that slot here and the endpoint
            becomes a candidate again. Stored on this machine only, and never sent back to this page.</p>
          ${rows}
          <div class="pk-row pk-new">
            <label>
              Add a key for another slot
              <input id="pk-new-name" type="text" autocomplete="off" spellcheck="false"
                     placeholder="INFURA_API_KEY" value="${escapeHtml(newKeyName)}" />
            </label>
            <label>
              <span class="muted small">its key</span>
              <input id="pk-new-value" type="password" autocomplete="off" placeholder="no key set" />
            </label>
            <p class="muted small">Use the exact name the rejection quotes. Letters, digits and underscores only.</p>
          </div>
        </section>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${escapeHtml(settings.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${error ? `<p class="error">${escapeHtml(error)}</p>` : ""}
        ${saved ? `<p class="ok">Saved.</p>` : ""}
        <button class="btn btn-primary" type="button" data-action="save" ${saving ? "disabled" : ""}>${saving ? "Saving…" : "Save"}</button>
      </form>
    `;

    const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
    keyInput?.addEventListener("input", () => {
      keyTouched = true;
      saved = false;
    });
    root.querySelector<HTMLInputElement>("#ref-rpc-base")?.addEventListener("input", () => {
      saved = false;
    });

    root.querySelectorAll<HTMLInputElement>("input.provider-key").forEach((el) => {
      const name = el.dataset.key;
      if (!name) return;
      // Put back what was typed before the last re-render. Assigned to the
      // property, not written into the markup, so the value never becomes an
      // attribute in the document.
      const typed = providerTyped.get(name);
      if (typed !== undefined) el.value = typed;
      el.addEventListener("input", () => {
        providerTouched.add(name);
        providerTyped.set(name, el.value);
        saved = false;
      });
    });

    const newValueInput = root.querySelector<HTMLInputElement>("#pk-new-value");
    if (newValueInput) newValueInput.value = newKeyValue;
    newValueInput?.addEventListener("input", () => {
      newKeyValue = newValueInput.value;
      saved = false;
    });
    const newNameInput = root.querySelector<HTMLInputElement>("#pk-new-name");
    newNameInput?.addEventListener("input", () => {
      newKeyName = newNameInput.value;
      saved = false;
    });
  }

  async function save(): Promise<void> {
    const keyInput = root.querySelector<HTMLInputElement>("#ai-key");
    const refRpcInput = root.querySelector<HTMLInputElement>("#ref-rpc-base");
    if (!keyInput || !refRpcInput || !current) return;

    const body: api.PutSettingsRequest = {
      aiProvider: providerChoice ?? current.aiProvider,
      refRpcBase: refRpcInput.value.trim(),
    };
    // Only send aiKey if the user actually touched the field this session —
    // omitting it preserves whatever key is already stored server-side.
    if (keyTouched) {
      body.aiKey = keyInput.value;
    }

    // Same rule for the provider keys, per name: a name the operator did not
    // touch is left out entirely, because sending it empty means "forget it".
    const providerKeys: Record<string, string> = {};
    for (const name of providerTouched) {
      providerKeys[name] = providerTyped.get(name) ?? "";
    }
    // The add-row is a name and a value together; an empty name is just an
    // untouched row. The name is sent as typed — the server is the authority on
    // what a placeholder name may be, and its complaint is the one worth
    // showing, so nothing is silently dropped or rewritten here.
    const typedName = newKeyName.trim();
    if (typedName) providerKeys[typedName] = newKeyValue;
    if (Object.keys(providerKeys).length > 0) body.providerKeys = providerKeys;

    saving = true;
    error = null;
    saved = false;
    render(current);
    try {
      const updated = await api.putSettings(body);
      if (disposed) return;
      current = updated;
      keyTouched = false;
      providerTouched.clear();
      providerTyped.clear();
      newKeyName = "";
      newKeyValue = "";
      saving = false;
      saved = true;
      render(updated);
    } catch (err) {
      if (disposed) return;
      saving = false;
      error = String(err instanceof Error ? err.message : err);
      render(current);
    }
  }

  return () => {
    disposed = true;
  };
}
