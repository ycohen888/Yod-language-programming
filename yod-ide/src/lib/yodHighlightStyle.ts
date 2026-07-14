import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";

/**
 * צבעי PHP Dark+ כמו ב־yod/internal/highlight/spans.go (Cursor theme-defaults).
 */
export const yodPhpDarkColors = {
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
} as const;

export const yodHighlightStyle = HighlightStyle.define([
  { tag: t.comment, color: yodPhpDarkColors.comment, fontStyle: "italic" },
  { tag: t.lineComment, color: yodPhpDarkColors.comment, fontStyle: "italic" },
  { tag: t.blockComment, color: yodPhpDarkColors.comment, fontStyle: "italic" },
  { tag: t.string, color: yodPhpDarkColors.string },
  { tag: t.special(t.string), color: yodPhpDarkColors.string },
  { tag: t.number, color: yodPhpDarkColors.number },
  { tag: t.bool, color: yodPhpDarkColors.keyword },
  { tag: t.null, color: yodPhpDarkColors.keyword },
  { tag: t.atom, color: yodPhpDarkColors.keyword },
  { tag: t.controlKeyword, color: yodPhpDarkColors.control },
  { tag: t.keyword, color: yodPhpDarkColors.keyword },
  { tag: t.modifier, color: yodPhpDarkColors.keyword },
  { tag: t.definitionKeyword, color: yodPhpDarkColors.keyword },
  { tag: t.moduleKeyword, color: yodPhpDarkColors.keyword },
  { tag: t.operatorKeyword, color: yodPhpDarkColors.control },
  { tag: t.function(t.variableName), color: yodPhpDarkColors.function },
  { tag: t.function(t.definition(t.variableName)), color: yodPhpDarkColors.function },
  { tag: t.definition(t.function(t.variableName)), color: yodPhpDarkColors.function },
  { tag: t.variableName, color: yodPhpDarkColors.variable },
  { tag: t.definition(t.variableName), color: yodPhpDarkColors.variable },
  { tag: t.propertyName, color: yodPhpDarkColors.variable },
  { tag: t.className, color: yodPhpDarkColors.className },
  { tag: t.typeName, color: yodPhpDarkColors.className },
  { tag: t.namespace, color: yodPhpDarkColors.className },
  { tag: t.operator, color: yodPhpDarkColors.operator },
  { tag: t.punctuation, color: yodPhpDarkColors.operator },
  { tag: t.bracket, color: yodPhpDarkColors.operator },
  { tag: t.brace, color: yodPhpDarkColors.operator },
  { tag: t.paren, color: yodPhpDarkColors.operator },
  { tag: t.squareBracket, color: yodPhpDarkColors.operator },
  { tag: t.separator, color: yodPhpDarkColors.operator },
  { tag: t.name, color: yodPhpDarkColors.foreground },
]);

export const yodSyntaxHighlighting = syntaxHighlighting(yodHighlightStyle);
