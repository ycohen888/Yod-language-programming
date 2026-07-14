export type SymbolKind = "משתנה" | "קבוע" | "פונקציה" | "מחלקה" | "מודול" | "יצוא" | "קובץ" | "פרמטר";

export type ProjectSymbol = {
  name: string;
  kind: SymbolKind;
  file: string;
  line: number;
  /** true = מהקובץ הפתוח כרגע (עדיפות בהשלמה) */
  live?: boolean;
};

export type ProjectIndex = {
  root: string | null;
  files: string[];
  symbols: ProjectSymbol[];
  updatedAt: number;
};

const empty: ProjectIndex = { root: null, files: [], symbols: [], updatedAt: 0 };

let current: ProjectIndex = empty;
/** סמלים חיים מהעורך — לפי מפתח טאב/נתיב */
const liveByKey = new Map<string, ProjectSymbol[]>();

export function getProjectIndex(): ProjectIndex {
  return current;
}

export function setProjectIndex(idx: ProjectIndex) {
  current = idx;
}

export function clearProjectIndex() {
  current = empty;
  liveByKey.clear();
}

export function setLiveDocument(key: string, source: string) {
  liveByKey.set(key, extractSymbols(key, source).map((s) => ({ ...s, live: true })));
}

export function clearLiveDocument(key: string) {
  liveByKey.delete(key);
}

/** סמלים חיים של מסמך פתוח (לפי מפתח טאב). */
export function getLiveSymbols(key: string): ProjectSymbol[] {
  return liveByKey.get(key) ?? [];
}

/**
 * ניתוח קובץ: מעדיף סמלים חיים מהעורך אם יש;
 * אחרת מהאינדקס בדיסק.
 */
export function getSymbolsForFile(filePath: string, liveKey?: string | null): ProjectSymbol[] {
  if (liveKey) {
    const live = liveByKey.get(liveKey);
    if (live && live.length > 0) {
      return [...live].sort((a, b) => a.line - b.line || a.name.localeCompare(b.name, "he"));
    }
  }
  const fromDisk = current.symbols.filter((s) => s.file === filePath);
  return fromDisk.sort((a, b) => a.line - b.line || a.name.localeCompare(b.name, "he"));
}

/** כל הסמלים: פרויקט + מסמכים פתוחים (חיים דורסים אותם שם). */
export function getMergedSymbols(): ProjectSymbol[] {
  const byName = new Map<string, ProjectSymbol>();
  for (const s of current.symbols) {
    const prev = byName.get(s.name);
    if (!prev || kindPriority(s.kind) >= kindPriority(prev.kind)) {
      byName.set(s.name, s);
    }
  }
  for (const list of liveByKey.values()) {
    for (const s of list) {
      byName.set(s.name, s); // live תמיד מנצח
    }
  }
  return [...byName.values()];
}

function kindPriority(kind: SymbolKind): number {
  switch (kind) {
    case "מחלקה":
      return 5;
    case "פונקציה":
      return 4;
    case "מודול":
    case "יצוא":
      return 3;
    case "קבוע":
      return 2;
    case "משתנה":
    case "פרמטר":
      return 1;
    default:
      return 0;
  }
}

const IDENT = "[א-תA-Za-z_][א-תA-Za-z0-9_]*";

function lineAt(src: string, index: number): number {
  let line = 1;
  for (let i = 0; i < index && i < src.length; i++) {
    if (src.charCodeAt(i) === 10) line++;
  }
  return line;
}

function addSym(
  out: ProjectSymbol[],
  seen: Set<string>,
  name: string,
  kind: SymbolKind,
  file: string,
  line: number
) {
  if (!name) return;
  const skip = new Set([
    "סוף",
    "אם",
    "אחרת",
    "עבור",
    "פונקציה",
    "משתנה",
    "קבוע",
    "מחלקה",
    "מודול",
    "פרטי",
    "ציבורי",
    "חדש",
    "זה",
    "הורה",
  ]);
  if (skip.has(name)) return;
  if (seen.has(name)) {
    const existing = out.find((s) => s.name === name);
    if (existing && kindPriority(kind) > kindPriority(existing.kind)) {
      existing.kind = kind;
      existing.line = line;
    }
    return;
  }
  seen.add(name);
  out.push({ name, kind, file, line });
}

export function extractSymbols(filePath: string, source: string): ProjectSymbol[] {
  const out: ProjectSymbol[] = [];
  const seen = new Set<string>();

  const rules: { kind: SymbolKind; re: RegExp }[] = [
    { kind: "משתנה", re: new RegExp(`(?:^|\\n)\\s*(?:פרטי\\s+|ציבורי\\s+)?משתנה\\s+(${IDENT})`, "g") },
    { kind: "קבוע", re: new RegExp(`(?:^|\\n)\\s*(?:פרטי\\s+|ציבורי\\s+)?קבוע\\s+(${IDENT})`, "g") },
    { kind: "פונקציה", re: new RegExp(`(?:^|\\n)\\s*(?:פרטי\\s+|ציבורי\\s+)?פונקציה\\s+(${IDENT})`, "g") },
    { kind: "מחלקה", re: new RegExp(`(?:^|\\n)\\s*מחלקה\\s+(${IDENT})`, "g") },
    { kind: "מודול", re: new RegExp(`(?:^|\\n)\\s*מודול\\s+(${IDENT})`, "g") },
    { kind: "יצוא", re: new RegExp(`(?:^|\\n)\\s*יצא\\s+(${IDENT})`, "g") },
    { kind: "משתנה", re: new RegExp(`(?:^|\\n)\\s*עבור\\s+(${IDENT})\\s+מ`, "g") },
  ];

  for (const { kind, re } of rules) {
    re.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(source))) {
      addSym(out, seen, m[1], kind, filePath, lineAt(source, m.index));
    }
  }

  const fnRe = new RegExp(`פונקציה\\s+${IDENT}\\s*\\(([^)]*)\\)`, "g");
  let fm: RegExpExecArray | null;
  while ((fm = fnRe.exec(source))) {
    const params = fm[1];
    if (!params.trim()) continue;
    for (const part of params.split(",")) {
      const pm = part.trim().match(new RegExp(`^(${IDENT})`));
      if (pm) addSym(out, seen, pm[1], "פרמטר", filePath, lineAt(source, fm.index));
    }
  }

  return out;
}

export async function collectYodFiles(
  root: string,
  readDir: (p: string) => Promise<{ name: string; path: string; isDir: boolean }[]>
): Promise<string[]> {
  const out: string[] = [];
  const queue = [root];
  const skip = new Set(["node_modules", "dist", ".git", "yod-ide"]);
  while (queue.length) {
    const dir = queue.pop()!;
    let kids: { name: string; path: string; isDir: boolean }[] = [];
    try {
      kids = await readDir(dir);
    } catch {
      continue;
    }
    for (const ent of kids) {
      if (ent.isDir) {
        if (skip.has(ent.name) || ent.name.startsWith(".")) continue;
        queue.push(ent.path);
      } else if (ent.name.endsWith(".יוד") || ent.name.toLowerCase().endsWith(".yod")) {
        out.push(ent.path);
      }
    }
  }
  return out.sort((a, b) => a.localeCompare(b, "he"));
}

export async function buildProjectIndex(
  root: string | null,
  api: {
    readDir: (p: string) => Promise<{ name: string; path: string; isDir: boolean }[]>;
    readFile: (p: string) => Promise<{ text: string }>;
  }
): Promise<ProjectIndex> {
  if (!root) {
    const idx = empty;
    setProjectIndex(idx);
    return idx;
  }
  const files = await collectYodFiles(root, api.readDir);
  const symbols: ProjectSymbol[] = [];
  const limit = Math.min(files.length, 250);
  for (let i = 0; i < limit; i++) {
    const f = files[i];
    try {
      const { text } = await api.readFile(f);
      symbols.push(...extractSymbols(f, text));
    } catch {
      /* ignore */
    }
  }
  const idx: ProjectIndex = { root, files, symbols, updatedAt: Date.now() };
  setProjectIndex(idx);
  return idx;
}
