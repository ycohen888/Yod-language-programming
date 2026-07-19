/** פירוק פלט בדיקה/הרצה של יוד לשורות בעיות. */
export function parseProblems(text: string): {
  file?: string;
  line?: number;
  message: string;
  severity: "error" | "warning" | "info";
}[] {
  const lines = text.split(/\r?\n/).filter((l) => l.trim());
  const out: ReturnType<typeof parseProblems> = [];
  const re = /(?:שורה\s+(\d+)|:(\d+):)/;
  for (const line of lines) {
    let severity: "error" | "warning" | "info" = "info";
    if (/שגיאה|error/i.test(line)) severity = "error";
    else if (/אזהרה|warning/i.test(line)) severity = "warning";
    const m = line.match(re);
    const lineNum = m ? Number(m[1] || m[2]) : undefined;
    const fileMatch = line.match(/([^\s:]+\.יוד)/);
    out.push({
      file: fileMatch?.[1],
      line: lineNum && Number.isFinite(lineNum) ? lineNum : undefined,
      message: line,
      severity,
    });
  }
  return out;
}

/** נתיב תיקייה אמיתי ל־Windows. */
export function pathDir(p: string): string {
  const idx = Math.max(p.lastIndexOf("\\"), p.lastIndexOf("/"));
  return idx >= 0 ? p.slice(0, idx) : p;
}

export function pathBase(p: string): string {
  const idx = Math.max(p.lastIndexOf("\\"), p.lastIndexOf("/"));
  return idx >= 0 ? p.slice(idx + 1) : p;
}

/** סיומות משפחת שפת יוד — רק אלה נפתחות ב־RTL בעורך. */
export function isYodFamilyFile(pathOrName: string | null | undefined): boolean {
  if (!pathOrName) return false;
  const name = pathBase(pathOrName);
  if (name.endsWith(".יוד")) return true;
  const lower = name.toLowerCase();
  return lower.endsWith(".yod");
}

export function joinPath(dir: string, name: string): string {
  if (!dir) return name;
  const sep = dir.includes("\\") ? "\\" : "/";
  return dir.replace(/[\\/]+$/, "") + sep + name;
}

/**
 * מנסה למצוא קובץ אמיתי כשהנתיב שגוי או יחסי
 * (למשל עיצוב.יוד תחת צייר במקום רכיבים).
 */
export async function resolveFilePath(
  candidate: string,
  opts: {
    root?: string | null;
    activeDir?: string | null;
    indexedFiles?: string[];
    exists: (p: string) => Promise<boolean>;
  }
): Promise<string | null> {
  if (!candidate) return null;
  const base = pathBase(candidate);
  const candidates: string[] = [];
  const add = (p: string | null | undefined) => {
    if (p && !candidates.includes(p)) candidates.push(p);
  };

  add(candidate);
  if (opts.root) {
    add(joinPath(opts.root, candidate));
    add(joinPath(opts.root, base));
    add(joinPath(opts.root, joinPath("רכיבים", base)));
  }
  if (opts.activeDir) {
    add(joinPath(opts.activeDir, base));
    add(joinPath(opts.activeDir, candidate));
    const parent = pathDir(opts.activeDir);
    add(joinPath(parent, joinPath("רכיבים", base)));
    add(joinPath(parent, base));
  }
  for (const f of opts.indexedFiles ?? []) {
    if (pathBase(f) === base) add(f);
  }

  for (const p of candidates) {
    try {
      if (await opts.exists(p)) return p;
    } catch {
      /* ignore */
    }
  }
  return null;
}
