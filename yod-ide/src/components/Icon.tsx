import type { CSSProperties } from "react";
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
  | "widgets";

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

type Props = {
  name: MaterialIconName;
  size?: number;
  className?: string;
  title?: string;
  style?: CSSProperties;
};

export function Icon({ name, size = 20, className = "", title, style }: Props) {
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
