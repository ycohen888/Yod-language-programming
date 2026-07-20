import type { CSSProperties, ReactNode } from "react";
import type { SymbolKind } from "../lib/projectIndex";

/** שמות ligature של Material Icons (מתוך MaterialIcons-Regular). */
export type MaterialIconName =
  | "folder"
  | "folder_open"
  | "insert_drive_file"
  | "note_add"
  | "create_new_folder"
  | "close"
  | "search"
  | "terminal"
  | "play_arrow"
  | "stop"
  | "play_circle"
  | "spellcheck"
  | "memory"
  | "save"
  | "error_outline"
  | "expand_more"
  | "chevron_left"
  | "code"
  | "description"
  | "settings"
  | "refresh"
  | "inventory_2"
  | "category"
  | "functions"
  | "label"
  | "lock"
  | "view_module"
  | "upload"
  | "list_alt"
  | "account_tree"
  | "push_pin"
  | "delete"
  | "add"
  | "remove"
  | "widgets"
  | "minimize"
  | "crop_square"
  | "filter_none";

export function iconForSymbolKind(kind: SymbolKind): MaterialIconName {
  switch (kind) {
    case "מחלקה":
      return "category";
    case "פונקציה":
      return "functions";
    case "משתנה":
    case "פרמטר":
      return "label";
    case "קבוע":
      return "lock";
    case "מודול":
      return "view_module";
    case "יצוא":
      return "upload";
    case "קובץ":
      return "code";
    default:
      return "widgets";
  }
}

export function colorForSymbolKind(kind: SymbolKind): string {
  switch (kind) {
    case "מחלקה":
      return "#4ec9b0";
    case "פונקציה":
      return "#dcdcaa";
    case "משתנה":
    case "פרמטר":
      return "#9cdcfe";
    case "קבוע":
      return "#4fc1ff";
    case "מודול":
      return "#c586c0";
    case "יצוא":
      return "#ce9178";
    default:
      return "#cccccc";
  }
}

/**
 * חלק מהאייקונים החדשים (terminal, inventory_2, play_circle) לא קיימים בגרסת
 * MaterialIcons-Regular המצורפת ולכן מוצגים ריקים / כעיגול בלבד. לאלו נצייר SVG
 * מוטבע (path רשמי של Material) כדי שיוצגו תמיד בצורה נכונה וברורה.
 */
const SVG_PATHS: Partial<Record<MaterialIconName, string>> = {
  play_arrow: "M8 5v14l11-7z",
  play_circle:
    "M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z",
  stop: "M6 6h12v12H6z",
  terminal:
    "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H4V8h16v10zM6.5 10l3.5 3-3.5 3v-1.7L8.3 13 6.5 11.7V10zm5 4.5h5V16h-5z",
  inventory_2:
    "M20 2H4c-1 0-2 .9-2 2v3.01c0 .72.43 1.34 1 1.69V20c0 1.1 1.1 2 2 2h14c.9 0 2-.9 2-2V8.7c.57-.35 1-.97 1-1.69V4c0-1.1-1-2-2-2zm-5 12H9v-2h6v2zm5-7H4V4l16-.02V7z",
  refresh:
    "M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z",
  minimize: "M6 19h12v2H6z",
  crop_square: "M18 4H6c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H6V6h12v12z",
  filter_none:
    "M3 5v14c0 1.1.89 2 2 2h14v-2H5V5H3zm16-4H9c-1.11 0-2 .9-2 2v12c0 1.1.89 2 2 2h10c1.1 0 2-.9 2-2V3c0-1.1-.9-2-2-2zm0 14H9V3h10v12z",
};

type Props = {
  name: MaterialIconName;
  size?: number;
  className?: string;
  title?: string;
  style?: CSSProperties;
};

export function Icon({ name, size = 20, className = "", title, style }: Props) {
  const svgPath = SVG_PATHS[name];
  if (svgPath) {
    const glyph: ReactNode = (
      <path d={svgPath} />
    );
    return (
      <svg
        className={`material-svg${className ? ` ${className}` : ""}`}
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="currentColor"
        style={style}
        aria-hidden={title ? undefined : true}
        role={title ? "img" : undefined}
        aria-label={title}
      >
        {title ? <title>{title}</title> : null}
        {glyph}
      </svg>
    );
  }
  return (
    <span
      className={`material-icons${className ? ` ${className}` : ""}`}
      style={{ fontSize: size, ...style }}
      title={title}
      aria-hidden={title ? undefined : true}
      role={title ? "img" : undefined}
      aria-label={title}
    >
      {name}
    </span>
  );
}
