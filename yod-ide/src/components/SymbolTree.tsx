import { useMemo, useState } from "react";
import type { ProjectSymbol, SymbolKind } from "../lib/projectIndex";
import { colorForSymbolKind, Icon, iconForSymbolKind } from "./Icon";
import { pathBase } from "../lib/paths";

export type SymbolGroupMode = "kind" | "file" | "flat";

type Props = {
  symbols: ProjectSymbol[];
  groupBy?: SymbolGroupMode;
  emptyText?: string;
  onNavigate: (sym: ProjectSymbol) => void;
  /** הדגשת השורה הנוכחית בעורך */
  activeLine?: number | null;
  activeFile?: string | null;
};

const KIND_ORDER: SymbolKind[] = ["מחלקה", "פונקציה", "קבוע", "משתנה", "פרמטר", "מודול", "יצוא", "קובץ"];

function kindLabel(kind: SymbolKind): string {
  switch (kind) {
    case "מחלקה":
      return "מחלקות";
    case "פונקציה":
      return "פונקציות";
    case "משתנה":
      return "משתנים";
    case "פרמטר":
      return "פרמטרים";
    case "קבוע":
      return "קבועים";
    case "מודול":
      return "מודולים";
    case "יצוא":
      return "יצואים";
    case "קובץ":
      return "קבצים";
    default:
      return kind;
  }
}

export function SymbolTree({
  symbols,
  groupBy = "kind",
  emptyText = "אין סמלים.",
  onNavigate,
  activeLine,
  activeFile,
}: Props) {
  const [filter, setFilter] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return symbols;
    return symbols.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.kind.includes(q) ||
        pathBase(s.file).toLowerCase().includes(q)
    );
  }, [symbols, filter]);

  const groups = useMemo(() => {
    if (groupBy === "flat") {
      return [{ key: "__flat", label: null as string | null, items: filtered }];
    }
    if (groupBy === "file") {
      const map = new Map<string, ProjectSymbol[]>();
      for (const s of filtered) {
        const k = s.file || "(ללא קובץ)";
        const list = map.get(k) ?? [];
        list.push(s);
        map.set(k, list);
      }
      return [...map.entries()]
        .sort(([a], [b]) => pathBase(a).localeCompare(pathBase(b), "he"))
        .map(([key, items]) => ({
          key,
          label: pathBase(key),
          items: items.sort((a, b) => a.line - b.line),
        }));
    }
    // kind
    const map = new Map<SymbolKind, ProjectSymbol[]>();
    for (const s of filtered) {
      const list = map.get(s.kind) ?? [];
      list.push(s);
      map.set(s.kind, list);
    }
    return KIND_ORDER.filter((k) => map.has(k)).map((k) => ({
      key: k,
      label: kindLabel(k),
      items: (map.get(k) ?? []).sort((a, b) => a.line - b.line || a.name.localeCompare(b.name, "he")),
    }));
  }, [filtered, groupBy]);

  return (
    <div className="symbol-tree">
      <div className="symbol-filter">
        <input
          type="search"
          placeholder="סינון…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          dir="rtl"
        />
      </div>
      <div className="symbol-list">
        {groups.length === 0 || filtered.length === 0 ? (
          <div className="tree-empty">{emptyText}</div>
        ) : (
          groups.map((g) => {
            const isOpen = !collapsed[g.key];
            return (
              <div key={g.key} className="symbol-group">
                {g.label ? (
                  <button
                    type="button"
                    className="symbol-group-title"
                    onClick={() => setCollapsed((c) => ({ ...c, [g.key]: !c[g.key] }))}
                  >
                    <Icon name={isOpen ? "expand_more" : "chevron_left"} size={16} />
                    <span>{g.label}</span>
                    <span className="symbol-count">{g.items.length}</span>
                  </button>
                ) : null}
                {isOpen
                  ? g.items.map((s) => {
                      const active =
                        !!activeFile &&
                        s.file === activeFile &&
                        activeLine != null &&
                        s.line === activeLine;
                      return (
                        <button
                          key={`${s.file}:${s.line}:${s.kind}:${s.name}`}
                          type="button"
                          className={`symbol-item${active ? " active" : ""}`}
                          title={`${s.file}:${s.line}`}
                          onClick={() => onNavigate(s)}
                        >
                          <Icon
                            name={iconForSymbolKind(s.kind)}
                            size={16}
                            className="symbol-kind-icon"
                            style={{ color: colorForSymbolKind(s.kind) }}
                          />
                          <span className="symbol-name">{s.name}</span>
                          {groupBy !== "file" ? (
                            <span className="symbol-meta">{pathBase(s.file)}:{s.line}</span>
                          ) : (
                            <span className="symbol-meta">:{s.line}</span>
                          )}
                        </button>
                      );
                    })
                  : null}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
