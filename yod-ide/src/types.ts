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

export type PanelKind = "output" | "problems" | "search";

export type Problem = {
  file?: string;
  line?: number;
  message: string;
  severity: "error" | "warning" | "info";
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
  runYod: (
    args: string[],
    cwd?: string
  ) => Promise<{ code: number; stdout: string; stderr: string; exe: string; args: string[] }>;
  onOpenPath: (cb: (p: string) => void) => () => void;
  onMenu: (cb: (action: string) => void) => () => void;
};

declare global {
  interface Window {
    yod: YodApi;
  }
}

export {};
