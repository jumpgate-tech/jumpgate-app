// One provider-key row: the ${NAME} slot's own password field, a note on
// what filling it buys, and — only once the server says a key is actually
// stored for it — a "Clear saved key" button. Mirrors settings.ts's
// providerKeyRow(); VALVE_API_KEY is the one name with something special to
// say, since a key ships with the app for it and the row is optional.
import { VALVE_KEY } from "./settingsModel";

export function ProviderKeyRow({
  name,
  isSet,
  value,
  onChange,
  onClear,
}: {
  name: string;
  isSet: boolean;
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
}) {
  return (
    <div className="pk-row">
      <label>
        <code>{name}</code>
        <input
          className="provider-key"
          type="password"
          autoComplete="off"
          placeholder={isSet ? "•••••••• (leave blank to keep)" : "no key set"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      </label>
      <p className="muted small">
        {name === VALVE_KEY ? (
          <>
            Optional — a key ships with the app and is used when this is empty. Enter your own account&apos;s
            key to use that instead.
          </>
        ) : (
          <>
            Fills the <code>{`\${${name}}`}</code> slot wherever an endpoint URL carries one.
          </>
        )}
      </p>
      {isSet && (
        <button className="btn btn-ghost" type="button" onClick={onClear}>
          Clear saved key
        </button>
      )}
    </div>
  );
}
