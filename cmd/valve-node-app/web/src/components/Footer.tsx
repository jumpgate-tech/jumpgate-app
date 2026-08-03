// The mandatory "learn how this works" link every screen carries, plus an
// optional per-context deep link. ui.ts's old footer() string-template
// helper was retired once every screen finished its React migration; this
// component is the sole remaining implementation.
import { LEARN_ROOT } from "../ui";

export function Footer({ contextLabel, contextUrl }: { contextLabel?: string; contextUrl?: string }) {
  // Only render the context link when it points somewhere other than the
  // base learn link — otherwise it's a pointless duplicate (every network's
  // LearnURL currently equals LEARN_ROOT).
  const showContext = !!contextLabel && !!contextUrl && contextUrl !== LEARN_ROOT;
  return (
    <footer className="footer">
      <a href={LEARN_ROOT} target="_blank" rel="noopener noreferrer">
        Learn how this works → learn.valve.city/rpc
      </a>
      {showContext && (
        <>
          <span className="footer-sep">·</span>
          <a href={contextUrl} target="_blank" rel="noopener noreferrer">
            {contextLabel}
          </a>
        </>
      )}
    </footer>
  );
}
