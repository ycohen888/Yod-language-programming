import {
  autocompletion,
  completionKeymap,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import { keymap } from "@codemirror/view";
import { YOD_BUILTINS, YOD_KEYWORDS } from "./yodKeywords";
import {
  extractSymbols,
  getMergedSymbols,
  getProjectIndex,
  type SymbolKind,
} from "./projectIndex";

export const BUILTIN_MODULES: Record<string, string[]> = {
  בסיס: ["הדפס", "קלט", "אורך", "טווח", "אקראי"],
  קבצים: ["קרא", "כתוב", "קיים", "מחק", "רשום", "העתק", "תיקייה", "קרא_תוצאה"],
  JSON: ["פרסר", "מחרוזת", "יפה"],
  זמן: ["עכשיו", "פורמט", "פרסר", "הפרש"],
  טיימרים: ["טיימר", "אחרי", "כל_כמה", "פעם_אחת", "עצור", "עצור_הכל", "השהה", "המשך"],
  מתמטיקה: ["פי", "אי", "סינוס", "קוסינוס", "טנגנס", "שורש", "חזקה", "ערך_מוחלט", "עיגול"],
  מספרים: ["שלם", "עשרוני", "פסיקים", "כסף", "אחוז"],
  מערכת: ["סביבה", "זמן", "מעבד", "זיכרון", "כונן", "הרץ"],
  רשת: ["קבל", "שלח", "שרת"],
  SQL: ["פתח", "שאילתה", "סגור"],
  חלונות: ["חלון", "כפתור", "תווית", "קלט_טקסט", "משטח", "דפדפן", "טבלה", "הודעה", "שורה", "עמודה"],
  גרפים: ["עמודות", "קו", "עוגה"],
  ציור: ["לוח", "טען_תמונה", "צבע"],
  הצפנה: ["Base64", "MD5", "SHA256", "HMAC"],
  לוח: ["העתק", "הדבק"],
  עכבר: ["מיקום", "שמאל_לחוץ", "ימין_לחוץ", "אמצע_לחוץ", "מצב", "שמאל", "ימין", "אמצע"],
  מקלדת: ["לחוץ", "מצב"],
  שמע: ["נגן", "נגן_אפקט", "השהה", "המשך", "עצור", "לולאה", "עוצמה", "מנגן", "בעת_סיום"],
  וידאו: ["חלונית"],
  תמונות: ["טען", "שמור"],
  דיבור: ["הקרא", "עצור", "קצב"],
  תוצאה: ["מ", "שגיאה"],
};

const BUILTIN_MODULE_NAMES = Object.keys(BUILTIN_MODULES);
const keywordSet = new Set<string>([...YOD_KEYWORDS, ...YOD_BUILTINS, ...BUILTIN_MODULE_NAMES]);

function pathBase(p: string): string {
  const i = Math.max(p.lastIndexOf("\\"), p.lastIndexOf("/"));
  return i >= 0 ? p.slice(i + 1) : p;
}

function relPath(root: string | null, abs: string): string {
  if (!root) return pathBase(abs);
  const normRoot = root.replace(/[\\/]+$/, "");
  if (abs.toLowerCase().startsWith(normRoot.toLowerCase())) {
    return abs.slice(normRoot.length).replace(/^[\\/]+/, "").replace(/\\/g, "/");
  }
  return pathBase(abs);
}

function wordBefore(ctx: CompletionContext): { from: number; text: string } | null {
  const word = ctx.matchBefore(/[א-תA-Za-z0-9_]*$/);
  if (!word || (word.from === word.to && !ctx.explicit)) return null;
  return { from: word.from, text: word.text };
}

function matchesPrefix(label: string, prefix: string): boolean {
  if (!prefix) return true;
  return label.startsWith(prefix) || label.includes(prefix);
}

function scoreMatch(label: string, prefix: string, base: number): number {
  if (!prefix) return base;
  if (label === prefix) return base + 40;
  if (label.startsWith(prefix)) return base + 25;
  if (label.includes(prefix)) return base + 5;
  return base;
}

function fileCompletions(ctx: CompletionContext): CompletionResult | null {
  const line = ctx.state.doc.lineAt(ctx.pos);
  const before = line.text.slice(0, ctx.pos - line.from);
  const m = before.match(/(?:כלול|יבא)\s+"([^"]*)$/);
  if (!m) return null;
  const prefix = m[1] || "";
  const idx = getProjectIndex();
  const options: Completion[] = idx.files
    .map((f) => {
      const rel = relPath(idx.root, f);
      return {
        label: rel,
        type: "text",
        boost: scoreMatch(rel, prefix, 50),
        detail: "קובץ",
        apply: rel.replace(/\\/g, "/"),
      };
    })
    .filter((o) => matchesPrefix(o.label, prefix));
  return { from: ctx.pos - prefix.length, options, filter: false };
}

function memberCompletions(ctx: CompletionContext): CompletionResult | null {
  const before = ctx.state.sliceDoc(Math.max(0, ctx.pos - 100), ctx.pos);
  const m = before.match(/([א-תA-Za-z_][א-תA-Za-z0-9_]*)\.\s*([א-תA-Za-z0-9_]*)$/);
  if (!m) return null;
  const owner = m[1];
  const prefix = m[2] || "";
  const options: Completion[] = [];

  const builtin = BUILTIN_MODULES[owner];
  if (builtin) {
    for (const a of builtin) {
      if (!matchesPrefix(a, prefix)) continue;
      options.push({ label: a, type: "method", boost: scoreMatch(a, prefix, 90), detail: owner });
    }
  }

  // מתודות/שדות: כל הפונקציות והמשתנים בפרויקט (שימושי אחרי מופע מחלקה)
  for (const sym of getMergedSymbols()) {
    if (sym.kind !== "פונקציה" && sym.kind !== "משתנה" && sym.kind !== "קבוע") continue;
    if (keywordSet.has(sym.name)) continue;
    if (!matchesPrefix(sym.name, prefix)) continue;
    options.push({
      label: sym.name,
      type: symbolType(sym.kind),
      boost: scoreMatch(sym.name, prefix, sym.live ? 88 : 70),
      detail: sym.kind,
    });
  }

  if (!options.length && !builtin) return null;
  options.sort((a, b) => (b.boost ?? 0) - (a.boost ?? 0));
  return { from: ctx.pos - prefix.length, options: options.slice(0, 40), filter: false };
}

export function yodCompletionSource(ctx: CompletionContext): CompletionResult | null {
  const files = fileCompletions(ctx);
  if (files) return files;
  const members = memberCompletions(ctx);
  if (members) return members;

  const w = wordBefore(ctx);
  if (!w) return null;
  const prefix = w.text;

  // סמלים חיים מהמסמך הנוכחי — תמיד מעודכן
  const bufferSyms = extractSymbols("__buffer__", ctx.state.doc.toString()).map((s) => ({
    ...s,
    live: true as const,
  }));

  const merged = new Map<string, { kind: SymbolKind; live?: boolean; file: string }>();
  for (const s of getMergedSymbols()) {
    merged.set(s.name, { kind: s.kind, live: s.live, file: s.file });
  }
  for (const s of bufferSyms) {
    merged.set(s.name, { kind: s.kind, live: true, file: s.file });
  }

  const options: Completion[] = [];
  const seen = new Set<string>();

  const add = (label: string, type: string, boost: number, detail?: string) => {
    if (seen.has(label)) return;
    if (!matchesPrefix(label, prefix)) return;
    seen.add(label);
    options.push({ label, type, boost: scoreMatch(label, prefix, boost), detail });
  };

  // קודם משתמש — עדיפות גבוהה
  for (const [name, meta] of merged) {
    if (keywordSet.has(name)) continue;
    const base =
      meta.kind === "מחלקה"
        ? 95
        : meta.kind === "פונקציה"
          ? 93
          : meta.kind === "קבוע" || meta.kind === "משתנה" || meta.kind === "פרמטר"
            ? 92
            : 88;
    add(name, symbolType(meta.kind), meta.live ? base + 8 : base, meta.kind);
  }

  // אחר כך מילות מפתח / מובנים (נמוך יותר כשיש התאמת משתמש)
  for (const k of YOD_KEYWORDS) {
    add(k, "keyword", 55, "מילת מפתח");
  }
  for (const b of YOD_BUILTINS) {
    add(b, "function", 58, "מובנה");
  }
  for (const m of BUILTIN_MODULE_NAMES) {
    add(m, "namespace", 60, "ספרייה");
  }

  for (const f of getProjectIndex().files) {
    const bare = pathBase(f).replace(/\.יוד$/i, "").replace(/\.yod$/i, "");
    if (bare && !keywordSet.has(bare)) add(bare, "text", 40, "קובץ");
  }

  options.sort((a, b) => (b.boost ?? 0) - (a.boost ?? 0) || a.label.localeCompare(b.label, "he"));
  return {
    from: w.from,
    options: options.slice(0, 50),
    validFor: /[א-תA-Za-z0-9_]*$/,
  };
}

function symbolType(kind: SymbolKind): string {
  switch (kind) {
    case "מחלקה":
      return "class";
    case "פונקציה":
      return "function";
    case "משתנה":
    case "קבוע":
    case "פרמטר":
      return "variable";
    case "מודול":
    case "יצוא":
      return "namespace";
    default:
      return "text";
  }
}

export const yodAutocompletion = [
  autocompletion({
    override: [yodCompletionSource],
    activateOnTyping: true,
    maxRenderedOptions: 10,
    closeOnBlur: true,
    icons: false,
  }),
  keymap.of(completionKeymap),
];
