import { Prec, RangeSetBuilder, type Extension } from "@codemirror/state";
import {
  Decoration,
  type DecorationSet,
  Direction,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from "@codemirror/view";

/**
 * עורך עברית מקצועי (RTL + Unicode):
 * - סוגריים/גרשיים — Unicode בלי עטיפת LTR
 * - אנגלית/מספרים — בידוד LTR
 * - הערת בלוק סגורה — LTR חיצוני + RTL פנימי לעברית
 */
const isolateLTR = Decoration.mark({
  attributes: { style: "direction: ltr; unicode-bidi: isolate" },
  bidiIsolate: Direction.LTR,
});

const isolateRTL = Decoration.mark({
  attributes: { style: "direction: rtl; unicode-bidi: isolate" },
  bidiIsolate: Direction.RTL,
});

export type BidiRange = { from: number; to: number; dir: "ltr" | "rtl" };

function isHebrew(cp: number): boolean {
  return (cp >= 0x0590 && cp <= 0x05ff) || (cp >= 0xfb1d && cp <= 0xfb4f);
}

function isLatin(ch: string): boolean {
  return (ch >= "A" && ch <= "Z") || (ch >= "a" && ch <= "z");
}

function isDigit(ch: string): boolean {
  return ch >= "0" && ch <= "9";
}

function isIdentRest(ch: string): boolean {
  return isLatin(ch) || isDigit(ch) || ch === "_";
}

function pushHebrewRtl(out: BidiRange[], text: string, from: number, to: number) {
  let i = from;
  while (i < to) {
    if (!isHebrew(text.charCodeAt(i))) {
      i++;
      continue;
    }
    const start = i;
    i++;
    while (i < to && isHebrew(text.charCodeAt(i))) i++;
    out.push({ from: start, to: i, dir: "rtl" });
  }
}

export function findBidiRanges(text: string): BidiRange[] {
  const out: BidiRange[] = [];
  let i = 0;
  const n = text.length;

  while (i < n) {
    // הערת בלוק
    if (text[i] === "/" && i + 1 < n && text[i + 1] === "*") {
      let j = i + 2;
      let closed = false;
      while (j < n - 1) {
        if (text[j] === "*" && text[j + 1] === "/") {
          j += 2;
          closed = true;
          break;
        }
        j++;
      }
      if (closed) {
        // חיצוני שומר על /* … */ ; פנימי נותן הקלדה עברית טבעית
        out.push({ from: i, to: j, dir: "ltr" });
        pushHebrewRtl(out, text, i + 2, j - 2);
        i = j;
        continue;
      }
      out.push({ from: i, to: i + 2, dir: "ltr" });
      i += 2;
      continue;
    }

    if (text[i] === "*" && i + 1 < n && text[i + 1] === "/") {
      out.push({ from: i, to: i + 2, dir: "ltr" });
      i += 2;
      continue;
    }

    if (text[i] === "/" && i + 1 < n && text[i + 1] === "/") {
      out.push({ from: i, to: i + 2, dir: "ltr" });
      i += 2;
      continue;
    }

    if (isLatin(text[i])) {
      const start = i;
      i++;
      while (i < n && isIdentRest(text[i])) i++;
      out.push({ from: start, to: i, dir: "ltr" });
      continue;
    }

    if (isDigit(text[i])) {
      const start = i;
      i++;
      while (i < n && (isDigit(text[i]) || text[i] === "_")) i++;
      if (text[i] === "." && i + 1 < n && isDigit(text[i + 1])) {
        i++;
        while (i < n && (isDigit(text[i]) || text[i] === "_")) i++;
      }
      out.push({ from: start, to: i, dir: "ltr" });
      continue;
    }

    i++;
  }

  // ארוך לפני קצר באותה התחלה — לקינון
  out.sort((a, b) => a.from - b.from || b.to - a.to);
  return out;
}

function buildBidiDeco(view: EditorView): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>();
  const seen = new Set<number>();
  for (const { from, to } of view.visibleRanges) {
    let pos = from;
    while (pos <= to) {
      const line = view.state.doc.lineAt(pos);
      if (!seen.has(line.from)) {
        seen.add(line.from);
        for (const r of findBidiRanges(line.text)) {
          if (r.to <= r.from) continue;
          builder.add(
            line.from + r.from,
            line.from + r.to,
            r.dir === "rtl" ? isolateRTL : isolateLTR
          );
        }
      }
      if (line.to >= to) break;
      pos = line.to + 1;
    }
  }
  return builder.finish();
}

export function yodBidiIsolates(): Extension {
  return ViewPlugin.fromClass(
    class {
      isolates: DecorationSet;
      constructor(view: EditorView) {
        this.isolates = buildBidiDeco(view);
      }
      update(update: ViewUpdate) {
        if (update.docChanged || update.viewportChanged) {
          this.isolates = buildBidiDeco(update.view);
        }
      }
    },
    {
      provide: (plugin) => {
        const access = (view: EditorView) =>
          view.plugin(plugin)?.isolates ?? Decoration.none;
        return Prec.lowest([
          EditorView.decorations.of(access),
          EditorView.bidiIsolatedRanges.of(access),
        ]);
      },
    }
  );
}
