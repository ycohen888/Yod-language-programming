/**
 * מסדר קוד יוד מקומית (הזחות) כש־`yod סדר` לא זמין.
 * כשה־CLI קיים — העורך מעדיף אותו (פורמט מלא כמו בעורך Win32).
 */
const INDENT = "    ";

/** סופר { / } מחוץ למחרוזות והערות. */
function braceDelta(line: string): { closeAtStart: number; net: number } {
  let i = 0;
  let closeAtStart = 0;
  let seenCode = false;
  let net = 0;
  let inStr: '"' | "'" | null = null;
  let inLineComment = false;
  let inBlockComment = false;

  while (i < line.length) {
    const c = line[i];
    const next = line[i + 1];

    if (inLineComment) break;

    if (inBlockComment) {
      if (c === "*" && next === "/") {
        inBlockComment = false;
        i += 2;
        continue;
      }
      i++;
      continue;
    }

    if (inStr) {
      if (c === "\\" && i + 1 < line.length) {
        i += 2;
        continue;
      }
      if (c === inStr) inStr = null;
      i++;
      continue;
    }

    if (c === "/" && next === "/") {
      inLineComment = true;
      break;
    }
    if (c === "/" && next === "*") {
      inBlockComment = true;
      i += 2;
      continue;
    }
    if (c === '"' || c === "'") {
      inStr = c;
      seenCode = true;
      i++;
      continue;
    }

    if (c === "}") {
      if (!seenCode) closeAtStart++;
      net--;
      i++;
      continue;
    }
    if (c === "{") {
      seenCode = true;
      net++;
      i++;
      continue;
    }
    if (!/\s/.test(c)) seenCode = true;
    i++;
  }

  return { closeAtStart, net };
}

/** הזחה לפי סוגריים מסולסלים + ניקוי רווחים בסוף שורה. */
export function formatYodSource(src: string): string {
  const text = src.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const lines = text.split("\n");
  const out: string[] = [];
  let depth = 0;

  for (const raw of lines) {
    const trimmed = raw.replace(/[ \t]+$/g, "").trimEnd();
    const content = trimmed.trim();
    if (!content) {
      out.push("");
      continue;
    }

    const { closeAtStart, net } = braceDelta(content);
    const indentLevel = Math.max(0, depth - closeAtStart);
    out.push(INDENT.repeat(indentLevel) + content);
    depth = Math.max(0, depth + net);
  }

  while (out.length > 0 && out[out.length - 1] === "") out.pop();
  return out.join("\n") + (out.length ? "\n" : "");
}
