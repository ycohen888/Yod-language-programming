import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import type { Extension } from "@codemirror/state";
import { tags as t } from "@lezer/highlight";
import type { ThemeName } from "../types";

/** מפת צבעים לערכת נושא של העורך (הדגשת תחביר + כרום). */
export type EditorPalette = {
  background: string;
  foreground: string;
  comment: string;
  string: string;
  number: string;
  control: string;
  keyword: string;
  function: string;
  variable: string;
  className: string;
  operator: string;
  lineNumber: string;
  tag: string;
  attribute: string;
  meta: string;
  regexp: string;
  heading: string;
  link: string;
  invalid: string;
  /** רקע השורה הפעילה */
  activeLine: string;
  /** רקע השוליים בשורה הפעילה */
  activeLineGutter: string;
  /** רקע הסימון (selection) */
  selection: string;
  /** קו מפריד בין השוליים לתוכן */
  gutterBorder: string;
  /** האם הנושא כהה (עבור CodeMirror) */
  dark: boolean;
};

/**
 * צבעי PHP Dark+ (Cursor theme-defaults) להדגשת תחביר —
 * משותף ליוד ולשפות נוספות (JS/PHP/HTML/CSS/SQL…).
 */
export const darkPalette: EditorPalette = {
  background: "#1E1E1E",
  foreground: "#D4D4D4",
  comment: "#6A9955",
  string: "#CE9178",
  number: "#B5CEA8",
  control: "#C586C0",
  keyword: "#569CD6",
  function: "#DCDCAA",
  variable: "#9CDCFE",
  className: "#4EC9B0",
  operator: "#D4D4D4",
  lineNumber: "#858585",
  tag: "#569CD6",
  attribute: "#9CDCFE",
  meta: "#D4D4D4",
  regexp: "#D16969",
  heading: "#569CD6",
  link: "#CE9178",
  invalid: "#f44747",
  activeLine: "rgba(255, 255, 255, 0.055)",
  activeLineGutter: "#2a2d2e",
  selection: "#264f78",
  gutterBorder: "#3c3c3c",
  dark: true,
};

/** ערכה בהירה (Light+). */
export const lightPalette: EditorPalette = {
  background: "#ffffff",
  foreground: "#1f1f1f",
  comment: "#008000",
  string: "#a31515",
  number: "#098658",
  control: "#af00db",
  keyword: "#0000ff",
  function: "#795e26",
  variable: "#001080",
  className: "#267f99",
  operator: "#1f1f1f",
  lineNumber: "#999999",
  tag: "#800000",
  attribute: "#e50000",
  meta: "#1f1f1f",
  regexp: "#811f3f",
  heading: "#0000ff",
  link: "#a31515",
  invalid: "#cd3131",
  activeLine: "rgba(0, 0, 0, 0.045)",
  activeLineGutter: "#e8e8e8",
  selection: "#add6ff",
  gutterBorder: "#e0e0e0",
  dark: false,
};

/** ערכת ניגודיות גבוהה (High Contrast). */
export const highContrastPalette: EditorPalette = {
  background: "#000000",
  foreground: "#ffffff",
  comment: "#7ca668",
  string: "#ce9178",
  number: "#b5cea8",
  control: "#c586c0",
  keyword: "#569cd6",
  function: "#dcdcaa",
  variable: "#9cdcfe",
  className: "#4ec9b0",
  operator: "#ffffff",
  lineNumber: "#ffffff",
  tag: "#569cd6",
  attribute: "#9cdcfe",
  meta: "#ffffff",
  regexp: "#d16969",
  heading: "#569cd6",
  link: "#ce9178",
  invalid: "#f48771",
  activeLine: "rgba(255, 255, 255, 0.10)",
  activeLineGutter: "#1a1a1a",
  selection: "#f3f518",
  gutterBorder: "#6fc3df",
  dark: true,
};

/** מחזיר את פלטת הצבעים לפי שם הנושא. */
export function paletteForTheme(theme: ThemeName): EditorPalette {
  switch (theme) {
    case "light":
      return lightPalette;
    case "high-contrast":
      return highContrastPalette;
    default:
      return darkPalette;
  }
}

/** בונה סגנון הדגשת תחביר לפי פלטה. */
export function makeHighlightStyle(p: EditorPalette): HighlightStyle {
  return HighlightStyle.define([
    { tag: t.comment, color: p.comment, fontStyle: "italic" },
    { tag: t.lineComment, color: p.comment, fontStyle: "italic" },
    { tag: t.blockComment, color: p.comment, fontStyle: "italic" },
    { tag: t.docComment, color: p.comment, fontStyle: "italic" },
    { tag: t.string, color: p.string },
    { tag: t.special(t.string), color: p.string },
    { tag: t.character, color: p.string },
    { tag: t.number, color: p.number },
    { tag: t.integer, color: p.number },
    { tag: t.float, color: p.number },
    { tag: t.bool, color: p.keyword },
    { tag: t.null, color: p.keyword },
    { tag: t.atom, color: p.keyword },
    { tag: t.unit, color: p.number },
    { tag: t.controlKeyword, color: p.control },
    { tag: t.keyword, color: p.keyword },
    { tag: t.modifier, color: p.keyword },
    { tag: t.definitionKeyword, color: p.keyword },
    { tag: t.moduleKeyword, color: p.keyword },
    { tag: t.operatorKeyword, color: p.control },
    { tag: t.function(t.variableName), color: p.function },
    { tag: t.function(t.definition(t.variableName)), color: p.function },
    { tag: t.definition(t.function(t.variableName)), color: p.function },
    { tag: t.variableName, color: p.variable },
    { tag: t.definition(t.variableName), color: p.variable },
    { tag: t.propertyName, color: p.variable },
    { tag: t.attributeName, color: p.attribute },
    { tag: t.attributeValue, color: p.string },
    { tag: t.className, color: p.className },
    { tag: t.typeName, color: p.className },
    { tag: t.namespace, color: p.className },
    { tag: t.labelName, color: p.variable },
    { tag: t.tagName, color: p.tag },
    { tag: t.angleBracket, color: p.operator },
    { tag: t.operator, color: p.operator },
    { tag: t.punctuation, color: p.operator },
    { tag: t.bracket, color: p.operator },
    { tag: t.brace, color: p.operator },
    { tag: t.paren, color: p.operator },
    { tag: t.squareBracket, color: p.operator },
    { tag: t.separator, color: p.operator },
    { tag: t.meta, color: p.meta },
    { tag: t.processingInstruction, color: p.meta },
    { tag: t.regexp, color: p.regexp },
    { tag: t.escape, color: p.string },
    { tag: t.color, color: p.number },
    { tag: t.heading, color: p.heading, fontWeight: "bold" },
    { tag: t.heading1, color: p.heading, fontWeight: "bold" },
    { tag: t.heading2, color: p.heading, fontWeight: "bold" },
    { tag: t.heading3, color: p.heading, fontWeight: "bold" },
    { tag: t.link, color: p.link },
    { tag: t.url, color: p.link },
    { tag: t.emphasis, fontStyle: "italic" },
    { tag: t.strong, fontWeight: "bold" },
    { tag: t.strikethrough, textDecoration: "line-through" },
    { tag: t.monospace, color: p.foreground },
    { tag: t.invalid, color: p.invalid },
    { tag: t.name, color: p.foreground },
    { tag: t.literal, color: p.string },
    { tag: t.self, color: p.keyword },
  ]);
}

/** מחזיר תוסף הדגשת תחביר לפי פלטה. */
export function makeSyntaxHighlighting(p: EditorPalette): Extension {
  return syntaxHighlighting(makeHighlightStyle(p));
}

// --- תאימות לאחור: ברירת המחדל הכהה ---
export const yodPhpDarkColors = darkPalette;
export const yodHighlightStyle = makeHighlightStyle(darkPalette);
export const yodSyntaxHighlighting = syntaxHighlighting(yodHighlightStyle);
