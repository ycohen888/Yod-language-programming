import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";

export type TreeCtxItem =
  | { type: "item"; label: string; action: string; danger?: boolean }
  | { type: "separator" };

type Props = {
  x: number;
  y: number;
  items: TreeCtxItem[];
  onAction: (action: string) => void;
  onClose: () => void;
};

export function TreeContextMenu({ x, y, items, onAction, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let alive = true;
    const onDown = (e: MouseEvent) => {
      // לא סוגרים על אותה לחיצה ימנית שפתחה את התפריט
      if (e.button === 2) return;
      if (!ref.current?.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    const onCtx = (e: Event) => {
      if (!ref.current?.contains(e.target as Node)) {
        e.preventDefault();
        onClose();
      }
    };
    // אחרי פריים — כדי שלא ייסגר מיד מאותו אירוע פתיחה
    const t = window.setTimeout(() => {
      if (!alive) return;
      window.addEventListener("mousedown", onDown, true);
      window.addEventListener("keydown", onKey);
      window.addEventListener("contextmenu", onCtx, true);
    }, 0);
    return () => {
      alive = false;
      window.clearTimeout(t);
      window.removeEventListener("mousedown", onDown, true);
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("contextmenu", onCtx, true);
    };
  }, [onClose]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    let left = x;
    let top = y;
    if (left + rect.width > window.innerWidth - 8) left = window.innerWidth - rect.width - 8;
    if (top + rect.height > window.innerHeight - 8) top = window.innerHeight - rect.height - 8;
    if (left < 8) left = 8;
    if (top < 8) top = 8;
    el.style.left = `${left}px`;
    el.style.top = `${top}px`;
  }, [x, y]);

  return createPortal(
    <div ref={ref} className="tree-ctx-menu" style={{ left: x, top: y }} role="menu">
      {items.map((it, i) =>
        it.type === "separator" ? (
          <div key={`sep-${i}`} className="tree-ctx-sep" />
        ) : (
          <button
            key={it.action + it.label}
            type="button"
            className={`tree-ctx-item${it.danger ? " danger" : ""}`}
            role="menuitem"
            onClick={(e) => {
              e.stopPropagation();
              onClose();
              onAction(it.action);
            }}
          >
            {it.label}
          </button>
        )
      )}
    </div>,
    document.body
  );
}
