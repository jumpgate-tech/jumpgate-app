// The attention strip: the ONE surface a gateway uses to say what is wrong,
// each line carrying the exact command where one exists (the orphan banner's
// `docker rm -f` was the model). It renders a list of derived Attention lines
// (see rpcModel.attentionLines) — never re-deriving them here.
import type { Attention } from "./rpcModel";
import { CopyButton } from "./CopyButton";

export function AttentionLine({ line }: { line: Attention }) {
  return (
    <div className={`strip-line strip-${line.tone}`}>
      <span className="strip-text">{line.text}</span>
      {line.cmd ? (
        <>
          <code className="strip-cmd">{line.cmd}</code>
          <CopyButton value={line.cmd} />
        </>
      ) : null}
    </div>
  );
}

export function AttentionStrip({ lines }: { lines: Attention[] }) {
  if (lines.length === 0) return null;
  return (
    <div className="strip">
      {lines.map((l, i) => (
        <AttentionLine key={i} line={l} />
      ))}
    </div>
  );
}
