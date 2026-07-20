export type DirEntry = {
  name: string;
  path: string;
  isDir: boolean;
};

export type OpenTab = {
  key: string;
  path: string | null;
  title: string;
  dirty: boolean;
  modelUri: string;
};

export type PanelKind = "output" | "problems" | "search" | "bookmarks" | "terminal";

/** סוג מעטפת לטרמינל המשולב. */
export type ShellKind = "powershell" | "cmd";

/** נקודה שמורה בקוד — קפיצה מהירה לטאב+שורה. */
export type Bookmark = {
  id: string;
  /** מפתח הטאב (בדרך כלל נתיב הקובץ) */
  key: string;
  /** נתיב הקובץ (null לקובץ ללא שם) */
  path: string | null;
  /** שם התצוגה של הקובץ */
  title: string;
  line: number;
};

export type Problem = {
  file?: string;
  line?: number;
  message: string;
  severity: "error" | "warning" | "info";
};

/** פריט ברשימת "נפתחו לאחרונה". */
export type RecentEntry = {
  kind: "folder" | "file";
  path: string;
};

/** מצב סשן שנשמר בין הפעלות. */
export type SessionState = {
  root: string | null;
  /** נתיבי הקבצים הפתוחים (טאבים ללא-שם לא נשמרים) */
  tabs: string[];
  /** מפתח הטאב הפעיל */
  activeKey: string | null;
  /** מיקום שורת הסמן לכל קובץ */
  lines: Record<string, number>;
};

/** ערכת נושא לעורך ול-IDE. */
export type ThemeName = "dark" | "light" | "high-contrast";

/** העדפות המשתמש (מסך הגדרות). */
export type Settings = {
  theme: ThemeName;
  fontFamily: string;
  fontSize: number;
  tabSize: number;
  wordWrap: boolean;
  autosave: boolean;
  formatOnSave: boolean;
};

export type CommandItem = {
  id: string;
  label: string;
  detail?: string;
  keybinding?: string;
  run: () => void | Promise<void>;
};

export type DialogPromptOpts =
  | { kind: "confirm"; title?: string; message: string }
  | { kind: "info"; title?: string; message: string; detail?: string };

export type YodApi = {
  readDir: (dirPath: string) => Promise<DirEntry[]>;
  readFile: (filePath: string) => Promise<{ text: string; size: number; mtimeMs: number }>;
  writeFile: (filePath: string, text: string) => Promise<boolean>;
  mkdir: (dirPath: string) => Promise<boolean>;
  rename: (from: string, to: string) => Promise<boolean>;
  remove: (p: string) => Promise<boolean>;
  stat: (p: string) => Promise<{ isDir: boolean; size: number; mtimeMs: number }>;
  exists: (p: string) => Promise<boolean>;
  openFolder: () => Promise<string | null>;
  openFileDialog: () => Promise<string | null>;
  saveFileDialog: (defaultPath?: string) => Promise<string | null>;
  dialogPrompt: (opts: DialogPromptOpts) => Promise<boolean | null>;
  showItem: (p: string) => Promise<void>;
  openPath: (p: string) => Promise<void>;
  openExternal: (url: string) => Promise<boolean>;
  writeClipboard: (text: string) => Promise<boolean>;
  openGuide: () => Promise<{ ok: boolean; path?: string; error?: string }>;
  getPaths: () => Promise<{
    yodExe: string;
    userData: string;
    version: string;
    ideRoot: string;
    guideHtml: string;
  }>;
  quit: () => Promise<boolean>;
  windowMinimize: () => Promise<boolean>;
  windowMaximizeToggle: () => Promise<boolean>;
  windowClose: () => Promise<boolean>;
  windowIsMaximized: () => Promise<boolean>;
  onMaximized: (cb: (maximized: boolean) => void) => () => void;
  runYod: (
    args: string[],
    cwd?: string
  ) => Promise<{ code: number; stdout: string; stderr: string; exe: string; args: string[] }>;
  terminal: {
    create: (opts: { id: string; cwd?: string; shell?: ShellKind }) => Promise<boolean>;
    write: (id: string, data: string) => Promise<boolean>;
    resize: (id: string, cols: number, rows: number) => Promise<boolean>;
    kill: (id: string) => Promise<boolean>;
    onData: (cb: (payload: { id: string; data: string }) => void) => () => void;
    onExit: (cb: (payload: { id: string; code: number }) => void) => () => void;
  };
  onOpenPath: (cb: (p: string) => void) => () => void;
  onMenu: (cb: (action: string) => void) => () => void;
};

declare global {
  interface Window {
    yod: YodApi;
  }
}

export {};
