// style: the few attributes the dashboard uses. A selected line starts with
// ">" (so it reads without color too) and is drawn in reverse video.
import { reverse, StyledText, type TextChunk } from "@opentui/core";

export const FOCUS_COLOR = "#5fd7ff";
export const DIM_COLOR = "#808080";

/** isSelected: the line carries the selection marker. */
export const isSelected = (line: string) => line.startsWith(">");

const plain = (text: string): TextChunk => ({ __isChunk: true, text });

/** styled draws lines as one text, the selected ones reversed across the width. */
export function styled(lines: string[], width: number): StyledText {
  const chunks: TextChunk[] = [];
  lines.forEach((l, i) => {
    if (i) chunks.push(plain("\n"));
    chunks.push(isSelected(l) ? reverse(l.padEnd(width)) : plain(l));
  });
  return new StyledText(chunks);
}
