import { useEffect, useMemo, useRef, useState } from "react";
import { getSymbolsForFile, type ProjectSymbol } from "../lib/projectIndex";
import { colorForSymbolKind, Icon, iconForSymbolKind } from "./Icon";
import { pathBase } from "../lib/paths";

type Props = {
  root: string | null;
  /** נתיב הקובץ הפעיל (null לקובץ ללא-שם) */
  path: string | null;
  /** כותרת הטאב (לקובץ ללא-שם) */
  title: string | null;
  /** מפתח הטאב (מקור לסמלים חיים) */
  tabKey: string | null;
  cursorLine: number;
  /** משתנה בכל שינוי בסמלים כדי לרענן */
  refreshKey: number;
  onJumpSymbol: (s: ProjectSymbol) => void;
};

/** מפרק נתיב מלא למקטעים יחסית לשורש הפרויקט. */
function relativeSegments(root: string | null, path: string): string[] {
  const norm = (s: string) => s.replace(/[\\/]+$/, "");
  const full = norm(path);
  if (root) {
    const r = norm(root);
    if (full.toLowerCase().startsWith(r.toLowerCase())) {
      const rel = full.slice(r.length).replace(/^[\\/]+/, "");
      return rel ? rel.split(/[\\/]+/) : [pathBase(full)];
    }
  }
  return full.split(/[\\/]+/).filter(Boolean);
}

export function Breadcrumbs({
  root,
  path,
  title,
  tabKey,
  cursorLine,
  refreshKey,
  onJumpSymbol,
}: Props) {
  const [symbolMenuOpen, setSymbolMenuOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  const symbols = useMemo(
    () => (path ? getSymbolsForFile(path, tabKey) : []),
    // refreshKey מכריח חישוב מחדש כשמשתנים סמלים חיים
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [path, tabKey, refreshKey]
  );

  const currentSymbol = useMemo(() => {
    let best: ProjectSymbol | null = null;
    for (const s of symbols) {
      if (s.line <= cursorLine && (!best || s.line > best.line)) best = s;
    }
    return best;
  }, [symbols, cursorLine]);

  useEffect(() => {
    if (!symbolMenuOpen) return;
    const onDown = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setSymbolMenuOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setSymbolMenuOpen(false);
    };
    window.addEventListener("mousedown", onDown);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onDown);
      window.removeEventListener("keydown", onKey);
    };
  }, [symbolMenuOpen]);

  const segments = path ? relativeSegments(root, path) : title ? [title] : [];
  if (segments.length === 0) return null;

  return (
    <div className="breadcrumbs" dir="rtl" ref={wrapRef}>
      {segments.map((seg, i) => (
        <span className="crumb" key={`${seg}-${i}`}>
          {i > 0 ? <Icon name="chevron_left" size={14} className="crumb-sep" /> : null}
          <span className={i === segments.length - 1 ? "crumb-file" : "crumb-dir"}>{seg}</span>
        </span>
      ))}

      {currentSymbol ? (
        <span className="crumb crumb-symbol-wrap">
          <Icon name="chevron_left" size={14} className="crumb-sep" />
          <button
            type="button"
            className="crumb-symbol"
            onClick={() => setSymbolMenuOpen((o) => !o)}
            title="קפיצה לסמל בקובץ"
          >
            <Icon
              name={iconForSymbolKind(currentSymbol.kind)}
              size={14}
              style={{ color: colorForSymbolKind(currentSymbol.kind) }}
            />
            <span>{currentSymbol.name}</span>
          </button>
        </span>
      ) : null}

      {symbolMenuOpen ? (
        <div className="crumb-menu" role="menu">
          {symbols.length === 0 ? (
            <div className="crumb-menu-empty">אין סמלים בקובץ</div>
          ) : (
            symbols.map((s) => (
              <button
                key={`${s.file}:${s.line}:${s.kind}:${s.name}`}
                type="button"
                className={`crumb-menu-item${
                  currentSymbol && s.line === currentSymbol.line && s.name === currentSymbol.name
                    ? " active"
                    : ""
                }`}
                onClick={() => {
                  setSymbolMenuOpen(false);
                  onJumpSymbol(s);
                }}
              >
                <Icon
                  name={iconForSymbolKind(s.kind)}
                  size={14}
                  style={{ color: colorForSymbolKind(s.kind) }}
                />
                <span className="crumb-menu-name">{s.name}</span>
                <span className="crumb-menu-line">:{s.line}</span>
              </button>
            ))
          )}
        </div>
      ) : null}
    </div>
  );
}
