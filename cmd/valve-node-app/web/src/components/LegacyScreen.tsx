import { useEffect, useRef } from "react";

export type LegacyRender = (root: HTMLElement) => (() => void) | void;

// LegacyScreen mounts a not-yet-converted vanilla screen (its render(root):
// cleanup function) into a React-owned div, mirroring main.ts's old mount():
// call render on mount, run the returned cleanup on unmount, then clear the
// node. `render` MUST be stable (useCallback keyed on its inputs) so the
// effect doesn't re-run every parent render.
export function LegacyScreen({ render }: { render: LegacyRender }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current!;
    const cleanup = render(el);
    return () => {
      try {
        cleanup?.();
      } catch {
        // A screen's cleanup throwing must not block navigating away.
      }
      el.replaceChildren();
    };
  }, [render]);
  return <div ref={ref} />;
}
