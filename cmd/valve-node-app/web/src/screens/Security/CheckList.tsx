import type { CheckItem } from "../../api";
import { CheckItemRow } from "./CheckItemRow";

export function CheckList({ items }: { items: CheckItem[] }) {
  return (
    <ul className="check-list">
      {items.map((item) => (
        <CheckItemRow key={item.ID} item={item} />
      ))}
    </ul>
  );
}
