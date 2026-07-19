import type { RecentEntry, SessionState } from "../types";

const SESSION_KEY = "yod-ide:session";
const RECENT_KEY = "yod-ide:recent";
const RECENT_MAX = 8;

/** טעינת הסשן האחרון (תיקייה + טאבים + סמן). */
export function loadSession(): SessionState | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as SessionState;
    if (!parsed || typeof parsed !== "object") return null;
    return {
      root: parsed.root ?? null,
      tabs: Array.isArray(parsed.tabs) ? parsed.tabs.filter((t) => typeof t === "string") : [],
      activeKey: parsed.activeKey ?? null,
      lines: parsed.lines && typeof parsed.lines === "object" ? parsed.lines : {},
    };
  } catch {
    return null;
  }
}

/** שמירת הסשן הנוכחי. */
export function saveSession(state: SessionState): void {
  try {
    localStorage.setItem(SESSION_KEY, JSON.stringify(state));
  } catch {
    /* אחסון חסום/מלא — מתעלמים */
  }
}

/** רשימת האחרונים (אחרון-ראשון). */
export function getRecent(): RecentEntry[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as RecentEntry[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (e) => e && typeof e.path === "string" && (e.kind === "folder" || e.kind === "file")
    );
  } catch {
    return [];
  }
}

/** הוספת פריט לרשימת האחרונים (מזיז לראש, ללא כפילויות, מוגבל ל-RECENT_MAX). */
export function pushRecent(kind: RecentEntry["kind"], path: string): RecentEntry[] {
  if (!path) return getRecent();
  const list = getRecent().filter((e) => !(e.kind === kind && e.path === path));
  list.unshift({ kind, path });
  const trimmed = list.slice(0, RECENT_MAX);
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(trimmed));
  } catch {
    /* מתעלמים */
  }
  return trimmed;
}

/** ניקוי רשימת האחרונים. */
export function clearRecent(): RecentEntry[] {
  try {
    localStorage.removeItem(RECENT_KEY);
  } catch {
    /* מתעלמים */
  }
  return [];
}
