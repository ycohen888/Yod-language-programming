import { foldService, indentService } from "@codemirror/language";
import type { Text } from "@codemirror/state";
import { tokenize } from "./yodBlockMatch";

type Pair = { openFrom: number; openTo: number; closeFrom: number; closeTo: number };

type Struct = {
  /** מיקומי סוף-פותח (openTo) ממוינים — לספירת בלוקים פתוחים לפני מיקום */
  opens: number[];
  /** מיקומי תחילת-סוגר (closeFrom) ממוינים */
  closes: number[];
  /** זוגות רב-שורתיים לקיפול, לפי מספר שורת הפותח */
  foldByLine: Map<number, { from: number; to: number }>;
};

/**
 * בונה את זוגות הבלוקים/סוגריים מהטקסט — כמו buildPairMap אבל מחזיר מערך זוגות
 * עם התאמת סגנון «אם (…) { }» (הסוגר המסולסל מחליף את «סוף»).
 */
function buildPairs(text: string): Pair[] {
  const toks = tokenize(text);
  const pairs: Pair[] = [];

  type Frame =
    | { type: "brace" | "paren" | "bracket" | "word"; from: number; to: number };
  const stack: Frame[] = [];
  let braceDepth = 0;
  let parenDepth = 0;
  let bracketDepth = 0;

  const push = (openFrom: number, openTo: number, closeFrom: number, closeTo: number) =>
    pairs.push({ openFrom, openTo, closeFrom, closeTo });

  for (const t of toks) {
    if (t.kind === "braceOpen") {
      if (stack.length && stack[stack.length - 1].type === "word") stack.pop();
      stack.push({ type: "brace", from: t.from, to: t.to });
      braceDepth++;
      continue;
    }
    if (t.kind === "braceClose") {
      for (let s = stack.length - 1; s >= 0; s--) {
        if (stack[s].type === "brace") {
          const open = stack[s];
          stack.splice(s, 1);
          push(open.from, open.to, t.from, t.to);
          braceDepth = Math.max(0, braceDepth - 1);
          break;
        }
      }
      continue;
    }
    if (t.kind === "parenOpen") {
      stack.push({ type: "paren", from: t.from, to: t.to });
      parenDepth++;
      continue;
    }
    if (t.kind === "parenClose") {
      for (let s = stack.length - 1; s >= 0; s--) {
        if (stack[s].type === "paren") {
          const open = stack[s];
          stack.splice(s, 1);
          push(open.from, open.to, t.from, t.to);
          parenDepth = Math.max(0, parenDepth - 1);
          break;
        }
      }
      continue;
    }
    if (t.kind === "bracketOpen") {
      stack.push({ type: "bracket", from: t.from, to: t.to });
      bracketDepth++;
      continue;
    }
    if (t.kind === "bracketClose") {
      for (let s = stack.length - 1; s >= 0; s--) {
        if (stack[s].type === "bracket") {
          const open = stack[s];
          stack.splice(s, 1);
          push(open.from, open.to, t.from, t.to);
          bracketDepth = Math.max(0, bracketDepth - 1);
          break;
        }
      }
      continue;
    }

    // מילות בלוק/סוף — רק מחוץ לסוגריים
    if (braceDepth > 0 || parenDepth > 0 || bracketDepth > 0) continue;

    if (t.kind === "openWord") {
      stack.push({ type: "word", from: t.from, to: t.to });
      continue;
    }
    if (t.kind === "closeWord") {
      for (let s = stack.length - 1; s >= 0; s--) {
        if (stack[s].type === "word") {
          const open = stack[s];
          stack.splice(s, 1);
          push(open.from, open.to, t.from, t.to);
          break;
        }
      }
    }
  }

  return pairs;
}

const cache = new WeakMap<Text, Struct>();

function getStruct(doc: Text): Struct {
  const cached = cache.get(doc);
  if (cached) return cached;

  const text = doc.toString();
  const pairs = buildPairs(text);
  const opens: number[] = [];
  const closes: number[] = [];
  const foldByLine = new Map<number, { from: number; to: number }>();

  for (const p of pairs) {
    opens.push(p.openTo);
    closes.push(p.closeFrom);
    const openLine = doc.lineAt(p.openFrom);
    const closeLine = doc.lineAt(p.closeFrom);
    if (closeLine.number > openLine.number) {
      const from = openLine.to;
      const to = p.closeFrom;
      if (to > from) {
        const prev = foldByLine.get(openLine.number);
        // שומרים את הטווח הגדול ביותר לכל שורת-פותח (הבלוק החיצוני)
        if (!prev || to - from > prev.to - prev.from) {
          foldByLine.set(openLine.number, { from, to });
        }
      }
    }
  }

  opens.sort((a, b) => a - b);
  closes.sort((a, b) => a - b);
  const struct: Struct = { opens, closes, foldByLine };
  cache.set(doc, struct);
  return struct;
}

/** מספר האיברים במערך ממוין שקטנים/שווים ל-x. */
function countLE(arr: number[], x: number): number {
  let lo = 0;
  let hi = arr.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (arr[mid] <= x) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

/** מספר האיברים במערך ממוין שקטנים ממש מ-x. */
function countLT(arr: number[], x: number): number {
  let lo = 0;
  let hi = arr.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (arr[mid] < x) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

const CLOSER_LINE_RE = /^\s*(סוף|\}|\)|\]|אחרת|אחרת_אם|תפוס|לבסוף)/;

/** שירות קיפול קוד ליוד — טווח קיפול לבלוק שמתחיל בשורה הנתונה. */
export const yodFoldService = foldService.of((state, lineStart) => {
  const line = state.doc.lineAt(lineStart);
  const r = getStruct(state.doc).foldByLine.get(line.number);
  return r && r.to > r.from ? r : null;
});

/** שירות הזחה חכמה ליוד — עומק בלוקים * יחידת הזחה, עם דה-הזחה לשורת סוגר. */
export const yodIndentService = indentService.of((context, pos) => {
  const doc = context.state.doc;
  const line = doc.lineAt(pos);
  const { opens, closes } = getStruct(doc);
  let depth = countLE(opens, line.from) - countLT(closes, line.from);
  if (CLOSER_LINE_RE.test(line.text)) depth -= 1;
  if (depth < 0) depth = 0;
  return depth * context.unit;
});
