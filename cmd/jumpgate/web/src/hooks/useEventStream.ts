import { useEffect, useRef } from "react";
import * as api from "../api";

// useEventStream subscribes to a target's setup event stream (api.streamSetup)
// whenever targetId is non-null, and unsubscribes on unmount or when targetId
// changes. onEvent is kept in a ref rather than a useEffect dependency, so a
// caller re-rendering with a new inline callback identity does NOT tear down
// and re-open the EventSource — only a genuine change of targetId does.
//
// This mirrors panel.ts's own streamStop bookkeeping (see provision/runSetup
// there): callers still decide what "finished" means (ev.err ||
// (ev.stepId === "run" && ev.done)) inside their own onEvent — this hook only
// owns the subscribe/unsubscribe lifecycle, not the finish condition.
export function useEventStream(targetId: string | null, onEvent: (ev: api.SetupEvent) => void): void {
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!targetId) return;
    const stop = api.streamSetup(targetId, (ev) => onEventRef.current(ev));
    return () => stop();
  }, [targetId]);
}
