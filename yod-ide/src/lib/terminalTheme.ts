import type { ITheme } from "@xterm/xterm";
import { paletteForTheme, type EditorPalette } from "./yodHighlightStyle";
import type { ThemeName } from "../types";

/** בונה ערכת צבעים ל-xterm מתוך פלטת העורך. */
export function xtermThemeFromPalette(p: EditorPalette): ITheme {
  return {
    background: p.background,
    foreground: p.foreground,
    cursor: p.foreground,
    cursorAccent: p.background,
    selectionBackground: p.selection,
    black: p.dark ? "#1e1e1e" : "#000000",
    red: p.regexp,
    green: p.comment,
    yellow: p.function,
    blue: p.keyword,
    magenta: p.control,
    cyan: p.className,
    white: p.foreground,
    brightBlack: p.lineNumber,
    brightRed: p.invalid,
    brightGreen: p.comment,
    brightYellow: p.function,
    brightBlue: p.keyword,
    brightMagenta: p.control,
    brightCyan: p.className,
    brightWhite: p.dark ? "#ffffff" : "#000000",
  };
}

/** ערכת צבעים ל-xterm לפי שם הנושא. */
export function xtermThemeForName(theme: ThemeName): ITheme {
  return xtermThemeFromPalette(paletteForTheme(theme));
}
