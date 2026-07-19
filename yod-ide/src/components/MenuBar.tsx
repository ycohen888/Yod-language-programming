import { useEffect, useRef, useState } from "react";
import { APP_MENU, type MenuEntry } from "../lib/menuConfig";

type Props = {
  onAction: (action: string) => void;
  /** פעולות שצריך לכבות (למשל פתח מיקום EXE בלי dist_exe). */
  disabledActions?: ReadonlySet<string>;
};

export function MenuBar({ onAction, disabledActions }: Props) {
  const [openId, setOpenId] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!openId) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpenId(null);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenId(null);
    };
    window.addEventListener("mousedown", onDown);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onDown);
      window.removeEventListener("keydown", onKey);
    };
  }, [openId]);

  const run = (entry: MenuEntry) => {
    if (entry.type === "separator") return;
    if (disabledActions?.has(entry.action)) return;
    setOpenId(null);
    onAction(entry.action);
  };

  return (
    <div className="menubar" ref={rootRef} role="menubar">
      {APP_MENU.map((group) => (
        <div key={group.id} className={`menubar-item${openId === group.id ? " open" : ""}`}>
          <button
            type="button"
            className="menubar-trigger"
            role="menuitem"
            aria-haspopup="true"
            aria-expanded={openId === group.id}
            onClick={() => setOpenId((id) => (id === group.id ? null : group.id))}
            onMouseEnter={() => {
              if (openId) setOpenId(group.id);
            }}
          >
            {group.label}
          </button>
          {openId === group.id ? (
            <div className="menubar-dropdown" role="menu">
              {group.items.map((entry, i) =>
                entry.type === "separator" ? (
                  <div key={`sep-${i}`} className="menubar-sep" />
                ) : (
                  <button
                    key={entry.action + entry.label}
                    type="button"
                    className="menubar-option"
                    role="menuitem"
                    disabled={!!disabledActions?.has(entry.action)}
                    onClick={() => run(entry)}
                  >
                    <span className="menubar-option-label">{entry.label}</span>
                    {entry.kb ? <span className="menubar-option-kb">{entry.kb}</span> : null}
                  </button>
                )
              )}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}
