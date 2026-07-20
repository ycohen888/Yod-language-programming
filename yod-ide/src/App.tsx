import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { CodeEditor, type CodeEditorHandle } from "./components/CodeEditor";
import { Icon } from "./components/Icon";
import { MenuBar } from "./components/MenuBar";
import { Breadcrumbs } from "./components/Breadcrumbs";
import type { MenuEntry } from "./lib/menuConfig";
import { TreeContextMenu, type TreeCtxItem } from "./components/TreeContextMenu";
import { TreeInlineInput } from "./components/TreeInlineInput";
import { parseProblems, pathBase, pathDir, resolveFilePath, isYodFamilyFile, joinPath } from "./lib/paths";
import { formatYodSource } from "./lib/yodFormat";
import { buildProjectIndex, clearProjectIndex, getMergedSymbols, getProjectIndex, getSymbolsForFile, type ProjectSymbol } from "./lib/projectIndex";
import type { Bookmark, CommandItem, OpenTab, PanelKind, Problem, RecentEntry } from "./types";
import { SymbolTree } from "./components/SymbolTree";
import { clearRecent, getRecent, loadSession, pushRecent, saveSession } from "./lib/session";
import { SettingsModal } from "./components/SettingsModal";
import { DiffView } from "./components/DiffView";
import { TerminalPanel, type TerminalPanelHandle } from "./components/TerminalPanel";
import { DEFAULT_SETTINGS, loadSettings, saveSettings } from "./lib/settings";
import type { Settings } from "./types";
import "./styles/app.css";

let untitledSeq = 1;

type TreeInlineEdit =
  | { kind: "create-file" | "create-folder"; parentDir: string }
  | { kind: "rename"; targetPath: string; isDir: boolean };

type TreeCtxState = { x: number; y: number; path: string; isDir: boolean };
type TabCtxState = { x: number; y: number; key: string };
type CloseTabMode = "ask" | "save" | "discard";

type SidebarView = "files" | "outline" | "symbols";

const SHORTCUTS_TEXT =
  "עורך יוד — קיצורי מקלדת\n\n" +
  "קובץ ← פתח תיקייה (Ctrl+Shift+O)\n" +
  "הקובץ הראשי: התחל.יוד — ממנו מריצים (F5)\n\n" +
  "F5 / F6 / F7  הרץ תוכנית (חלון Windows עולה) / מכונה / בדוק\n" +
  "Shift+F5  הרץ בטרמינל המשולב · Ctrl+`  טרמינל · קלט אינטראקטיבי (stdin)\n" +
  "Ctrl+P  פתיחה מהירה · Ctrl+Shift+F  חיפוש בפרויקט\n" +
  "F12 / Ctrl+לחיצה  מעבר להגדרה · Shift+F12  הפניות\n" +
  "F1  פלטת פקודות · Shift+Alt+F  סדר קוד\n" +
  "Ctrl+Shift+P  ארוז ל־EXE\n" +
  "Ctrl+N  קובץ חדש בסייר · Ctrl+O / S  פתח / שמור\n" +
  "F2  שינוי שם · Delete  מחיקה בסייר\n" +
  "Ctrl+W  סגור טאב\n" +
  "לחיצה ימנית על טאב  סגור / אחרים / כולם / ושמור\n" +
  "Ctrl+Z / Y  בטל / בצע שוב\n" +
  "Ctrl+F / H  חיפוש / החלפה\n" +
  "Ctrl+G  מעבר לשורה\n" +
  "Ctrl+/  הערה · Ctrl+Shift+D  שכפול שורה\n" +
  "Ctrl+B  קבע נקודה בקוד (לשונית נקודות)\n" +
  "Ctrl+D  בחר את המופע הבא · Alt+לחיצה  סמן נוסף\n" +
  "Ctrl+Alt+[ / ]  כווץ / הרחב בלוק · הקלד פונקציה/אם/עבור ובחר תבנית (Tab)\n" +
  "Ctrl+,  הגדרות · תצוגה ← השוואה מול הדיסק\n" +
  "Ctrl± / Ctrl+0  גודל גופן";

export default function App() {
  const [root, setRoot] = useState<string | null>(null);
  const [tabs, setTabs] = useState<OpenTab[]>([]);
  const [activeKey, setActiveKey] = useState<string | null>(null);
  const [treeExpanded, setTreeExpanded] = useState<Record<string, boolean>>({});
  const [treeChildren, setTreeChildren] = useState<
    Record<string, { name: string; path: string; isDir: boolean }[]>
  >({});
  const [treeSelected, setTreeSelected] = useState<string | null>(null);
  const [treeEdit, setTreeEdit] = useState<TreeInlineEdit | null>(null);
  const [treeCtx, setTreeCtx] = useState<TreeCtxState | null>(null);
  const [tabCtx, setTabCtx] = useState<TabCtxState | null>(null);
  const [sidebarView, setSidebarView] = useState<SidebarView>("files");
  const [symbolTick, setSymbolTick] = useState(0);
  const [quickOpen, setQuickOpen] = useState(false);
  const [quickQ, setQuickQ] = useState("");
  const [quickIdx, setQuickIdx] = useState(0);
  const [findOpen, setFindOpen] = useState(false);
  const [findQ, setFindQ] = useState("");
  const [searchHits, setSearchHits] = useState<Problem[]>([]);
  const [status, setStatus] = useState("מוכן");
  const [cursor, setCursor] = useState({ line: 1, col: 1 });
  const [panel, setPanel] = useState<PanelKind>("output");
  const [output, setOutput] = useState("");
  const [problems, setProblems] = useState<Problem[]>([]);
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [recent, setRecent] = useState<RecentEntry[]>(() => getRecent());
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [paletteQ, setPaletteQ] = useState("");
  const [paletteIdx, setPaletteIdx] = useState(0);
  const [busy, setBusy] = useState(false);
  const [distExeExists, setDistExeExists] = useState(false);
  const [yodExe, setYodExe] = useState("");
  const [aboutOpen, setAboutOpen] = useState<{
    engineVer: string;
    ideVer: string;
  } | null>(null);
  const [emailCopied, setEmailCopied] = useState(false);
  const [settings, setSettings] = useState<Settings>(() => loadSettings());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [winMaximized, setWinMaximized] = useState(false);
  const [diffState, setDiffState] = useState<{
    title: string;
    path: string;
    diskText: string;
    bufferText: string;
    yod: boolean;
  } | null>(null);
  const [inputPrompt, setInputPrompt] = useState<{
    title: string;
    message: string;
    value: string;
    resolve: (v: string | null) => void;
  } | null>(null);
  const editorRef = useRef<CodeEditorHandle | null>(null);
  const terminalRef = useRef<TerminalPanelHandle | null>(null);
  const tabsRef = useRef(tabs);
  const activeKeyRef = useRef(activeKey);
  const rootRef = useRef(root);
  const treeSelectedRef = useRef(treeSelected);
  const cursorRef = useRef(cursor);
  const savingRef = useRef(false);
  const tabLinesRef = useRef<Record<string, number>>({});
  const openedFromArgRef = useRef(false);
  const readyToSaveRef = useRef(false);
  const settingsRef = useRef(settings);
  const autosaveTimerRef = useRef<Record<string, number>>({});

  tabsRef.current = tabs;
  activeKeyRef.current = activeKey;
  rootRef.current = root;
  treeSelectedRef.current = treeSelected;
  cursorRef.current = cursor;
  settingsRef.current = settings;

  const activeTab = tabs.find((t) => t.key === activeKey) ?? null;

  // החלת העדפות: נושא ה-IDE + מראה העורך + שמירה מתמשכת
  useEffect(() => {
    document.documentElement.dataset.theme = settings.theme;
    editorRef.current?.applySettings(settings);
    saveSettings(settings);
  }, [settings]);

  // כותרת החלון: "יוד - <תיקיית הפרויקט>" כשפתוח פרויקט, אחרת "יוד"
  useEffect(() => {
    document.title = root ? `יוד - ${pathBase(root)}` : "יוד";
  }, [root]);

  const askText = useCallback((title: string, message: string, initial: string) => {
    return new Promise<string | null>((resolve) => {
      setInputPrompt({ title, message, value: initial, resolve });
    });
  }, []);

  const showError = useCallback((msg: string) => {
    setStatus(msg);
    setOutput(msg);
    setPanel("output");
  }, []);

  const reindexProject = useCallback(async (dir: string | null) => {
    if (!dir) {
      clearProjectIndex();
      setSymbolTick((t) => t + 1);
      return;
    }
    try {
      await buildProjectIndex(dir, {
        readDir: (p) => window.yod.readDir(p),
        readFile: async (p) => {
          const r = await window.yod.readFile(p);
          return { text: r.text };
        },
      });
      setSymbolTick((t) => t + 1);
    } catch {
      /* השלמה תעבוד חלקית */
    }
  }, []);

  const loadProjectRoot = useCallback(
    async (dir: string) => {
      setRoot(dir);
      setTreeExpanded((prev) => ({ ...prev, [dir]: true }));
      try {
        const kids = await window.yod.readDir(dir);
        setTreeChildren((prev) => ({ ...prev, [dir]: kids }));
        setStatus(`פרויקט: ${pathBase(dir)}`);
        setRecent(pushRecent("folder", dir));
        void reindexProject(dir);
        return kids;
      } catch (e) {
        showError(`לא הצלחתי לפתוח תיקייה: ${e instanceof Error ? e.message : String(e)}`);
        return [];
      }
    },
    [showError, reindexProject]
  );

  const reloadDir = useCallback(
    async (dir: string) => {
      try {
        const kids = await window.yod.readDir(dir);
        setTreeChildren((prev) => ({ ...prev, [dir]: kids }));
        return kids;
      } catch (e) {
        showError(String(e));
        return [];
      }
    },
    [showError]
  );

  const refreshTree = useCallback(async () => {
    const r = rootRef.current;
    if (!r) return;
    try {
      setTreeExpanded((exp) => {
        const dirs = new Set<string>([r]);
        for (const [p, open] of Object.entries(exp)) {
          if (open) dirs.add(p);
        }
        void (async () => {
          const next: Record<string, { name: string; path: string; isDir: boolean }[]> = {};
          for (const d of dirs) {
            try {
              next[d] = await window.yod.readDir(d);
            } catch {
              /* תיקייה נמחקה */
            }
          }
          setTreeChildren(next);
          setStatus("העץ רוענן");
        })();
        return { ...exp, [r]: true };
      });
    } catch (e) {
      showError(String(e));
    }
  }, [showError]);

  const distExeDir = useCallback((projectRoot: string | null | undefined) => {
    if (!projectRoot) return null;
    return joinPath(projectRoot, "dist_exe");
  }, []);

  const refreshDistExeExists = useCallback(async () => {
    const dir = distExeDir(rootRef.current);
    if (!dir) {
      setDistExeExists(false);
      return false;
    }
    try {
      const ok = await window.yod.exists(dir);
      if (!ok) {
        setDistExeExists(false);
        return false;
      }
      const st = await window.yod.stat(dir);
      const isDir = !!st?.isDir;
      setDistExeExists(isDir);
      return isDir;
    } catch {
      setDistExeExists(false);
      return false;
    }
  }, [distExeDir]);

  const openDistExeLocation = useCallback(async () => {
    const dir = distExeDir(rootRef.current);
    if (!dir) {
      setStatus("אין תיקיית פרויקט פתוחה");
      return;
    }
    const exists = await refreshDistExeExists();
    if (!exists) {
      setStatus("עדיין אין dist_exe — ארוז ל־EXE קודם");
      return;
    }
    try {
      const kids = await window.yod.readDir(dir);
      const exe = kids.find((k) => !k.isDir && k.name.toLowerCase().endsWith(".exe"));
      if (exe) await window.yod.showItem(exe.path);
      else await window.yod.showItem(dir);
      setStatus("נפתח מיקום EXE");
    } catch (e) {
      showError(String(e));
    }
  }, [distExeDir, refreshDistExeExists, showError]);

  const resolveParentDir = useCallback(async (): Promise<string | null> => {
    const r = rootRef.current;
    if (!r) return null;
    const sel = treeSelectedRef.current;
    if (!sel) return r;
    try {
      const st = await window.yod.stat(sel);
      return st.isDir ? sel : pathDir(sel);
    } catch {
      return r;
    }
  }, []);

  const beginCreate = useCallback(
    async (kind: "create-file" | "create-folder", parentOverride?: string) => {
      const parent = parentOverride || (await resolveParentDir());
      if (!parent) {
        await window.yod.dialogPrompt({
          kind: "info",
          title: "סייר",
          message: "פתחו תיקייה קודם (קובץ ← פתח תיקייה).",
        });
        return;
      }
      setTreeExpanded((prev) => ({ ...prev, [parent]: true }));
      await reloadDir(parent);
      setTreeSelected(parent);
      setTreeEdit({ kind, parentDir: parent });
      setTreeCtx(null);
    },
    [resolveParentDir, reloadDir]
  );

  const beginRename = useCallback(
    async (targetPath?: string) => {
      const sel = targetPath || treeSelectedRef.current;
      if (!sel) {
        await window.yod.dialogPrompt({
          kind: "info",
          title: "שינוי שם",
          message: "בחרו קובץ או תיקייה בסייר.",
        });
        return;
      }
      try {
        const st = await window.yod.stat(sel);
        setTreeSelected(sel);
        setTreeEdit({ kind: "rename", targetPath: sel, isDir: st.isDir });
        setTreeCtx(null);
      } catch (e) {
        showError(String(e));
      }
    },
    [showError]
  );

  const cancelTreeEdit = useCallback(() => setTreeEdit(null), []);

  const treeEditRef = useRef(treeEdit);
  treeEditRef.current = treeEdit;

  const activateTab = useCallback((key: string) => {
    setActiveKey(key);
    requestAnimationFrame(() => editorRef.current?.activate(key));
  }, []);

  const openFile = useCallback(
    async (filePath: string) => {
      const existing = tabsRef.current.find((t) => t.path === filePath);
      if (existing) {
        activateTab(existing.key);
        return;
      }

      let resolved = filePath;
      try {
        const ok = await window.yod.exists(filePath);
        if (!ok) {
          const tab = tabsRef.current.find((t) => t.key === activeKeyRef.current);
          const found = await resolveFilePath(filePath, {
            root: rootRef.current,
            activeDir: tab?.path ? pathDir(tab.path) : rootRef.current,
            indexedFiles: getProjectIndex().files,
            exists: (p) => window.yod.exists(p),
          });
          if (!found) {
            showError(`הקובץ לא נמצא:\n${pathBase(filePath)}\n\nחיפשתי גם תחת רכיבים ואינדקס הפרויקט.`);
            return;
          }
          resolved = found;
          const already = tabsRef.current.find((t) => t.path === resolved);
          if (already) {
            activateTab(already.key);
            return;
          }
        }
      } catch {
        /* continue to read — may still fail */
      }

      setBusy(true);
      setStatus("טוען…");
      try {
        const { text } = await window.yod.readFile(resolved);
        const key = resolved;
        const tab: OpenTab = {
          key,
          path: resolved,
          title: pathBase(resolved),
          dirty: false,
          modelUri: key,
        };
        editorRef.current?.openDocument(key, text);
        setTabs((prev) => [...prev, tab]);
        setActiveKey(key);
        requestAnimationFrame(() => {
          editorRef.current?.activate(key);
          editorRef.current?.focus();
        });
        setRecent(pushRecent("file", resolved));
        setStatus(`נפתח: ${pathBase(resolved)}`);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        if (/ENOENT|no such file/i.test(msg)) {
          showError(`הקובץ לא נמצא:\n${pathBase(resolved)}`);
        } else {
          showError(msg);
        }
      } finally {
        setBusy(false);
      }
    },
    [activateTab, showError]
  );

  const navigateToSymbol = useCallback(
    async (sym: ProjectSymbol) => {
      const existing = tabsRef.current.find((t) => t.key === sym.file || t.path === sym.file);
      if (existing) {
        activateTab(existing.key);
      } else if (sym.file && !sym.file.startsWith("untitled:")) {
        await openFile(sym.file);
      } else if (sym.file) {
        activateTab(sym.file);
      }
      requestAnimationFrame(() => {
        editorRef.current?.revealLine(sym.line);
        editorRef.current?.focus();
      });
      setStatus(`${sym.kind}: ${sym.name} · שורה ${sym.line}`);
    },
    [activateTab, openFile]
  );

  const bookmarksStorageKey = useCallback(
    () => `yod-ide:bookmarks:${rootRef.current ?? "__no-root"}`,
    []
  );

  // טעינת נקודות שמורות בעת פתיחת/החלפת תיקיית פרויקט
  useEffect(() => {
    try {
      const raw = localStorage.getItem(`yod-ide:bookmarks:${root ?? "__no-root"}`);
      const parsed = raw ? (JSON.parse(raw) as Bookmark[]) : [];
      setBookmarks(Array.isArray(parsed) ? parsed : []);
    } catch {
      setBookmarks([]);
    }
  }, [root]);

  // שמירה אוטומטית של הנקודות
  useEffect(() => {
    try {
      localStorage.setItem(bookmarksStorageKey(), JSON.stringify(bookmarks));
    } catch {
      /* אחסון מלא/חסום — מתעלמים */
    }
  }, [bookmarks, bookmarksStorageKey]);

  const addBookmark = useCallback(() => {
    const key = activeKeyRef.current;
    const tab = tabsRef.current.find((t) => t.key === key);
    if (!key || !tab) {
      setStatus("אין קובץ פעיל לקביעת נקודה");
      return;
    }
    const line = cursorRef.current.line;
    const bm: Bookmark = {
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      key,
      path: tab.path,
      title: tab.title,
      line,
    };
    setBookmarks((prev) => {
      // מניעת כפילות מדויקת (אותו קובץ + אותה שורה)
      if (prev.some((b) => b.key === key && b.line === line)) return prev;
      return [...prev, bm];
    });
    setPanel("bookmarks");
    setStatus(`נקבעה נקודה: ${tab.title} · שורה ${line}`);
  }, []);

  const removeBookmark = useCallback((id: string) => {
    setBookmarks((prev) => prev.filter((b) => b.id !== id));
  }, []);

  const clearBookmarks = useCallback(() => {
    setBookmarks([]);
    setStatus("כל הנקודות נמחקו");
  }, []);

  // דחיפת שורות הנקודות לעורך (סימון בשוליים) לכל הטאבים הפתוחים
  useEffect(() => {
    const ed = editorRef.current;
    if (!ed) return;
    const byKey = new Map<string, number[]>();
    for (const t of tabs) byKey.set(t.key, []);
    for (const b of bookmarks) {
      const arr = byKey.get(b.key);
      if (arr) arr.push(b.line);
      else byKey.set(b.key, [b.line]);
    }
    for (const [key, lines] of byKey) ed.setBookmarkLines(key, lines);
  }, [bookmarks, tabs]);

  const gotoBookmark = useCallback(
    async (bm: Bookmark) => {
      const existing = tabsRef.current.find((t) => t.key === bm.key || t.path === bm.path);
      if (existing) {
        activateTab(existing.key);
      } else if (bm.path) {
        await openFile(bm.path);
      } else {
        setStatus(`הקובץ אינו פתוח: ${bm.title}`);
        return;
      }
      requestAnimationFrame(() => {
        editorRef.current?.revealLine(bm.line);
        editorRef.current?.focus();
      });
      setStatus(`נקודה: ${bm.title} · שורה ${bm.line}`);
    },
    [activateTab, openFile]
  );

  const gotoDefinitionFor = useCallback(
    async (word: string | null | undefined) => {
      if (!word) {
        setStatus("אין מזהה תחת הסמן");
        return;
      }
      const matches = getMergedSymbols().filter((s) => s.name === word);
      if (matches.length === 0) {
        setStatus(`לא נמצאה הגדרה ל־${word}`);
        return;
      }
      const tab = tabsRef.current.find((t) => t.key === activeKeyRef.current);
      const local =
        matches.find((s) => tab?.path && s.file === tab.path) ||
        matches.find((s) => s.file === tab?.key) ||
        matches.find((s) => s.live);
      await navigateToSymbol(local ?? matches[0]);
    },
    [navigateToSymbol]
  );

  const gotoDefinition = useCallback(async () => {
    await gotoDefinitionFor(editorRef.current?.getIdentifierAtCursor());
  }, [gotoDefinitionFor]);

  const findReferences = useCallback(async () => {
    const word = editorRef.current?.getIdentifierAtCursor();
    if (!word) {
      setStatus("אין מזהה תחת הסמן");
      return;
    }
    const hits: Problem[] = [];
    for (const s of getMergedSymbols()) {
      if (s.name === word) {
        hits.push({
          file: s.file,
          line: s.line,
          message: `הגדרה (${s.kind}): ${s.name}`,
          severity: "info",
        });
      }
    }
    const files = getProjectIndex().files;
    const needle = word;
    for (const f of files.slice(0, 200)) {
      try {
        const { text } = await window.yod.readFile(f);
        const lines = text.split(/\r?\n/);
        for (let i = 0; i < lines.length; i++) {
          if (!lines[i].includes(needle)) continue;
          if (hits.some((h) => h.file === f && h.line === i + 1)) continue;
          hits.push({
            file: f,
            line: i + 1,
            message: lines[i].trim().slice(0, 120) || `${needle} · שורה ${i + 1}`,
            severity: "info",
          });
        }
      } catch {
        /* ignore */
      }
    }
    setSearchHits(hits);
    setPanel("search");
    setStatus(hits.length ? `${hits.length} הפניות ל־${word}` : `לא נמצאו הפניות ל־${word}`);
  }, []);

  const runFindInFiles = useCallback(async (query: string) => {
    const q = query.trim();
    if (!q) return;
    const root = rootRef.current;
    if (!root) {
      setStatus("פתחו תיקייה קודם");
      return;
    }
    setBusy(true);
    setStatus("מחפש…");
    try {
      let files = getProjectIndex().files;
      if (!files.length) {
        await reindexProject(root);
        files = getProjectIndex().files;
      }
      const hits: Problem[] = [];
      for (const f of files.slice(0, 300)) {
        try {
          const { text } = await window.yod.readFile(f);
          const lines = text.split(/\r?\n/);
          for (let i = 0; i < lines.length; i++) {
            if (!lines[i].includes(q)) continue;
            hits.push({
              file: f,
              line: i + 1,
              message: lines[i].trim().slice(0, 140) || `שורה ${i + 1}`,
              severity: "info",
            });
            if (hits.length >= 500) break;
          }
        } catch {
          /* ignore */
        }
        if (hits.length >= 500) break;
      }
      setSearchHits(hits);
      setPanel("search");
      setFindOpen(false);
      setStatus(hits.length ? `${hits.length} תוצאות ל־"${q}"` : `אין תוצאות ל־"${q}"`);
    } finally {
      setBusy(false);
    }
  }, [reindexProject]);

  const commitTreeEdit = useCallback(
    async (name: string) => {
      const edit = treeEditRef.current;
      setTreeEdit(null);
      const trimmed = name.trim();
      if (!edit || !trimmed) return;
      if (/[\\/]/.test(trimmed) || trimmed === "." || trimmed === "..") {
        showError("שם קובץ לא חוקי");
        return;
      }
      try {
        if (edit.kind === "create-file") {
          const full = joinPath(edit.parentDir, trimmed);
          if (await window.yod.exists(full)) {
            showError(`כבר קיים: ${trimmed}`);
            return;
          }
          await window.yod.writeFile(full, "// קובץ יוד חדש\n");
          await reloadDir(edit.parentDir);
          setTreeSelected(full);
          await openFile(full);
          setStatus(`נוצר: ${trimmed}`);
        } else if (edit.kind === "create-folder") {
          const full = joinPath(edit.parentDir, trimmed);
          if (await window.yod.exists(full)) {
            showError(`כבר קיים: ${trimmed}`);
            return;
          }
          await window.yod.mkdir(full);
          await reloadDir(edit.parentDir);
          setTreeSelected(full);
          setTreeExpanded((prev) => ({ ...prev, [full]: true }));
          setStatus(`נוצרה תיקייה: ${trimmed}`);
        } else if (edit.kind === "rename") {
          if (trimmed === pathBase(edit.targetPath)) return;
          const dest = joinPath(pathDir(edit.targetPath), trimmed);
          if (await window.yod.exists(dest)) {
            showError(`כבר קיים: ${trimmed}`);
            return;
          }
          const textByKey = new Map<string, string>();
          for (const t of tabsRef.current) {
            if (!t.path) continue;
            if (
              t.path === edit.targetPath ||
              t.path.startsWith(edit.targetPath + "\\") ||
              t.path.startsWith(edit.targetPath + "/")
            ) {
              const txt = editorRef.current?.getText(t.key);
              if (txt != null) textByKey.set(t.path, txt);
            }
          }
          await window.yod.rename(edit.targetPath, dest);
          await reloadDir(pathDir(edit.targetPath));
          setTreeSelected(dest);
          setTabs((prev) => {
            const next: OpenTab[] = [];
            for (const t of prev) {
              if (!t.path) {
                next.push(t);
                continue;
              }
              const hit =
                t.path === edit.targetPath ||
                t.path.startsWith(edit.targetPath + "\\") ||
                t.path.startsWith(edit.targetPath + "/");
              if (!hit) {
                next.push(t);
                continue;
              }
              const newPath = dest + t.path.slice(edit.targetPath.length);
              const body = textByKey.get(t.path) ?? "";
              editorRef.current?.closeDocument(t.key);
              editorRef.current?.openDocument(newPath, body);
              next.push({
                ...t,
                key: newPath,
                path: newPath,
                title: pathBase(newPath),
                modelUri: newPath,
              });
              if (activeKeyRef.current === t.key) {
                setActiveKey(newPath);
                requestAnimationFrame(() => editorRef.current?.activate(newPath));
              }
            }
            return next;
          });
          setStatus(`שם חדש: ${trimmed}`);
        }
        void reindexProject(rootRef.current);
      } catch (e) {
        showError(e instanceof Error ? e.message : String(e));
      }
    },
    [showError, reloadDir, openFile, reindexProject]
  );

  const deleteTreeItem = useCallback(
    async (targetPath?: string) => {
      const sel = targetPath || treeSelectedRef.current;
      if (!sel) {
        await window.yod.dialogPrompt({
          kind: "info",
          title: "מחיקה",
          message: "בחרו קובץ או תיקייה למחיקה.",
        });
        return;
      }
      const ok = await window.yod.dialogPrompt({
        kind: "confirm",
        title: "מחיקה",
        message: `למחוק את ${pathBase(sel)}?`,
      });
      if (!ok) return;
      try {
        await window.yod.remove(sel);
        const parent = pathDir(sel);
        for (const t of [...tabsRef.current]) {
          if (
            t.path &&
            (t.path === sel || t.path.startsWith(sel + "\\") || t.path.startsWith(sel + "/"))
          ) {
            editorRef.current?.closeDocument(t.key);
          }
        }
        setTabs((prev) => {
          const next = prev.filter(
            (t) =>
              !t.path ||
              (t.path !== sel && !t.path.startsWith(sel + "\\") && !t.path.startsWith(sel + "/"))
          );
          if (activeKeyRef.current && !next.some((t) => t.key === activeKeyRef.current)) {
            const fallback = next[next.length - 1] ?? null;
            setActiveKey(fallback?.key ?? null);
            if (fallback) requestAnimationFrame(() => editorRef.current?.activate(fallback.key));
          }
          return next;
        });
        setTreeSelected(parent || rootRef.current);
        if (parent) await reloadDir(parent);
        else await refreshTree();
        void reindexProject(rootRef.current);
        setStatus(`נמחק: ${pathBase(sel)}`);
      } catch (e) {
        showError(String(e));
      }
    },
    [reloadDir, refreshTree, reindexProject, showError]
  );

  const openPath = useCallback(
    async (p: string) => {
      if (!p) return;
      try {
        const st = await window.yod.stat(p);
        if (st.isDir) {
          const kids = await loadProjectRoot(p);
          const start = kids.find(
            (k) => !k.isDir && (k.name === "התחל.יוד" || k.name === "התחלה.יוד")
          );
          if (start) await openFile(start.path);
          return;
        }
        await loadProjectRoot(pathDir(p));
        await openFile(p);
      } catch (e) {
        showError(e instanceof Error ? e.message : String(e));
      }
    },
    [loadProjectRoot, openFile, showError]
  );

  const newUntitled = useCallback(() => {
    const key = `untitled:${untitledSeq++}.יוד`;
    const tab: OpenTab = {
      key,
      path: null,
      title: `ללא־שם-${untitledSeq - 1}.יוד`,
      dirty: true,
      modelUri: key,
    };
    editorRef.current?.openDocument(
      key,
      `// קובץ יוד — ניתן לכלול מתוך התחל.יוד או קבצים אחרים\n// כלול "שם_הקובץ.יוד"\n`
    );
    setTabs((prev) => [...prev, tab]);
    setActiveKey(key);
    requestAnimationFrame(() => {
      editorRef.current?.activate(key);
      editorRef.current?.focus();
    });
  }, []);

  const saveTabByKey = useCallback(
    async (key: string, forceSaveAs = false): Promise<string | null> => {
      const tab = tabsRef.current.find((t) => t.key === key);
      if (!tab || savingRef.current) return null;
      const raw = editorRef.current?.getText(tab.key);
      if (raw == null) return null;
      let text = raw;
      // סידור-בשמירה: מפעיל את מסדר הקוד המקומי לקבצי יוד לפני הכתיבה
      if (settingsRef.current.formatOnSave && isYodFamilyFile(tab.path ?? tab.title)) {
        try {
          const formatted = formatYodSource(raw);
          const norm = (s: string) => s.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
          if (typeof formatted === "string" && norm(formatted) !== norm(raw)) {
            text = formatted;
            editorRef.current?.setText(tab.key, formatted);
          }
        } catch {
          /* אם הסידור נכשל — שומרים כמו שהוא */
        }
      }
      let target = forceSaveAs ? null : tab.path;
      if (!target) {
        target = await window.yod.saveFileDialog(tab.path || tab.title);
        if (!target) return null;
      }
      savingRef.current = true;
      try {
        await window.yod.writeFile(target, text);
        const newKey = target;
        if (newKey !== tab.key) {
          editorRef.current?.closeDocument(tab.key);
          editorRef.current?.openDocument(newKey, text);
        }
        setTabs((prev) => {
          const next = prev.map((t) =>
            t.key === tab.key
              ? { ...t, path: target, title: pathBase(target!), dirty: false, key: newKey, modelUri: newKey }
              : t
          );
          tabsRef.current = next;
          return next;
        });
        if (activeKeyRef.current === tab.key) {
          setActiveKey(newKey);
          activeKeyRef.current = newKey;
          requestAnimationFrame(() => editorRef.current?.activate(newKey));
        }
        setStatus(`נשמר: ${pathBase(target)}`);
        void reindexProject(rootRef.current);
        return newKey;
      } catch (e) {
        showError(e instanceof Error ? e.message : String(e));
        return null;
      } finally {
        savingRef.current = false;
      }
    },
    [showError, reindexProject]
  );

  const saveActive = useCallback(
    async (forceSaveAs = false) => {
      const key = activeKeyRef.current;
      if (!key) return;
      await saveTabByKey(key, forceSaveAs);
    },
    [saveTabByKey]
  );

  const closeTab = useCallback(
    async (key: string, mode: CloseTabMode = "ask") => {
      let workingKey = key;
      const tab = tabsRef.current.find((t) => t.key === workingKey);
      if (!tab) return;
      if (tab.dirty) {
        if (mode === "save") {
          const savedKey = await saveTabByKey(workingKey, false);
          if (!savedKey) return;
          workingKey = savedKey;
        } else if (mode === "ask") {
          const ok = await window.yod.dialogPrompt({
            kind: "confirm",
            title: "שמירה",
            message: `לשמור שינויים ב־${tab.title}?`,
          });
          if (ok === null) return;
          if (ok) {
            const savedKey = await saveTabByKey(workingKey, false);
            if (!savedKey) return;
            workingKey = savedKey;
          }
        }
      }
      if (!tabsRef.current.some((t) => t.key === workingKey)) return;
      editorRef.current?.closeDocument(workingKey);
      setTabs((prev) => {
        const next = prev.filter((t) => t.key !== workingKey);
        tabsRef.current = next;
        if (activeKeyRef.current === workingKey) {
          const idx = prev.findIndex((t) => t.key === workingKey);
          const fallback = next[Math.min(idx, next.length - 1)] ?? null;
          setActiveKey(fallback?.key ?? null);
          activeKeyRef.current = fallback?.key ?? null;
          requestAnimationFrame(() => {
            if (fallback) editorRef.current?.activate(fallback.key);
          });
        }
        return next;
      });
    },
    [saveTabByKey]
  );

  const closeOtherTabs = useCallback(
    async (keepKey: string, mode: CloseTabMode = "ask") => {
      const keys = tabsRef.current.filter((t) => t.key !== keepKey).map((t) => t.key);
      for (const k of keys) {
        await closeTab(k, mode);
      }
      if (tabsRef.current.some((t) => t.key === keepKey)) {
        activateTab(keepKey);
      }
    },
    [closeTab, activateTab]
  );

  const closeAllTabs = useCallback(
    async (mode: CloseTabMode = "ask") => {
      const keys = tabsRef.current.map((t) => t.key);
      for (const k of keys) {
        await closeTab(k, mode);
      }
    },
    [closeTab]
  );

  const runYodCmd = useCallback(
    async (args: string[], label: string) => {
      const tab = tabsRef.current.find((t) => t.key === activeKeyRef.current);
      const cmd = args[0] ?? "";
      const projectCmds = cmd === "הרץ" || cmd === "מכונה" || cmd === "ארוז";
      const projectRoot = rootRef.current;

      setBusy(true);
      setPanel("output");
      setStatus(label);
      setOutput(`${label}…\n`);
      try {
        if (tab?.dirty && tab.path) await saveActive(false);

        let cwd: string | undefined = tab?.path ? pathDir(tab.path) : projectRoot ?? undefined;
        let target: string | undefined = tab?.path ?? undefined;

        // עם תיקיית פרויקט — מריצים/אורזים דרך התחל.יוד (תיקייה → ResolveEntry)
        if (projectRoot && projectCmds) {
          cwd = projectRoot;
          target = projectRoot;
          const mainTab = tabsRef.current.find(
            (t) =>
              t.path &&
              (pathBase(t.path) === "התחל.יוד" || pathBase(t.path) === "התחלה.יוד") &&
              t.dirty
          );
          if (mainTab?.path) {
            const text = editorRef.current?.getText(mainTab.key);
            if (text != null) {
              await window.yod.writeFile(mainTab.path, text);
              setTabs((prevTabs) =>
                prevTabs.map((t) => (t.key === mainTab.key ? { ...t, dirty: false } : t))
              );
            }
          }
        } else if (cmd === "בדוק" && projectRoot && !tab?.path) {
          target = projectRoot;
          cwd = projectRoot;
        }

        const fullArgs = target ? [...args, target] : args;
        const res = await window.yod.runYod(fullArgs, cwd);
        const text = [res.stdout, res.stderr].filter(Boolean).join("\n");
        setOutput(`> ${res.exe} ${res.args.join(" ")}\n\n${text || "(אין פלט)"}\n\nקוד יציאה: ${res.code}`);
        const probs = parseProblems(text);
        setProblems(probs);
        if (probs.some((p) => p.severity === "error")) setPanel("problems");
        setStatus(res.code === 0 ? "הסתיים בהצלחה" : `הסתיים עם שגיאה (${res.code})`);
        if (cmd === "ארוז" && res.code === 0) {
          void refreshDistExeExists();
          void refreshTree();
        }
      } catch (e) {
        showError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [saveActive, showError, refreshDistExeExists, refreshTree]
  );

  // הרצת התוכנית דרך מנוע יוד (כמו פעם): חלון קונסול מוסתר, אבל חלונות GUI של «חלונות»
  // עולים כרגיל; פלט/שגיאות בפאנל «פלט». where="terminal" — הרצה בטרמינל המשולב (stdin).
  const runProgram = useCallback(
    async (
      kind: "project" | "current",
      mode: "הרץ" | "מכונה",
      where: "app" | "terminal" = "app"
    ) => {
      const exe = yodExe;
      if (!exe) {
        setStatus("לא נמצא מנוע יוד (yod)");
        return;
      }
      const tab = tabsRef.current.find((t) => t.key === activeKeyRef.current);
      if (tab?.dirty && tab.path) await saveActive(false);

      let cwd: string | undefined;
      let target: string | undefined;
      if (kind === "current") {
        if (!tab?.path || !isYodFamilyFile(tab.path)) {
          setStatus("אין קובץ יוד פעיל להרצה");
          return;
        }
        cwd = pathDir(tab.path);
        target = tab.path;
      } else {
        const projectRoot = rootRef.current;
        if (projectRoot) {
          cwd = projectRoot;
          target = projectRoot;
          const mainTab = tabsRef.current.find(
            (t) =>
              t.path &&
              (pathBase(t.path) === "התחל.יוד" || pathBase(t.path) === "התחלה.יוד") &&
              t.dirty
          );
          if (mainTab?.path) {
            const text = editorRef.current?.getText(mainTab.key);
            if (text != null) {
              await window.yod.writeFile(mainTab.path, text);
              setTabs((prev) => prev.map((t) => (t.key === mainTab.key ? { ...t, dirty: false } : t)));
            }
          }
        } else if (tab?.path && isYodFamilyFile(tab.path)) {
          cwd = pathDir(tab.path);
          target = tab.path;
        } else {
          setStatus("אין פרויקט או קובץ יוד להרצה");
          return;
        }
      }

      if (where === "terminal") {
        setPanel("terminal");
        setStatus(mode === "מכונה" ? "מכונה (טרמינל)" : "הרצה (טרמינל)");
        await terminalRef.current?.openAndRun({ exe, args: [mode, target as string], cwd });
        return;
      }

      // מפעיל את התוכנית עצמה — חלונות Windows עולים; פלט בפאנל
      setBusy(true);
      setPanel("output");
      setStatus(mode === "מכונה" ? "מכונה…" : "מריץ…");
      setOutput(`${mode}…\n`);
      try {
        const res = await window.yod.runYod([mode, target as string], cwd);
        const text = [res.stdout, res.stderr].filter(Boolean).join("\n");
        setOutput(`> ${res.exe} ${res.args.join(" ")}\n\n${text || "(אין פלט)"}\n\nקוד יציאה: ${res.code}`);
        const probs = parseProblems(text);
        setProblems(probs);
        if (probs.some((p) => p.severity === "error")) setPanel("problems");
        setStatus(res.code === 0 ? "הסתיים בהצלחה" : `הסתיים עם שגיאה (${res.code})`);
      } catch (e) {
        showError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [yodExe, saveActive, showError]
  );

  const handleMenu = useCallback(
    async (action: string) => {
      const ed = editorRef.current;
      if (action.startsWith("recent:open:")) {
        const p = action.slice("recent:open:".length);
        await openPath(p);
        return;
      }
      switch (action) {
        case "file.clearRecent":
          setRecent(clearRecent());
          break;
        case "file.new":
          if (rootRef.current) await beginCreate("create-file");
          else newUntitled();
          break;
        case "file.open": {
          const p = await window.yod.openFileDialog();
          if (p) await openPath(p);
          break;
        }
        case "file.openFolder": {
          const p = await window.yod.openFolder();
          if (p) await openPath(p);
          break;
        }
        case "file.closeFolder":
          setRoot(null);
          setTreeChildren({});
          setTreeExpanded({});
          setTreeSelected(null);
          setTreeEdit(null);
          setDistExeExists(false);
          clearProjectIndex();
          setStatus("נסגרה תיקייה");
          break;
        case "file.save":
          await saveActive(false);
          break;
        case "file.saveAs":
          await saveActive(true);
          break;
        case "file.closeTab":
          if (activeKeyRef.current) await closeTab(activeKeyRef.current);
          break;
        case "tree.refresh":
          await refreshTree();
          await reindexProject(rootRef.current);
          await refreshDistExeExists();
          break;
        case "tree.newFile":
          await beginCreate("create-file");
          break;
        case "tree.newFolder":
          await beginCreate("create-folder");
          break;
        case "tree.rename":
          await beginRename();
          break;
        case "tree.delete":
          await deleteTreeItem();
          break;
        case "tree.reveal": {
          const sel = treeSelectedRef.current || rootRef.current;
          if (sel) await window.yod.showItem(sel);
          break;
        }
        case "tree.open": {
          const sel = treeSelectedRef.current;
          if (!sel) break;
          try {
            const st = await window.yod.stat(sel);
            if (st.isDir) {
              setTreeExpanded((prev) => ({ ...prev, [sel]: true }));
              await reloadDir(sel);
            } else await openFile(sel);
          } catch (e) {
            showError(String(e));
          }
          break;
        }
        case "edit.undo":
          ed?.undo();
          break;
        case "edit.redo":
          ed?.redo();
          break;
        case "edit.selectAll":
          ed?.selectAll();
          break;
        case "edit.find":
          ed?.openFind();
          break;
        case "edit.replace":
          ed?.openReplace();
          break;
        case "edit.goto": {
          const raw = await askText("מעבר לשורה", "מספר שורה:", String(cursor.line));
          const n = raw ? Number(raw) : NaN;
          if (Number.isFinite(n) && n >= 1) ed?.revealLine(n);
          break;
        }
        case "edit.comment":
          ed?.toggleComment();
          break;
        case "edit.duplicate":
          ed?.duplicateLine();
          break;
        case "edit.matchPair":
          ed?.jumpToMatchingPair();
          break;
        case "run.interpreter":
          await runProgram("project", "הרץ");
          break;
        case "run.vm":
          await runProgram("project", "מכונה");
          break;
        case "run.current":
          await runProgram("current", "הרץ");
          break;
        case "run.interpreter.terminal":
          await runProgram("project", "הרץ", "terminal");
          break;
        case "run.current.terminal":
          await runProgram("current", "הרץ", "terminal");
          break;
        case "run.stop":
          terminalRef.current?.interruptActive();
          break;
        case "run.check":
          await runYodCmd(["בדוק"], "בודק");
          break;
        case "run.pack":
          await runYodCmd(["ארוז"], "אורז");
          break;
        case "run.openDistExe":
          await openDistExeLocation();
          break;
        case "view.format": {
          const key = activeKeyRef.current;
          const ed = editorRef.current;
          if (!key || !ed) break;
          const src = ed.getText(key);
          if (src == null) break;
          const tab = tabsRef.current.find((t) => t.key === key);
          let formatted: string | null = null;
          try {
            if (tab?.path) {
              if (tab.dirty) await saveActive(false);
              const res = await window.yod.runYod(["סדר", tab.path], pathDir(tab.path));
              const errText = `${res.stderr}\n${res.stdout}`;
              if (
                res.code === 0 &&
                res.stdout.trim() &&
                !/פקודה לא מוכרת|לא מוכרת/.test(errText)
              ) {
                formatted = res.stdout.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
              }
            }
          } catch {
            /* fallback מקומי */
          }
          if (formatted == null) formatted = formatYodSource(src);
          const norm = (s: string) => s.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
          if (norm(formatted) === norm(src)) {
            setStatus("הקוד כבר מסודר");
          } else {
            ed.setText(key, formatted);
            setTabs((prev) => prev.map((t) => (t.key === key ? { ...t, dirty: true } : t)));
            setStatus("הקוד סודר");
          }
          break;
        }
        case "view.settings":
          setSettingsOpen(true);
          break;
        case "view.diff": {
          const key = activeKeyRef.current;
          const tab = key ? tabsRef.current.find((t) => t.key === key) : null;
          if (!tab || !tab.path) {
            setStatus("השוואה זמינה רק לקובץ שמור");
            break;
          }
          const bufferText = editorRef.current?.getText(tab.key) ?? "";
          let diskText = "";
          try {
            const res = await window.yod.readFile(tab.path);
            diskText = res.text.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
          } catch (e) {
            showError(e instanceof Error ? e.message : String(e));
            break;
          }
          const normB = bufferText.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
          if (diskText === normB) {
            setStatus("אין הבדלים מול הגרסה שעל הדיסק");
            break;
          }
          setDiffState({
            title: tab.title,
            path: tab.path,
            diskText,
            bufferText: normB,
            yod: isYodFamilyFile(tab.path),
          });
          break;
        }
        case "view.sidebar.files":
          setSidebarView("files");
          break;
        case "view.sidebar.outline":
          setSidebarView("outline");
          break;
        case "view.sidebar.symbols":
          setSidebarView("symbols");
          break;
        case "view.quickOpen":
          setQuickOpen(true);
          setQuickQ("");
          setQuickIdx(0);
          setPaletteOpen(false);
          setFindOpen(false);
          break;
        case "view.findInFiles":
          setFindOpen(true);
          setFindQ("");
          setPaletteOpen(false);
          setQuickOpen(false);
          break;
        case "nav.gotoDef":
          await gotoDefinition();
          break;
        case "nav.findRefs":
          await findReferences();
          break;
        case "view.zoomIn":
          ed?.zoom(1);
          setStatus(`גופן ${ed?.getFontSize() ?? ""}`);
          break;
        case "view.zoomOut":
          ed?.zoom(-1);
          setStatus(`גופן ${ed?.getFontSize() ?? ""}`);
          break;
        case "view.zoomReset":
          ed?.zoomReset();
          setStatus("גופן 14");
          break;
        case "code.addBookmark":
          addBookmark();
          break;
        case "code.showBookmarks":
          setPanel("bookmarks");
          break;
        case "code.clearBookmarks":
          clearBookmarks();
          break;
        case "code.fold":
          ed?.foldCode();
          break;
        case "code.unfold":
          ed?.unfoldCode();
          break;
        case "code.foldAll":
          ed?.foldAll();
          break;
        case "code.unfoldAll":
          ed?.unfoldAll();
          break;
        case "view.problems":
          setPanel("problems");
          break;
        case "view.output":
          setPanel("output");
          break;
        case "view.terminal":
          setPanel("terminal");
          terminalRef.current?.focus();
          break;
        case "terminal.new":
          setPanel("terminal");
          terminalRef.current?.newSession("powershell");
          break;
        case "view.palette":
          setPaletteOpen(true);
          setPaletteQ("");
          setPaletteIdx(0);
          break;
        case "help.guide": {
          const res = await window.yod.openGuide();
          if (!res?.ok) {
            await window.yod.dialogPrompt({
              kind: "info",
              title: "מדריך",
              message: res?.error || "לא נמצא קובץ המדריך ליד העורך.",
            });
          }
          break;
        }
        case "help.shortcuts":
          await window.yod.dialogPrompt({
            kind: "info",
            title: "קיצורי מקלדת",
            message: "קיצורי מקלדת",
            detail: SHORTCUTS_TEXT,
          });
          break;
        case "help.about": {
          const paths = await window.yod.getPaths();
          let engineVer = "";
          try {
            const ver = await window.yod.runYod(["גרסה"], undefined);
            engineVer = (ver.stdout || "").trim().split(/\r?\n/)[0] || "";
          } catch {
            /* ignore */
          }
          setEmailCopied(false);
          setAboutOpen({
            engineVer: engineVer || `יוד ${paths.version}`,
            ideVer: paths.version,
          });
          break;
        }
        case "app.quit":
          await window.yod.quit();
          break;
        default:
          break;
      }
    },
    [
      newUntitled,
      openPath,
      openFile,
      saveActive,
      closeTab,
      refreshTree,
      reindexProject,
      beginCreate,
      beginRename,
      deleteTreeItem,
      reloadDir,
      askText,
      runYodCmd,
      runProgram,
      openDistExeLocation,
      refreshDistExeExists,
      gotoDefinition,
      findReferences,
      cursor.line,
      addBookmark,
      clearBookmarks,
      showError,
    ]
  );

  useEffect(() => {
    void refreshDistExeExists();
  }, [root, refreshDistExeExists]);

  useEffect(() => {
    const onFocus = () => {
      void refreshDistExeExists();
    };
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [refreshDistExeExists]);

  useEffect(() => {
    void window.yod.getPaths().then((p) => {
      setYodExe(p.yodExe);
    });
    void window.yod.windowIsMaximized().then(setWinMaximized);
    const offPath = window.yod.onOpenPath((p) => {
      openedFromArgRef.current = true;
      readyToSaveRef.current = true;
      void openPath(p);
    });
    const offMenu = window.yod.onMenu((action) => {
      void handleMenu(action);
    });
    const offMax = window.yod.onMaximized(setWinMaximized);
    return () => {
      offPath();
      offMenu();
      offMax();
    };
  }, [openPath, handleMenu]);

  // שחזור סשן: אם לא נפתח נתיב מ-argv, טוענים את הסשן האחרון (תיקייה + טאבים + סמן)
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      void (async () => {
        if (cancelled || openedFromArgRef.current) {
          readyToSaveRef.current = true;
          return;
        }
        const s = loadSession();
        if (!s || !s.root) {
          readyToSaveRef.current = true;
          return;
        }
        try {
          await loadProjectRoot(s.root);
          for (const p of s.tabs) {
            try {
              await openFile(p);
            } catch {
              /* קובץ חסר — מדלגים בשקט */
            }
          }
          tabLinesRef.current = { ...s.lines };
          const openPaths = tabsRef.current.filter((t) => t.path).map((t) => t.path as string);
          const active =
            s.activeKey && openPaths.includes(s.activeKey)
              ? s.activeKey
              : openPaths[openPaths.length - 1] ?? null;
          if (active) {
            activateTab(active);
            const line = s.lines?.[active];
            if (line) requestAnimationFrame(() => editorRef.current?.revealLine(line));
          }
        } catch {
          /* שחזור נכשל — ממשיכים ריק */
        } finally {
          readyToSaveRef.current = true;
        }
      })();
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // שמירת סשן (debounce) כשמשתנים תיקייה/טאבים/טאב פעיל/שורת סמן
  useEffect(() => {
    if (!readyToSaveRef.current) return;
    const handle = setTimeout(() => {
      const fileTabs = tabs.filter((t) => t.path).map((t) => t.path as string);
      const activeIsFile = tabs.some((t) => t.key === activeKey && t.path);
      if (activeKey && activeIsFile) tabLinesRef.current[activeKey] = cursor.line;
      const lines: Record<string, number> = {};
      for (const p of fileTabs) {
        if (tabLinesRef.current[p] != null) lines[p] = tabLinesRef.current[p];
      }
      saveSession({
        root,
        tabs: fileTabs,
        activeKey: activeIsFile ? activeKey : null,
        lines,
      });
    }, 400);
    return () => clearTimeout(handle);
  }, [root, tabs, activeKey, cursor.line]);

  // סימוני gutter בעורך לפי תוצאות F7 / הרצה
  useEffect(() => {
    const tab = tabs.find((t) => t.key === activeKey);
    const path = tab?.path;
    const base = path ? pathBase(path) : null;
    const diags = problems
      .filter((p) => {
        if (!p.line) return false;
        if (!p.file) return true;
        if (!base) return true;
        const fb = pathBase(p.file);
        return fb === base || (path != null && (path.endsWith(p.file) || path.includes(`\\${p.file}`) || path.includes(`/${p.file}`)));
      })
      .map((p) => ({
        line: p.line!,
        severity: p.severity,
        message: p.message,
      }));
    editorRef.current?.setDiagnostics(diags);
  }, [problems, activeKey, tabs]);

  const toggleDir = useCallback(
    async (dirPath: string) => {
      setTreeSelected(dirPath);
      setTreeExpanded((prev) => ({ ...prev, [dirPath]: !prev[dirPath] }));
      setTreeChildren((prev) => {
        if (prev[dirPath]) return prev;
        void window.yod
          .readDir(dirPath)
          .then((kids) => setTreeChildren((c) => ({ ...c, [dirPath]: kids })))
          .catch((e) => showError(String(e)));
        return prev;
      });
    },
    [showError]
  );

  const commands: CommandItem[] = useMemo(
    () => [
      { id: "open.folder", label: "פתיחת תיקייה…", keybinding: "Ctrl+Shift+O", run: () => handleMenu("file.openFolder") },
      { id: "open.file", label: "פתיחת קובץ…", keybinding: "Ctrl+O", run: () => handleMenu("file.open") },
      { id: "file.new", label: "קובץ חדש", keybinding: "Ctrl+N", run: () => handleMenu("file.new") },
      { id: "file.save", label: "שמירה", keybinding: "Ctrl+S", run: () => handleMenu("file.save") },
      { id: "yod.run", label: "הרץ תוכנית (התחל.יוד / פרויקט)", keybinding: "F5", run: () => handleMenu("run.interpreter") },
      { id: "yod.runCurrent", label: "הרץ קובץ נוכחי", run: () => handleMenu("run.current") },
      { id: "yod.vm", label: "הרץ תוכנית — מכונה/bytecode", keybinding: "F6", run: () => handleMenu("run.vm") },
      { id: "yod.runTerm", label: "הרץ בטרמינל המשולב", keybinding: "Shift+F5", run: () => handleMenu("run.interpreter.terminal") },
      { id: "yod.runCurrentTerm", label: "הרץ קובץ נוכחי בטרמינל", run: () => handleMenu("run.current.terminal") },
      { id: "yod.stop", label: "עצור הרצה (Ctrl+C בטרמינל)", run: () => handleMenu("run.stop") },
      { id: "yod.check", label: "בדיקת סגנון ותחביר", keybinding: "F7", run: () => handleMenu("run.check") },
      { id: "view.terminal", label: "טרמינל — הצג/מקד", keybinding: "Ctrl+`", run: () => handleMenu("view.terminal") },
      { id: "terminal.new", label: "טרמינל חדש (PowerShell)", run: () => handleMenu("terminal.new") },
      { id: "yod.pack", label: "ארוז ל־EXE", keybinding: "Ctrl+Shift+P", run: () => handleMenu("run.pack") },
      {
        id: "yod.openDistExe",
        label: "פתח מיקום EXE",
        run: () => handleMenu("run.openDistExe"),
      },
      { id: "code.addBookmark", label: "קבע נקודה בקוד", keybinding: "Ctrl+B", run: () => handleMenu("code.addBookmark") },
      { id: "code.showBookmarks", label: "נקודות — הצג לשונית", run: () => handleMenu("code.showBookmarks") },
      { id: "code.fold", label: "כווץ בלוק", keybinding: "Ctrl+Alt+[", run: () => handleMenu("code.fold") },
      { id: "code.unfold", label: "הרחב בלוק", keybinding: "Ctrl+Alt+]", run: () => handleMenu("code.unfold") },
      { id: "code.foldAll", label: "כווץ את כל הבלוקים", run: () => handleMenu("code.foldAll") },
      { id: "code.unfoldAll", label: "הרחב את כל הבלוקים", run: () => handleMenu("code.unfoldAll") },
      { id: "view.format", label: "סדר קוד", keybinding: "Shift+Alt+F", run: () => handleMenu("view.format") },
      { id: "view.diff", label: "השוואה מול הגרסה שעל הדיסק", run: () => handleMenu("view.diff") },
      { id: "view.settings", label: "הגדרות", keybinding: "Ctrl+,", run: () => handleMenu("view.settings") },
      { id: "edit.matchPair", label: "זוג תואם (התחלה/סוף / סוגריים)", keybinding: "Ctrl+}", run: () => handleMenu("edit.matchPair") },
      { id: "edit.replace", label: "החלפה…", keybinding: "Ctrl+H", run: () => handleMenu("edit.replace") },
      { id: "view.palette", label: "הצגת פלטת פקודות", keybinding: "F1", run: () => handleMenu("view.palette") },
    ],
    [handleMenu]
  );

  const filteredCommands = useMemo(() => {
    const q = paletteQ.trim().toLowerCase();
    if (!q) return commands;
    return commands.filter((c) => c.label.toLowerCase().includes(q) || c.id.includes(q));
  }, [commands, paletteQ]);

  // קיצורים — כי התפריט מותאם ואינו נייטיב
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.ctrlKey || e.metaKey;
      const key = e.key.toLowerCase();
      if (e.key === "F1") {
        e.preventDefault();
        void handleMenu("view.palette");
        return;
      }
      if (mod && e.key === ",") {
        e.preventDefault();
        void handleMenu("view.settings");
        return;
      }
      if (e.key === "F12") {
        e.preventDefault();
        void handleMenu(e.shiftKey ? "nav.findRefs" : "nav.gotoDef");
        return;
      }
      if (e.key === "F5") {
        e.preventDefault();
        void handleMenu(e.shiftKey ? "run.interpreter.terminal" : "run.interpreter");
        return;
      }
      if (e.key === "F6") {
        e.preventDefault();
        void handleMenu("run.vm");
        return;
      }
      if (e.key === "F7") {
        e.preventDefault();
        void handleMenu("run.check");
        return;
      }
      if (mod && (e.key === "`" || e.code === "Backquote")) {
        e.preventDefault();
        void handleMenu("view.terminal");
        return;
      }
      if (mod && e.altKey && (e.key === "[" || e.code === "BracketLeft")) {
        e.preventDefault();
        void handleMenu("code.fold");
        return;
      }
      if (mod && e.altKey && (e.key === "]" || e.code === "BracketRight")) {
        e.preventDefault();
        void handleMenu("code.unfold");
        return;
      }
      if (e.key === "Escape") {
        if (paletteOpen) setPaletteOpen(false);
        if (quickOpen) setQuickOpen(false);
        if (findOpen) setFindOpen(false);
        return;
      }
      if (e.key === "F2") {
        e.preventDefault();
        void beginRename();
        return;
      }
      if (
        e.key === "Delete" &&
        treeSelectedRef.current &&
        !(e.target as HTMLElement)?.closest("input, textarea, .cm-editor")
      ) {
        e.preventDefault();
        void deleteTreeItem();
        return;
      }
      if (!mod) return;
      if (e.altKey && e.shiftKey && key === "f") {
        e.preventDefault();
        void handleMenu("view.format");
        return;
      }
      if (e.shiftKey && key === "o") {
        e.preventDefault();
        void handleMenu("file.openFolder");
      } else if (e.shiftKey && key === "p") {
        e.preventDefault();
        void handleMenu("run.pack");
      } else if (e.shiftKey && key === "d") {
        e.preventDefault();
        void handleMenu("edit.duplicate");
      } else if (e.shiftKey && key === "f") {
        e.preventDefault();
        void handleMenu("view.findInFiles");
      } else if (key === "p") {
        e.preventDefault();
        void handleMenu("view.quickOpen");
      } else if (key === "n") {
        e.preventDefault();
        void handleMenu("file.new");
      } else if (key === "o") {
        e.preventDefault();
        void handleMenu("file.open");
      } else if (key === "s") {
        e.preventDefault();
        void handleMenu("file.save");
      } else if (key === "w") {
        e.preventDefault();
        void handleMenu("file.closeTab");
      } else if (key === "b") {
        e.preventDefault();
        void handleMenu("code.addBookmark");
      } else if (key === "z") {
        e.preventDefault();
        void handleMenu("edit.undo");
      } else if (key === "y") {
        e.preventDefault();
        void handleMenu("edit.redo");
      } else if (key === "a") {
        if ((e.target as HTMLElement)?.tagName === "INPUT") return;
        e.preventDefault();
        void handleMenu("edit.selectAll");
      } else if (key === "f") {
        e.preventDefault();
        void handleMenu("edit.find");
      } else if (key === "h") {
        e.preventDefault();
        void handleMenu("edit.replace");
      } else if (key === "g") {
        e.preventDefault();
        void handleMenu("edit.goto");
      } else if (key === "/" || e.code === "Slash") {
        e.preventDefault();
        void handleMenu("edit.comment");
      } else if (
        (key === "}" ||
          key === "{" ||
          key === "]" ||
          key === "[" ||
          e.code === "BracketRight" ||
          e.code === "BracketLeft") &&
        !(e.target as HTMLElement)?.closest("input, textarea")
      ) {
        // בעורך: CodeMirror כבר קופץ — קריאה נוספת כאן קופצת חזרה לצד ההפוך
        if ((e.target as HTMLElement)?.closest(".cm-editor, .cm-content, .cm-scroller")) {
          return;
        }
        e.preventDefault();
        e.stopPropagation();
        void handleMenu("edit.matchPair");
      } else if (key === "=" || key === "+") {
        e.preventDefault();
        void handleMenu("view.zoomIn");
      } else if (key === "-") {
        e.preventDefault();
        void handleMenu("view.zoomOut");
      } else if (key === "0") {
        e.preventDefault();
        void handleMenu("view.zoomReset");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [handleMenu, paletteOpen, quickOpen, findOpen, beginRename, deleteTreeItem]);

  const renderTree = (dirPath: string, depth: number): ReactNode => {
    const kids = treeChildren[dirPath] || [];
    const creatingHere =
      treeEdit &&
      (treeEdit.kind === "create-file" || treeEdit.kind === "create-folder") &&
      treeEdit.parentDir === dirPath;

    if (depth === 0 && kids.length === 0 && !creatingHere) {
      return <div className="tree-empty">התיקייה ריקה או בטעינה…</div>;
    }

    return (
      <>
        {creatingHere ? (
          <TreeInlineInput
            depth={depth}
            isDir={treeEdit.kind === "create-folder"}
            defaultValue={treeEdit.kind === "create-folder" ? "תיקייה" : "חדש.יוד"}
            onCommit={(n) => void commitTreeEdit(n)}
            onCancel={cancelTreeEdit}
          />
        ) : null}
        {kids.map((ent) => {
          const open = !!treeExpanded[ent.path];
          const isActive = activeTab?.path === ent.path || treeSelected === ent.path;
          const renaming =
            treeEdit?.kind === "rename" && treeEdit.targetPath === ent.path;
          return (
            <div key={ent.path}>
              {renaming ? (
                <TreeInlineInput
                  depth={depth}
                  isDir={ent.isDir}
                  defaultValue={ent.name}
                  onCommit={(n) => void commitTreeEdit(n)}
                  onCancel={cancelTreeEdit}
                />
              ) : (
                <button
                  type="button"
                  className={`tree-item${isActive ? " active" : ""}`}
                  style={{ paddingInlineStart: 8 + depth * 14 }}
                  onClick={() => {
                    setTreeSelected(ent.path);
                    if (ent.isDir) void toggleDir(ent.path);
                    else void openFile(ent.path);
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setTreeSelected(ent.path);
                    setTabCtx(null);
                    setTreeCtx({ x: e.clientX, y: e.clientY, path: ent.path, isDir: ent.isDir });
                  }}
                  title={ent.path}
                >
                  <span className="chev">
                    {ent.isDir ? (
                      <Icon name={open ? "expand_more" : "chevron_left"} size={18} />
                    ) : (
                      <Icon
                        name={ent.name.endsWith(".יוד") || ent.name.endsWith(".yod") ? "code" : "insert_drive_file"}
                        size={16}
                        className="file-kind"
                      />
                    )}
                  </span>
                  {ent.isDir ? (
                    <Icon name={open ? "folder_open" : "folder"} size={16} className="folder-kind" />
                  ) : null}
                  <span>{ent.name}</span>
                </button>
              )}
              {ent.isDir && open ? renderTree(ent.path, depth + 1) : null}
            </div>
          );
        })}
      </>
    );
  };

  const outlineSymbols = useMemo(() => {
    if (!activeTab) return [];
    const fileId = activeTab.path ?? activeTab.key;
    return getSymbolsForFile(fileId, activeTab.key);
  }, [activeTab, symbolTick]);

  const projectSymbols = useMemo(() => {
    void symbolTick;
    return getMergedSymbols().sort(
      (a, b) => a.name.localeCompare(b.name, "he") || a.line - b.line
    );
  }, [symbolTick, root]);

  const quickItems = useMemo(() => {
    const q = quickQ.trim().toLowerCase();
    type QI = { id: string; label: string; detail: string; run: () => void };
    const items: QI[] = [];
    const files = getProjectIndex().files;
    for (const f of files) {
      const base = pathBase(f);
      if (q && !base.toLowerCase().includes(q) && !f.toLowerCase().includes(q)) continue;
      items.push({
        id: `f:${f}`,
        label: base,
        detail: f,
        run: () => {
          setQuickOpen(false);
          void openFile(f);
        },
      });
    }
    for (const s of getMergedSymbols()) {
      if (q && !s.name.toLowerCase().includes(q) && !s.kind.includes(q)) continue;
      items.push({
        id: `s:${s.file}:${s.line}:${s.name}`,
        label: s.name,
        detail: `${s.kind} · ${pathBase(s.file)}:${s.line}`,
        run: () => {
          setQuickOpen(false);
          void navigateToSymbol(s);
        },
      });
    }
    return items.slice(0, 80);
  }, [quickQ, symbolTick, root, openFile, navigateToSymbol]);

  const disabledMenuActions = useMemo(
    () => (distExeExists ? undefined : new Set(["run.openDistExe"])),
    [distExeExists]
  );

  // פריטי "נפתחו לאחרונה" בתפריט קובץ
  const dynamicMenuItems = useMemo(() => {
    if (recent.length === 0) return undefined;
    const items: MenuEntry[] = [{ type: "separator" }];
    for (const e of recent) {
      items.push({
        label: pathBase(e.path) || e.path,
        action: `recent:open:${e.path}`,
        kb: e.kind === "folder" ? "תיקייה" : "קובץ",
      });
    }
    items.push({ type: "separator" });
    items.push({ label: "נקה רשימת אחרונים", action: "file.clearRecent" });
    return { file: items };
  }, [recent]);

  const sidebarTitle =
    sidebarView === "files" ? "סייר" : sidebarView === "outline" ? "ניתוח קובץ" : "סימבולי פרויקט";

  const tabCtxItems: TreeCtxItem[] = tabCtx
    ? [
        { type: "item", label: "סגור", action: "tab.close" },
        { type: "item", label: "סגור אחרים", action: "tab.closeOthers" },
        { type: "item", label: "סגור כולם", action: "tab.closeAll" },
        { type: "separator" },
        { type: "item", label: "סגור ושמור", action: "tab.closeSave" },
        { type: "item", label: "סגור כולם ושמור", action: "tab.closeAllSave" },
      ]
    : [];

  const ctxItems: TreeCtxItem[] = treeCtx
    ? [
        ...(treeCtx.isDir
          ? ([] as TreeCtxItem[])
          : ([{ type: "item", label: "פתח", action: "tree.open" }] as TreeCtxItem[])),
        { type: "item", label: "קובץ חדש", action: "tree.newFile" },
        { type: "item", label: "תיקייה חדשה", action: "tree.newFolder" },
        { type: "separator" },
        { type: "item", label: "שינוי שם", action: "tree.rename" },
        { type: "item", label: "מחק", action: "tree.delete", danger: true },
        { type: "separator" },
        { type: "item", label: "הצג בסייר Windows", action: "tree.reveal" },
      ]
    : [];

  return (
    <div className="app">
      <header className="titlebar titlebar-slim">
        <div className="brand" title="יוד">
          <img className="brand-icon" src={`${import.meta.env.BASE_URL}icon.png`} alt="" width={16} height={16} />
          <span className="brand-name">יוד</span>
        </div>
        <MenuBar
          onAction={(a) => void handleMenu(a)}
          disabledActions={disabledMenuActions}
          dynamicItems={dynamicMenuItems}
        />
        <div className="titlebar-project">
          {root ? pathBase(root) : "אין פרויקט"}
        </div>
        <div className="titlebar-win-controls" dir="rtl">
          <button
            type="button"
            className="win-ctrl"
            title="מזער"
            aria-label="מזער"
            onClick={() => void window.yod.windowMinimize()}
          >
            <Icon name="minimize" size={16} />
          </button>
          <button
            type="button"
            className="win-ctrl"
            title={winMaximized ? "שחזר" : "הגדל"}
            aria-label={winMaximized ? "שחזר" : "הגדל"}
            onClick={() => void window.yod.windowMaximizeToggle().then(setWinMaximized)}
          >
            <Icon name={winMaximized ? "filter_none" : "crop_square"} size={15} />
          </button>
          <button
            type="button"
            className="win-ctrl win-ctrl-close"
            title="סגור"
            aria-label="סגור"
            onClick={() => void window.yod.windowClose()}
          >
            <Icon name="close" size={17} />
          </button>
        </div>
      </header>

      <div className="workspace">
        <aside className="activity" aria-label="פעילות">
          <div className="activity-group" role="group" aria-label="ניווט">
            <button
              type="button"
              className={sidebarView === "files" ? "active" : ""}
              title="סייר קבצים"
              onClick={() => setSidebarView("files")}
            >
              <Icon name="folder" size={22} />
            </button>
            <button
              type="button"
              className={sidebarView === "outline" ? "active" : ""}
              title="ניתוח קובץ"
              onClick={() => setSidebarView("outline")}
            >
              <Icon name="list_alt" size={22} />
            </button>
            <button
              type="button"
              className={sidebarView === "symbols" ? "active" : ""}
              title="סימבולי פרויקט"
              onClick={() => setSidebarView("symbols")}
            >
              <Icon name="account_tree" size={22} />
            </button>
            <button type="button" title="פלטת פקודות (F1)" onClick={() => setPaletteOpen(true)}>
              <Icon name="search" size={22} />
            </button>
          </div>

          <div className="activity-sep" role="separator" />

          <div className="activity-group" role="group" aria-label="הרצה">
            <button
              type="button"
              className="activity-run"
              title="הרץ תוכנית — חלון Windows עולה (F5) · Shift+F5 בטרמינל"
              onClick={() => void handleMenu("run.interpreter")}
              disabled={!activeTab?.path && !root}
            >
              <Icon name="play_arrow" size={22} />
            </button>
            <button
              type="button"
              title="הרץ קובץ נוכחי — חלון Windows עולה"
              onClick={() => void handleMenu("run.current")}
              disabled={!isYodFamilyFile(activeTab?.path ?? activeTab?.title ?? "")}
            >
              <Icon name="play_circle" size={22} />
            </button>
            <button type="button" title="עצור טרמינל (Ctrl+C)" onClick={() => void handleMenu("run.stop")}>
              <Icon name="stop" size={22} />
            </button>
            <button type="button" title="טרמינל (Ctrl+`)" onClick={() => void handleMenu("view.terminal")}>
              <Icon name="terminal" size={22} />
            </button>
          </div>

          <div className="activity-sep" role="separator" />

          <div className="activity-group" role="group" aria-label="בנייה">
            <button
              type="button"
              title="בדיקה (F7)"
              onClick={() => void handleMenu("run.check")}
              disabled={!activeTab?.path && !root}
            >
              <Icon name="spellcheck" size={22} />
            </button>
            <button
              type="button"
              title="ארוז ל־EXE (Ctrl+Shift+P)"
              onClick={() => void handleMenu("run.pack")}
              disabled={!root && !activeTab?.path}
            >
              <Icon name="inventory_2" size={22} />
            </button>
            <button
              type="button"
              title={distExeExists ? "פתח מיקום EXE" : "פתח מיקום EXE (אין dist_exe עדיין)"}
              onClick={() => void handleMenu("run.openDistExe")}
              disabled={!distExeExists}
            >
              <Icon name="folder_open" size={22} />
            </button>
          </div>
        </aside>

        <aside className="sidebar">
          <div className="sidebar-header">
            <span>{sidebarTitle}</span>
            <div className="sidebar-actions">
              {sidebarView === "files" ? (
                <>
                  <button type="button" title="פתיחת תיקייה" onClick={() => void handleMenu("file.openFolder")}>
                    <Icon name="folder_open" size={18} />
                  </button>
                  <button type="button" title="קובץ חדש" onClick={() => void beginCreate("create-file")} disabled={!root}>
                    <Icon name="note_add" size={18} />
                  </button>
                  <button type="button" title="תיקייה חדשה" onClick={() => void beginCreate("create-folder")} disabled={!root}>
                    <Icon name="create_new_folder" size={18} />
                  </button>
                  <button type="button" title="רענון" onClick={() => void handleMenu("tree.refresh")}>
                    <Icon name="refresh" size={18} />
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  title="רענון סמלים"
                  onClick={() => {
                    void reindexProject(rootRef.current);
                    setSymbolTick((t) => t + 1);
                  }}
                >
                  <Icon name="refresh" size={18} />
                </button>
              )}
            </div>
          </div>
          {sidebarView === "files" ? (
            <div
              className="tree"
              onContextMenu={(e) => {
                if (!root) return;
                if ((e.target as HTMLElement).closest(".tree-item")) return;
                e.preventDefault();
                setTreeSelected(root);
                setTabCtx(null);
                setTreeCtx({ x: e.clientX, y: e.clientY, path: root, isDir: true });
              }}
            >
              {!root ? (
                <div className="tree-empty">
                  פתחו תיקיית פרויקט
                  <br />
                  <strong>קובץ ← פתח תיקייה</strong>
                </div>
              ) : (
                renderTree(root, 0)
              )}
            </div>
          ) : sidebarView === "outline" ? (
            <SymbolTree
              symbols={outlineSymbols}
              groupBy="kind"
              emptyText={
                !activeTab
                  ? "פתחו קובץ כדי לראות ניתוח."
                  : "אין מחלקות/פונקציות/משתנים בקובץ זה."
              }
              onNavigate={(s) => void navigateToSymbol(s)}
              activeFile={activeTab?.path ?? activeTab?.key ?? null}
              activeLine={cursor.line}
            />
          ) : (
            <SymbolTree
              symbols={projectSymbols}
              groupBy="kind"
              emptyText={!root ? "פתחו תיקיית פרויקט." : "לא נמצאו סמלים בפרויקט."}
              onNavigate={(s) => void navigateToSymbol(s)}
              activeFile={activeTab?.path ?? activeTab?.key ?? null}
              activeLine={cursor.line}
            />
          )}
        </aside>
        <section className="main-col">
          <div
            className="tabs"
            role="tablist"
            onContextMenu={(e) => {
              e.preventDefault();
            }}
          >
            {tabs.map((t) => (
              <div
                key={t.key}
                className={`tab${t.key === activeKey ? " active" : ""}`}
                role="tab"
                onClick={() => activateTab(t.key)}
                onContextMenu={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setTreeCtx(null);
                  setTabCtx({ x: e.clientX, y: e.clientY, key: t.key });
                }}
              >
                <button
                  type="button"
                  className="title"
                  onClick={() => activateTab(t.key)}
                  style={{ background: "none", border: "none", color: "inherit", padding: 0, pointerEvents: "none" }}
                >
                  {t.dirty ? <span className="dirty">● </span> : null}
                  {t.title}
                </button>
                <button
                  type="button"
                  className="close"
                  title="סגירה"
                  onClick={(e) => {
                    e.stopPropagation();
                    void closeTab(t.key);
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setTreeCtx(null);
                    setTabCtx({ x: e.clientX, y: e.clientY, key: t.key });
                  }}
                >
                  <Icon name="close" size={14} />
                </button>
              </div>
            ))}
          </div>

          <div className="breadcrumbs-slot">
            {activeTab ? (
              <Breadcrumbs
                root={root}
                path={activeTab.path}
                title={activeTab.title}
                tabKey={activeTab.key}
                cursorLine={cursor.line}
                refreshKey={symbolTick}
                onJumpSymbol={(s) => void navigateToSymbol(s)}
              />
            ) : null}
          </div>

          <div className="editor-wrap">
            {busy ? <div className="busy-bar" /> : null}
            {tabs.length === 0 ? (
              <div className="editor-empty">
                <div>
                  <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
                    <img
                      src={`${import.meta.env.BASE_URL}icon.png`}
                      alt=""
                      width={40}
                      height={40}
                      style={{ borderRadius: 8 }}
                    />
                    <div style={{ fontSize: 28, fontWeight: 700, color: "#fff" }}>יוד IDE</div>
                  </div>
                  השתמשו בתפריט העליון כמו בעורך הקלאסי
                  <br />
                  <kbd>Ctrl+Shift+O</kbd> תיקייה · <kbd>F5</kbd> הרצה · <kbd>F1</kbd> פקודות
                </div>
              </div>
            ) : null}
            <div
              className={`editor-surface${tabs.length === 0 ? " is-hidden" : ""}${
                activeTab && !isYodFamilyFile(activeTab.path ?? activeTab.title) ? " is-ltr" : ""
              }`}
              dir={activeTab && !isYodFamilyFile(activeTab.path ?? activeTab.title) ? "ltr" : "rtl"}
            >
              <CodeEditor
                ref={editorRef}
                className="cm-host"
                onGotoDefinition={(word) => {
                  void gotoDefinitionFor(word);
                }}
                onDirty={(key) => {
                  setTabs((prev) =>
                    prev.map((t) => (t.key === key && !t.dirty ? { ...t, dirty: true } : t))
                  );
                  setSymbolTick((t) => t + 1);
                  // שמירה אוטומטית (debounce) — רק לקבצים עם נתיב
                  if (settingsRef.current.autosave) {
                    const tabForSave = tabsRef.current.find((t) => t.key === key);
                    if (tabForSave?.path) {
                      const timers = autosaveTimerRef.current;
                      if (timers[key]) window.clearTimeout(timers[key]);
                      timers[key] = window.setTimeout(() => {
                        delete timers[key];
                        void saveTabByKey(key, false);
                      }, 1000);
                    }
                  }
                  // אחרי תיקון בעורך — מסירים סימוני שגיאה לקובץ הזה
                  const tab = tabsRef.current.find((t) => t.key === key);
                  const base = tab?.path ? pathBase(tab.path) : null;
                  if (key === activeKeyRef.current) {
                    editorRef.current?.setDiagnostics([]);
                  }
                  setProblems((prev) => {
                    if (prev.length === 0) return prev;
                    const next = prev.filter((p) => {
                      if (!p.file) return false;
                      if (!base) return true;
                      const fb = pathBase(p.file);
                      if (fb === base) return false;
                      if (
                        tab?.path &&
                        (tab.path.endsWith(p.file) ||
                          tab.path.includes(`\\${p.file}`) ||
                          tab.path.includes(`/${p.file}`))
                      ) {
                        return false;
                      }
                      return true;
                    });
                    return next.length === prev.length ? prev : next;
                  });
                }}
                onCursor={(line, col) => {
                  setCursor({ line, col });
                  if (activeKeyRef.current) tabLinesRef.current[activeKeyRef.current] = line;
                }}
              />
            </div>
          </div>

          <div className="panel">
            <div className="panel-tabs">
              <button type="button" className={panel === "problems" ? "active" : ""} onClick={() => setPanel("problems")}>
                <Icon name="error_outline" size={14} />
                <span>בעיות{problems.length ? ` (${problems.length})` : ""}</span>
              </button>
              <button type="button" className={panel === "search" ? "active" : ""} onClick={() => setPanel("search")}>
                <Icon name="search" size={14} />
                <span>חיפוש{searchHits.length ? ` (${searchHits.length})` : ""}</span>
              </button>
              <button type="button" className={panel === "bookmarks" ? "active" : ""} onClick={() => setPanel("bookmarks")}>
                <Icon name="push_pin" size={14} />
                <span>נקודות{bookmarks.length ? ` (${bookmarks.length})` : ""}</span>
              </button>
              <button type="button" className={panel === "output" ? "active" : ""} onClick={() => setPanel("output")}>
                <Icon name="description" size={14} />
                <span>פלט</span>
              </button>
              <button type="button" className={panel === "terminal" ? "active" : ""} onClick={() => void handleMenu("view.terminal")}>
                <Icon name="terminal" size={14} />
                <span>טרמינל</span>
              </button>
            </div>
            <div className="panel-body" style={panel === "terminal" ? { display: "none" } : undefined}>
              {panel === "bookmarks"
                ? bookmarks.length === 0
                  ? "אין נקודות. הצב את הסמן בשורה ולחץ 'קוד ← קבע נקודה' (Ctrl+B)."
                  : bookmarks.map((b, i) => (
                      <div key={b.id} className="bookmark-row">
                        <button
                          type="button"
                          className="bookmark-goto"
                          title={`${b.path ?? b.title} · שורה ${b.line}`}
                          onClick={() => void gotoBookmark(b)}
                        >
                          <span className="bookmark-num">{i + 1}:</span>
                          <span className="bookmark-label">
                            {b.title} , שורה {b.line}
                          </span>
                        </button>
                        <button
                          type="button"
                          className="bookmark-del"
                          title="מחק נקודה"
                          aria-label="מחק נקודה"
                          onClick={() => removeBookmark(b.id)}
                        >
                          <Icon name="delete" size={14} />
                        </button>
                      </div>
                    ))
                : panel === "output"
                ? output || "אין פלט עדיין."
                : panel === "search"
                  ? searchHits.length === 0
                    ? "אין תוצאות חיפוש. Ctrl+Shift+F או Shift+F12."
                    : searchHits.map((p, i) => (
                        <button
                          key={i}
                          type="button"
                          className={`problem-row ${p.severity}`}
                          onClick={() => {
                            if (p.file) {
                              void openFile(p.file).then(() => {
                                if (p.line) editorRef.current?.revealLine(p.line);
                              });
                            } else if (p.line) {
                              editorRef.current?.revealLine(p.line);
                            }
                          }}
                        >
                          {p.file ? `${pathBase(p.file)}${p.line ? `:${p.line}` : ""} — ` : ""}
                          {p.message}
                        </button>
                      ))
                  : problems.length === 0
                    ? "אין בעיות."
                    : problems.map((p, i) => (
                        <button
                          key={i}
                          type="button"
                          className={`problem-row ${p.severity}`}
                          onClick={() => {
                            if (p.file) {
                              void openFile(p.file).then(() => {
                                if (p.line) editorRef.current?.revealLine(p.line);
                              });
                            } else if (p.line) {
                              editorRef.current?.revealLine(p.line);
                            }
                          }}
                        >
                          {p.message}
                        </button>
                      ))}
            </div>
            <div
              className="panel-term-wrap"
              style={panel === "terminal" ? undefined : { display: "none" }}
            >
              <TerminalPanel
                ref={terminalRef}
                visible={panel === "terminal"}
                theme={settings.theme}
                fontFamily={settings.fontFamily}
                fontSize={settings.fontSize}
                defaultCwd={root ?? undefined}
              />
            </div>
          </div>
        </section>
      </div>

      <footer className="statusbar">
        <span>{status}</span>
        <div className="right">
          <span>
            שורה {cursor.line}, עמודה {cursor.col}
          </span>
          <span>
            UTF-8 ·{" "}
            {activeTab && !isYodFamilyFile(activeTab.path ?? activeTab.title) ? "LTR" : "RTL"}
          </span>
          <span title={yodExe}>{yodExe ? pathBase(yodExe) : "yod"}</span>
        </div>
      </footer>

      {treeCtx ? (
        <TreeContextMenu
          x={treeCtx.x}
          y={treeCtx.y}
          items={ctxItems}
          onClose={() => setTreeCtx(null)}
          onAction={(action) => {
            const ctx = treeCtx;
            setTreeCtx(null);
            if (!ctx) return;
            treeSelectedRef.current = ctx.path;
            setTreeSelected(ctx.path);
            void (async () => {
              if (action === "tree.newFile") {
                await beginCreate("create-file", ctx.isDir ? ctx.path : pathDir(ctx.path));
              } else if (action === "tree.newFolder") {
                await beginCreate("create-folder", ctx.isDir ? ctx.path : pathDir(ctx.path));
              } else if (action === "tree.rename") {
                await beginRename(ctx.path);
              } else if (action === "tree.delete") {
                await deleteTreeItem(ctx.path);
              } else {
                await handleMenu(action);
              }
            })();
          }}
        />
      ) : null}

      {tabCtx ? (
        <TreeContextMenu
          x={tabCtx.x}
          y={tabCtx.y}
          items={tabCtxItems}
          onClose={() => setTabCtx(null)}
          onAction={(action) => {
            const ctx = tabCtx;
            setTabCtx(null);
            if (!ctx) return;
            void (async () => {
              if (action === "tab.close") {
                await closeTab(ctx.key, "ask");
              } else if (action === "tab.closeOthers") {
                await closeOtherTabs(ctx.key, "ask");
              } else if (action === "tab.closeAll") {
                await closeAllTabs("ask");
              } else if (action === "tab.closeSave") {
                await closeTab(ctx.key, "save");
              } else if (action === "tab.closeAllSave") {
                await closeAllTabs("save");
              }
            })();
          }}
        />
      ) : null}

      {paletteOpen ? (
        <div
          className="palette-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setPaletteOpen(false);
          }}
        >
          <div className="palette" role="dialog" aria-label="פלטת פקודות">
            <input
              autoFocus
              placeholder="הקלידו פקודה…"
              value={paletteQ}
              onChange={(e) => {
                setPaletteQ(e.target.value);
                setPaletteIdx(0);
              }}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setPaletteIdx((i) => Math.min(i + 1, filteredCommands.length - 1));
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setPaletteIdx((i) => Math.max(i - 1, 0));
                } else if (e.key === "Enter") {
                  e.preventDefault();
                  const cmd = filteredCommands[paletteIdx];
                  if (cmd) {
                    setPaletteOpen(false);
                    void cmd.run();
                  }
                } else if (e.key === "Escape") {
                  setPaletteOpen(false);
                }
              }}
            />
            <div className="palette-list">
              {filteredCommands.map((c, i) => (
                <button
                  key={c.id}
                  type="button"
                  className={`palette-item${i === paletteIdx ? " active" : ""}`}
                  onMouseEnter={() => setPaletteIdx(i)}
                  onClick={() => {
                    setPaletteOpen(false);
                    void c.run();
                  }}
                >
                  <span>{c.label}</span>
                  {c.keybinding ? <span className="kb">{c.keybinding}</span> : null}
                </button>
              ))}
              {filteredCommands.length === 0 ? <div className="palette-item">אין תוצאות</div> : null}
            </div>
          </div>
        </div>
      ) : null}

      {quickOpen ? (
        <div
          className="palette-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setQuickOpen(false);
          }}
        >
          <div className="palette" role="dialog" aria-label="פתיחה מהירה">
            <input
              autoFocus
              placeholder="קובץ או סמל…"
              value={quickQ}
              onChange={(e) => {
                setQuickQ(e.target.value);
                setQuickIdx(0);
              }}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setQuickIdx((i) => Math.min(i + 1, quickItems.length - 1));
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setQuickIdx((i) => Math.max(i - 1, 0));
                } else if (e.key === "Enter") {
                  e.preventDefault();
                  const item = quickItems[quickIdx];
                  if (item) item.run();
                } else if (e.key === "Escape") {
                  setQuickOpen(false);
                }
              }}
            />
            <div className="palette-list">
              {quickItems.map((c, i) => (
                <button
                  key={c.id}
                  type="button"
                  className={`palette-item${i === quickIdx ? " active" : ""}`}
                  onMouseEnter={() => setQuickIdx(i)}
                  onClick={() => c.run()}
                >
                  <span>{c.label}</span>
                  <span className="detail">{c.detail}</span>
                </button>
              ))}
              {quickItems.length === 0 ? <div className="palette-item">אין תוצאות</div> : null}
            </div>
          </div>
        </div>
      ) : null}

      {findOpen ? (
        <div
          className="palette-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setFindOpen(false);
          }}
        >
          <div className="palette" role="dialog" aria-label="חיפוש בפרויקט">
            <input
              autoFocus
              placeholder="חיפוש בכל קבצי הפרויקט…"
              value={findQ}
              onChange={(e) => setFindQ(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void runFindInFiles(findQ);
                } else if (e.key === "Escape") {
                  setFindOpen(false);
                }
              }}
            />
            <div className="palette-list">
              <button type="button" className="palette-item active" onClick={() => void runFindInFiles(findQ)}>
                <span>חפש Enter</span>
                <span className="kb">Ctrl+Shift+F</span>
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {aboutOpen ? (
        <div
          className="palette-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setAboutOpen(null);
          }}
        >
          <div
            className="palette about-box"
            role="dialog"
            aria-label="אודות יוד"
            tabIndex={-1}
            onKeyDown={(e) => {
              if (e.key === "Escape") setAboutOpen(null);
            }}
            ref={(n) => n?.focus()}
          >
            <div className="prompt-title">אודות יוד</div>
            <div className="about-body">
              <div className="about-heading">שפת יוד</div>
              <div className="about-meta">
                {aboutOpen.engineVer}
                <br />
                עורך: יוד IDE {aboutOpen.ideVer}
              </div>
              <p className="about-text">
                יוד היא שפת תכנות מודרנית בעברית — תחביר מימין־לשמאל, ספריות מובנות,
                עורך בסגנון Visual Studio Code, ומכונה וירטואלית. נועדה לדוברי עברית
                שרוצים לפתח בלי תלות באנגלית.
              </p>
              <div className="about-author">יוסי.כ</div>
              <div className="about-email-row">
                <button
                  type="button"
                  className="about-email-link"
                  title="פתח Gmail לשליחה"
                  onClick={() => {
                    void window.yod.openExternal(
                      "https://mail.google.com/mail/?view=cm&fs=1&to=ycohen888@gmail.com"
                    );
                  }}
                >
                  ycohen888@gmail.com
                </button>
                <button
                  type="button"
                  className="about-copy-btn"
                  title="העתק כתובת"
                  onClick={() => {
                    void (async () => {
                      await window.yod.writeClipboard("ycohen888@gmail.com");
                      setEmailCopied(true);
                      window.setTimeout(() => setEmailCopied(false), 1800);
                    })();
                  }}
                >
                  {emailCopied ? "הועתק" : "העתק"}
                </button>
              </div>
            </div>
            <div className="prompt-actions">
              <button type="button" className="primary" onClick={() => setAboutOpen(null)}>
                סגור
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {settingsOpen ? (
        <SettingsModal
          settings={settings}
          onChange={setSettings}
          onReset={() => setSettings(DEFAULT_SETTINGS)}
          onClose={() => setSettingsOpen(false)}
        />
      ) : null}

      {diffState ? (
        <DiffView
          title={diffState.title}
          diskText={diffState.diskText}
          bufferText={diffState.bufferText}
          settings={settings}
          yod={diffState.yod}
          onApply={(text) => {
            const key = diffState.path;
            editorRef.current?.setText(key, text);
            if (activeKeyRef.current === key) editorRef.current?.activate(key);
            setTabs((prev) =>
              prev.map((t) => (t.key === key ? { ...t, dirty: true } : t))
            );
            setDiffState(null);
          }}
          onClose={() => setDiffState(null)}
        />
      ) : null}

      {inputPrompt ? (
        <div className="palette-backdrop">
          <div className="palette prompt-box" role="dialog">
            <div className="prompt-title">{inputPrompt.title}</div>
            <div className="prompt-msg">{inputPrompt.message}</div>
            <input
              autoFocus
              value={inputPrompt.value}
              onChange={(e) => setInputPrompt({ ...inputPrompt, value: e.target.value })}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  const v = inputPrompt.value.trim();
                  inputPrompt.resolve(v || null);
                  setInputPrompt(null);
                } else if (e.key === "Escape") {
                  inputPrompt.resolve(null);
                  setInputPrompt(null);
                }
              }}
            />
            <div className="prompt-actions">
              <button
                type="button"
                onClick={() => {
                  inputPrompt.resolve(null);
                  setInputPrompt(null);
                }}
              >
                ביטול
              </button>
              <button
                type="button"
                className="primary"
                onClick={() => {
                  const v = inputPrompt.value.trim();
                  inputPrompt.resolve(v || null);
                  setInputPrompt(null);
                }}
              >
                אישור
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
