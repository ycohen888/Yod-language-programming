import type { Settings, ThemeName } from "../types";

const SETTINGS_KEY = "yod-ide:settings";

/** ברירות המחדל להעדפות המשתמש. */
export const DEFAULT_SETTINGS: Settings = {
  theme: "dark",
  fontFamily: 'Consolas, "Courier New", "Noto Sans Hebrew", monospace',
  fontSize: 14,
  tabSize: 4,
  wordWrap: false,
  autosave: false,
  formatOnSave: false,
};

/** רשימת משפחות גופן זמינות לבחירה במסך ההגדרות. */
export const FONT_FAMILIES: { label: string; value: string }[] = [
  { label: "Consolas (ברירת מחדל)", value: 'Consolas, "Courier New", "Noto Sans Hebrew", monospace' },
  { label: "Cascadia Code", value: '"Cascadia Code", Consolas, "Noto Sans Hebrew", monospace' },
  { label: "Fira Code", value: '"Fira Code", Consolas, "Noto Sans Hebrew", monospace' },
  { label: "JetBrains Mono", value: '"JetBrains Mono", Consolas, "Noto Sans Hebrew", monospace' },
  { label: "Courier New", value: '"Courier New", monospace' },
  { label: "Menlo / Monaco", value: 'Menlo, Monaco, "Courier New", monospace' },
];

const VALID_THEMES: ThemeName[] = ["dark", "light", "high-contrast"];

function clampNumber(n: unknown, min: number, max: number, fallback: number): number {
  const v = typeof n === "number" && Number.isFinite(n) ? n : fallback;
  return Math.min(max, Math.max(min, Math.round(v)));
}

/** טעינת העדפות המשתמש (עם מיזוג לברירות מחדל וולידציה). */
export function loadSettings(): Settings {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY);
    if (!raw) return { ...DEFAULT_SETTINGS };
    const parsed = JSON.parse(raw) as Partial<Settings>;
    if (!parsed || typeof parsed !== "object") return { ...DEFAULT_SETTINGS };
    return {
      theme: VALID_THEMES.includes(parsed.theme as ThemeName)
        ? (parsed.theme as ThemeName)
        : DEFAULT_SETTINGS.theme,
      fontFamily:
        typeof parsed.fontFamily === "string" && parsed.fontFamily.trim()
          ? parsed.fontFamily
          : DEFAULT_SETTINGS.fontFamily,
      fontSize: clampNumber(parsed.fontSize, 10, 28, DEFAULT_SETTINGS.fontSize),
      tabSize: clampNumber(parsed.tabSize, 1, 8, DEFAULT_SETTINGS.tabSize),
      wordWrap: typeof parsed.wordWrap === "boolean" ? parsed.wordWrap : DEFAULT_SETTINGS.wordWrap,
      autosave: typeof parsed.autosave === "boolean" ? parsed.autosave : DEFAULT_SETTINGS.autosave,
      formatOnSave:
        typeof parsed.formatOnSave === "boolean"
          ? parsed.formatOnSave
          : DEFAULT_SETTINGS.formatOnSave,
    };
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
}

/** שמירת העדפות המשתמש. */
export function saveSettings(settings: Settings): void {
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
  } catch {
    /* אחסון חסום/מלא — מתעלמים */
  }
}
