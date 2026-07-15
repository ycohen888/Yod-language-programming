/**
 * מעבר לזוג תואם בעורך יוד:
 * - סוגריים: () [] {}
 * - בלוקים בעברית: פתיחה (אם / פונקציה / …) ↔ סוף
 *   (מילות מפתח בתוך { } מדולגות — בסגנון המודרני עם סוגריים)
 */
import { EditorView } from "@codemirror/view";
import { matchBrackets } from "@codemirror/language";

const WORD_RE = /^[א-תA-Za-z_־][א-תA-Za-z0-9_־]*/;

/** פותחי בלוק שנסגרים ב־«סוף» (בלי סוגריים מסולסלים). */
export const YOD_BLOCK_OPENERS = new Set([
  "אם",
  "אחרת_אם",
  "כל_עוד",
  "עבור",
  "פונקציה",
  "מחלקה",
  "נסה",
  "בחר",
  "בצע",
  "משימה",
]);

type PairHit = { from: number; to: number; mateFrom: number; mateTo: number };

type ScanTok =
  | { kind: "braceOpen" | "braceClose" | "parenOpen" | "parenClose" | "bracketOpen" | "bracketClose"; from: number; to: number }
  | { kind: "openWord" | "closeWord"; from: number; to: number; word: string };

function skipString(text: string, i: number, quote: '"' | "'"): number {
  i++;
  while (i < text.length) {
    if (text[i] === "\\" && i + 1 < text.length) {
      i += 2;
      continue;
    }
    if (text[i] === quote) return i + 1;
    i++;
  }
  return text.length;
}

function tokenize(text: string): ScanTok[] {
  const out: ScanTok[] = [];
  let i = 0;
  while (i < text.length) {
    const c = text[i];
    const n = text[i + 1];

    if (c === "/" && n === "/") {
      while (i < text.length && text[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && n === "*") {
      i += 2;
      while (i + 1 < text.length && !(text[i] === "*" && text[i + 1] === "/")) i++;
      i = Math.min(text.length, i + 2);
      continue;
    }
    if (c === '"' || c === "'") {
      i = skipString(text, i, c);
      continue;
    }

    if (c === "{") {
      out.push({ kind: "braceOpen", from: i, to: i + 1 });
      i++;
      continue;
    }
    if (c === "}") {
      out.push({ kind: "braceClose", from: i, to: i + 1 });
      i++;
      continue;
    }
    if (c === "(") {
      out.push({ kind: "parenOpen", from: i, to: i + 1 });
      i++;
      continue;
    }
    if (c === ")") {
      out.push({ kind: "parenClose", from: i, to: i + 1 });
      i++;
      continue;
    }
    if (c === "[") {
      out.push({ kind: "bracketOpen", from: i, to: i + 1 });
      i++;
      continue;
    }
    if (c === "]") {
      out.push({ kind: "bracketClose", from: i, to: i + 1 });
      i++;
      continue;
    }

    const rest = text.slice(i);
    const wm = WORD_RE.exec(rest);
    if (wm) {
      const word = wm[0];
      const from = i;
      const to = i + word.length;
      if (word === "סוף") {
        out.push({ kind: "closeWord", from, to, word });
      } else if (YOD_BLOCK_OPENERS.has(word)) {
        out.push({ kind: "openWord", from, to, word });
      }
      i = to;
      continue;
    }
    i++;
  }
  return out;
}

/** בונה מפת זוגות: ממיקום התחלה/סוף אל הצד השני. */
function buildPairMap(text: string): Map<number, PairHit> {
  const map = new Map<number, PairHit>();
  const toks = tokenize(text);

  type Frame =
    | { type: "brace"; from: number; to: number }
    | { type: "paren"; from: number; to: number }
    | { type: "bracket"; from: number; to: number }
    | { type: "word"; from: number; to: number };

  const stack: Frame[] = [];
  let braceDepth = 0;
  let parenDepth = 0;
  let bracketDepth = 0;

  const link = (aFrom: number, aTo: number, bFrom: number, bTo: number) => {
    const left: PairHit = { from: aFrom, to: aTo, mateFrom: bFrom, mateTo: bTo };
    const right: PairHit = { from: bFrom, to: bTo, mateFrom: aFrom, mateTo: aTo };
    map.set(aFrom, left);
    map.set(bFrom, right);
  };

  for (const t of toks) {
    if (t.kind === "braceOpen") {
      // בסגנון «אם (…) { }» הסוגריים מחליפים את «סוף»
      if (stack.length && stack[stack.length - 1].type === "word") {
        stack.pop();
      }
      stack.push({ type: "brace", from: t.from, to: t.to });
      braceDepth++;
      continue;
    }
    if (t.kind === "braceClose") {
      for (let s = stack.length - 1; s >= 0; s--) {
        if (stack[s].type === "brace") {
          const open = stack[s];
          stack.splice(s, 1);
          link(open.from, open.to, t.from, t.to);
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
          link(open.from, open.to, t.from, t.to);
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
          link(open.from, open.to, t.from, t.to);
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
          link(open.from, open.to, t.from, t.to);
          break;
        }
      }
    }
  }

  return map;
}

function hitAt(map: Map<number, PairHit>, pos: number, docLen: number, text: string): PairHit | null {
  // קודם מילה/טוקן ארוך שמכיל את הסמן (סוף, אם, …)
  for (const [, h] of map) {
    if (pos >= h.from && pos < h.to && h.to - h.from > 1) return h;
    if (pos >= h.mateFrom && pos < h.mateTo && h.mateTo - h.mateFrom > 1) {
      return { from: h.mateFrom, to: h.mateTo, mateFrom: h.from, mateTo: h.to };
    }
  }
  for (const [, h] of map) {
    if (pos === h.to && h.to - h.from > 1) return h;
    if (pos === h.mateTo && h.mateTo - h.mateFrom > 1) {
      return { from: h.mateFrom, to: h.mateTo, mateFrom: h.from, mateTo: h.to };
    }
  }

  // סוגריים — רק כשהסמן על הסוגר או מיד אחריו (לא בתוך מילה כמו הדפס(סוף))
  const ch = pos < text.length ? text[pos] : "";
  const prev = pos > 0 ? text[pos - 1] : "";
  if ("(){}[]".includes(ch)) {
    const h = map.get(pos);
    if (h) return h;
  }
  if ("(){}[]".includes(prev) && (ch === "" || /\s/.test(ch) || "(){}[]".includes(ch))) {
    const h = map.get(pos - 1);
    if (h && h.to - h.from === 1) return h;
  }
  return null;
}

/** מוצא את הצד השני של הזוג ליד הסמן; null אם אין. */
export function findMatchingPairPos(text: string, pos: number): { from: number; to: number } | null {
  const map = buildPairMap(text);
  const hit = hitAt(map, pos, text.length, text);
  if (!hit) return null;
  return { from: hit.mateFrom, to: hit.mateTo };
}

/** פקודת CodeMirror: קופצת לזוג התואם. */
export function jumpToYodPair(view: EditorView): boolean {
  const head = view.state.selection.main.head;
  const text = view.state.doc.toString();
  const pair = findMatchingPairPos(text, head);
  if (pair) {
    view.dispatch({
      selection: { anchor: pair.from },
      scrollIntoView: true,
    });
    return true;
  }

  // גיבוי: התאמת סוגריים של CodeMirror
  for (const dir of [-1, 1] as const) {
    const matched = matchBrackets(view.state, head, dir);
    if (matched?.end) {
      const onStart = head >= matched.start.from && head <= matched.start.to;
      const jumpTo = onStart ? matched.end.from : matched.start.from;
      view.dispatch({
        selection: { anchor: jumpTo },
        scrollIntoView: true,
      });
      return true;
    }
  }
  return false;
}
