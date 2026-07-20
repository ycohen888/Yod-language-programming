import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import {
  Compartment,
  EditorState,
  Prec,
  StateEffect,
  StateField,
  type Extension,
} from "@codemirror/state";
import {
  EditorView,
  keymap,
  highlightActiveLine,
  lineNumbers,
  drawSelection,
  dropCursor,
  gutter,
  GutterMarker,
  rectangularSelection,
  crosshairCursor,
} from "@codemirror/view";
import {
  defaultKeymap,
  history,
  historyKeymap,
  indentWithTab,
  undo,
  redo,
  selectAll,
} from "@codemirror/commands";
import {
  search,
  searchKeymap,
  highlightSelectionMatches,
  openSearchPanel,
} from "@codemirror/search";
import { createYodSearchPanel } from "../lib/yodSearchPanel";
import {
  bracketMatching,
  indentUnit,
  indentOnInput,
  codeFolding,
  foldGutter,
  foldCode,
  unfoldCode,
  foldAll,
  unfoldAll,
} from "@codemirror/language";
import {
  closeBrackets,
  closeBracketsKeymap,
  nextSnippetField,
  prevSnippetField,
  clearSnippet,
} from "@codemirror/autocomplete";
import { yodFoldService, yodIndentService } from "../lib/yodCodeStructure";
import { yodStreamLanguage } from "../lib/yodCmLanguage";
import {
  makeSyntaxHighlighting,
  paletteForTheme,
  type EditorPalette,
} from "../lib/yodHighlightStyle";
import { DEFAULT_SETTINGS } from "../lib/settings";
import type { Settings } from "../types";
import { yodAutocompletion } from "../lib/yodCompletion";
import { yodBidiIsolates } from "../lib/yodBidi";
import {
  setEditorDiagnostics,
  yodDiagnosticsExt,
  type EditorDiagnostic,
} from "../lib/yodDiagnostics";
import { setLiveDocument, clearLiveDocument } from "../lib/projectIndex";
import { jumpToYodPair } from "../lib/yodBlockMatch";
import { isYodFamilyFile } from "../lib/paths";
import { languageSupportForPath } from "../lib/languageModes";

export type CodeEditorHandle = {
  openDocument: (key: string, text: string) => void;
  activate: (key: string) => void;
  closeDocument: (key: string) => void;
  getText: (key: string) => string | null;
  setText: (key: string, text: string) => void;
  revealLine: (line: number) => void;
  focus: () => void;
  layout: () => void;
  undo: () => void;
  redo: () => void;
  selectAll: () => void;
  openFind: () => void;
  openReplace: () => void;
  toggleComment: () => void;
  duplicateLine: () => void;
  jumpToMatchingPair: () => void;
  foldCode: () => void;
  unfoldCode: () => void;
  foldAll: () => void;
  unfoldAll: () => void;
  zoom: (delta: number) => void;
  zoomReset: () => void;
  getFontSize: () => number;
  getIdentifierAtCursor: () => string | null;
  setDiagnostics: (diags: EditorDiagnostic[]) => void;
  /** מסמן שורות כנקודות (סימניות) בשוליים, לכל טאב לפי מפתח. */
  setBookmarkLines: (key: string, lines: number[]) => void;
  /** מחיל העדפות (נושא/גופן/טאב/גלישה) על התצוגה בזמן אמת. */
  applySettings: (settings: Settings) => void;
};

type Props = {
  onDirty?: (key: string) => void;
  onCursor?: (line: number, col: number) => void;
  /** Ctrl/Cmd+לחיצה על מזהה → מעבר להגדרה */
  onGotoDefinition?: (word: string) => void;
  className?: string;
};

const MIN_FONT = 10;
const MAX_FONT = 28;

/** compartments לשינוי חי של מראה/טאב/גלישה בלי לבנות מחדש את המצב. */
const appearanceCompartment = new Compartment();
const tabCompartment = new Compartment();
const wrapCompartment = new Compartment();

/** גודל גופן אפקטיבי = בסיס מההגדרות + הזחת זום, מוגבל לטווח. */
function effectiveFont(settings: Settings, offset: number): number {
  return Math.min(MAX_FONT, Math.max(MIN_FONT, settings.fontSize + offset));
}

/** תוסף הזחת טאב (רוחב טאב + יחידת הזחה). */
function tabExtension(settings: Settings): Extension {
  return [EditorState.tabSize.of(settings.tabSize), indentUnit.of(" ".repeat(settings.tabSize))];
}

/** תוסף גלישת שורות (או ריק). */
function wrapExtension(settings: Settings): Extension {
  return settings.wordWrap ? EditorView.lineWrapping : [];
}

/** בונה את תוסף המראה (הדגשת תחביר + ערכת צבעים + גופן) לפי הגדרות + כיווניות הטאב. */
function appearanceExtension(key: string, settings: Settings, offset: number): Extension {
  const yodMode = key !== "__empty" && isYodFamilyFile(key);
  const dir = yodMode ? "rtl" : "ltr";
  const textAlign = yodMode ? "right" : "left";
  const p: EditorPalette = paletteForTheme(settings.theme);
  const fontSize = effectiveFont(settings, offset);
  return [
    makeSyntaxHighlighting(p),
    EditorView.theme(
      {
        "&": {
          height: "100%",
          width: "100%",
          fontSize: `${fontSize}px`,
          backgroundColor: p.background,
          color: p.foreground,
        },
        "&.cm-ctrl-nav .cm-content": {
          cursor: "pointer",
        },
        ".cm-scroller": {
          overflow: "auto",
          fontFamily: settings.fontFamily,
          direction: dir,
        },
        ".cm-content": {
          direction: dir,
          textAlign,
          caretColor: p.foreground,
          padding: "8px 0",
          color: p.foreground,
        },
        ".cm-line": {
          direction: dir,
          textAlign,
        },
        ".cm-gutters": {
          backgroundColor: p.background,
          color: p.lineNumber,
          border: "none",
          ...(yodMode
            ? { borderLeft: `1px solid ${p.gutterBorder}` }
            : { borderRight: `1px solid ${p.gutterBorder}` }),
        },
        ".cm-diag-gutter": {
          width: "14px",
          minWidth: "14px",
        },
        ".cm-diag-mark": {
          width: "8px",
          height: "8px",
          margin: "4px auto",
          borderRadius: "0",
        },
        ".cm-diag-error": { backgroundColor: "#f14c4c" },
        ".cm-diag-warning": { backgroundColor: "#cca700" },
        ".cm-diag-info": { backgroundColor: "#3794ff" },
        ".cm-diag-line-error": { backgroundColor: "rgba(241, 76, 76, 0.12)" },
        ".cm-diag-line-warning": { backgroundColor: "rgba(204, 167, 0, 0.10)" },
        ".cm-diag-line-info": { backgroundColor: "rgba(55, 148, 255, 0.08)" },
        ".cm-panels.cm-panels-top": {
          backgroundColor: "transparent",
          borderBottom: "none",
        },
        ".cm-searchMatch": { backgroundColor: "rgba(234, 92, 0, 0.33)" },
        ".cm-searchMatch.cm-searchMatch-selected": {
          backgroundColor: "rgba(234, 92, 0, 0.55)",
        },
        ".cm-activeLineGutter": { backgroundColor: p.activeLineGutter },
        // רקע חצי-שקוף כדי שצבע הסימון (selection) ייראה גם בשורה הפעילה
        ".cm-activeLine": { backgroundColor: p.activeLine },
        ".cm-bookmark-gutter": {
          width: "12px",
          minWidth: "12px",
        },
        ".cm-bookmark-mark": {
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100%",
        },
        ".cm-bookmark-mark::before": {
          content: '""',
          width: "7px",
          height: "7px",
          borderRadius: "50%",
          backgroundColor: "#e0a33e",
          boxShadow: "0 0 0 1px rgba(0,0,0,0.35)",
        },
        "&.cm-focused .cm-cursor": { borderLeftColor: p.foreground },
        "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
          backgroundColor: `${p.selection} !important`,
        },
      },
      { dark: p.dark }
    ),
  ];
}

/** תמיד מחזיר true — מונע מ־defaultKeymap להפעיל הזחה על Ctrl+[ / Ctrl+] */
function jumpToYodPairAlways(view: EditorView): boolean {
  jumpToYodPair(view);
  return true;
}

/** אפקט לעדכון רשימת שורות הנקודות (סימניות) שמסומנות בשוליים. */
const setBookmarkLinesEffect = StateEffect.define<number[]>();

/** שדה מצב עם קבוצת מספרי-השורות המסומנות כנקודות. */
const bookmarkLinesField = StateField.define<Set<number>>({
  create: () => new Set<number>(),
  update(value, tr) {
    for (const e of tr.effects) {
      if (e.is(setBookmarkLinesEffect)) return new Set(e.value);
    }
    return value;
  },
});

class BookmarkMarker extends GutterMarker {
  toDOM() {
    const el = document.createElement("span");
    el.className = "cm-bookmark-mark";
    el.title = "נקודת קוד";
    return el;
  }
}
const bookmarkMarker = new BookmarkMarker();

const bookmarkGutter = gutter({
  class: "cm-bookmark-gutter",
  lineMarker(view, line) {
    const set = view.state.field(bookmarkLinesField, false);
    if (!set || set.size === 0) return null;
    const n = view.state.doc.lineAt(line.from).number;
    return set.has(n) ? bookmarkMarker : null;
  },
  lineMarkerChange(update) {
    return update.transactions.some((tr) =>
      tr.effects.some((e) => e.is(setBookmarkLinesEffect))
    );
  },
});

const IDENT_RE = /[א-תA-Za-z_][א-תA-Za-z0-9_]*/g;

function identifierAt(text: string, col: number): string | null {
  IDENT_RE.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = IDENT_RE.exec(text))) {
    if (m.index <= col && col <= m.index + m[0].length) return m[0];
  }
  return null;
}

function buildExtensions(
  key: string,
  settings: Settings,
  fontOffset: number,
  onDirty?: (key: string) => void,
  onCursor?: (line: number, col: number) => void,
  onGotoDefinition?: (word: string) => void
): Extension {
  const yodMode = key !== "__empty" && isYodFamilyFile(key);
  const dir = yodMode ? "rtl" : "ltr";
  const lang = yodMode ? "he" : "en";

  const yodOnly: Extension[] = yodMode
    ? [
        ...yodDiagnosticsExt,
        yodStreamLanguage,
        yodBidiIsolates(),
        yodFoldService,
        yodIndentService,
        ...yodAutocompletion,
        Prec.highest(
          keymap.of([
            { key: "Ctrl-}", run: jumpToYodPairAlways },
            { key: "Ctrl-{", run: jumpToYodPairAlways },
            { key: "Ctrl-]", run: jumpToYodPairAlways },
            { key: "Ctrl-[", run: jumpToYodPairAlways },
            { key: "Ctrl-Shift-]", run: jumpToYodPairAlways },
            { key: "Ctrl-Shift-[", run: jumpToYodPairAlways },
            { key: "Mod-}", run: jumpToYodPairAlways },
            { key: "Mod-{", run: jumpToYodPairAlways },
            { key: "Mod-]", run: jumpToYodPairAlways },
            { key: "Mod-[", run: jumpToYodPairAlways },
            { key: "Mod-Shift-]", run: jumpToYodPairAlways },
            { key: "Mod-Shift-[", run: jumpToYodPairAlways },
          ])
        ),
      ]
    : (() => {
        const langExt = languageSupportForPath(key);
        const extras: Extension[] = [...yodDiagnosticsExt];
        if (langExt) extras.push(langExt);
        return extras;
      })();

  return [
    appearanceCompartment.of(appearanceExtension(key, settings, fontOffset)),
    tabCompartment.of(tabExtension(settings)),
    wrapCompartment.of(wrapExtension(settings)),
    lineNumbers(),
    bookmarkLinesField,
    bookmarkGutter,
    codeFolding(),
    foldGutter({
      markerDOM(open) {
        const el = document.createElement("span");
        el.className = "cm-yod-fold-marker";
        el.textContent = open ? "\u25be" : "\u25b8";
        return el;
      },
    }),
    highlightActiveLine(),
    drawSelection(),
    dropCursor(),
    EditorState.allowMultipleSelections.of(true),
    rectangularSelection(),
    crosshairCursor(),
    history(),
    bracketMatching(),
    closeBrackets(),
    indentOnInput(),
    highlightSelectionMatches(),
    search({ top: true, createPanel: createYodSearchPanel }),
    // ניווט בין שדות תבנית (snippet) — קודם ל-Tab של הזחה; מחזיר false כשאין שדה פעיל
    Prec.highest(
      keymap.of([
        { key: "Tab", run: nextSnippetField },
        { key: "Shift-Tab", run: prevSnippetField },
        { key: "Escape", run: clearSnippet },
      ])
    ),
    ...yodOnly,
    keymap.of([
      ...closeBracketsKeymap,
      indentWithTab,
      ...defaultKeymap,
      ...historyKeymap,
      ...searchKeymap,
    ]),
    EditorView.domEventHandlers({
      click(event, view) {
        if (!(event.ctrlKey || event.metaKey)) return false;
        const pos = view.posAtCoords({ x: event.clientX, y: event.clientY });
        if (pos == null) return false;
        const line = view.state.doc.lineAt(pos);
        const word = identifierAt(line.text, pos - line.from);
        if (!word) return false;
        event.preventDefault();
        onGotoDefinition?.(word);
        return true;
      },
    }),
    EditorView.editorAttributes.of({ dir, spellcheck: "false", lang }),
    EditorView.contentAttributes.of({ dir, lang }),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        if (onDirty) onDirty(key);
        setLiveDocument(key, update.state.doc.toString());
      }
      if (update.selectionSet && onCursor) {
        const head = update.state.selection.main.head;
        const line = update.state.doc.lineAt(head);
        onCursor(line.number, head - line.from + 1);
      }
    }),
  ];
}

function toggleLineComment(view: EditorView) {
  const { state } = view;
  const changes: { from: number; to: number; insert: string }[] = [];
  const lines = new Set<number>();
  for (const r of state.selection.ranges) {
    const fromLine = state.doc.lineAt(r.from).number;
    const toLine = state.doc.lineAt(r.to).number;
    for (let n = fromLine; n <= toLine; n++) lines.add(n);
  }
  let allCommented = true;
  for (const n of lines) {
    const line = state.doc.line(n);
    if (!/^\s*\/\//.test(line.text)) {
      allCommented = false;
      break;
    }
  }
  for (const n of lines) {
    const line = state.doc.line(n);
    if (allCommented) {
      const m = line.text.match(/^(\s*)\/\/\s?/);
      if (m) {
        changes.push({ from: line.from, to: line.from + m[0].length, insert: m[1] });
      }
    } else {
      const m = line.text.match(/^(\s*)/);
      const indent = m ? m[1] : "";
      changes.push({
        from: line.from + indent.length,
        to: line.from + indent.length,
        insert: "// ",
      });
    }
  }
  if (changes.length) view.dispatch({ changes });
}

function duplicateLine(view: EditorView) {
  const { state } = view;
  const line = state.doc.lineAt(state.selection.main.head);
  view.dispatch({
    changes: { from: line.to, insert: "\n" + line.text },
    selection: { anchor: line.to + 1 + (state.selection.main.head - line.from) },
  });
}

export const CodeEditor = forwardRef<CodeEditorHandle, Props>(function CodeEditor(
  { onDirty, onCursor, onGotoDefinition, className },
  ref
) {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const statesRef = useRef<Map<string, EditorState>>(new Map());
  const scrollRef = useRef<Map<string, StateEffect<unknown>>>(new Map());
  const bookmarkLinesRef = useRef<Map<string, number[]>>(new Map());
  const activeKeyRef = useRef<string | null>(null);
  const settingsRef = useRef<Settings>(DEFAULT_SETTINGS);
  const fontOffsetRef = useRef(0);
  const [contentDir, setContentDir] = useState<"rtl" | "ltr">("rtl");
  const dirtyFn = useRef(onDirty);
  const cursorFn = useRef(onCursor);
  const gotoFn = useRef(onGotoDefinition);
  dirtyFn.current = onDirty;
  cursorFn.current = onCursor;
  gotoFn.current = onGotoDefinition;

  const syncHostDir = (key: string | null) => {
    const rtl = !key || key === "__empty" || isYodFamilyFile(key);
    setContentDir(rtl ? "rtl" : "ltr");
  };

  // scrollSnapshot מייצר StateEffect שמשחזר את מיקום הגלילה בצורה יציבה
  // (מיקום הגלילה אינו חלק מ-EditorState, לכן הוא מתאפס בכל setState).
  const saveScroll = (key: string | null) => {
    const view = viewRef.current;
    if (!view || !key) return;
    scrollRef.current.set(key, view.scrollSnapshot());
  };

  const restoreScroll = (key: string) => {
    const view = viewRef.current;
    if (!view) return;
    const snap = scrollRef.current.get(key);
    if (snap) view.dispatch({ effects: snap });
  };

  /** מחיל מחדש מראה/טאב/גלישה על ה-view הפעיל בלי לבנות את המצב מחדש. */
  const reconfigureView = (view: EditorView, key: string | null) => {
    const s = settingsRef.current;
    view.dispatch({
      effects: [
        appearanceCompartment.reconfigure(appearanceExtension(key ?? "__empty", s, fontOffsetRef.current)),
        tabCompartment.reconfigure(tabExtension(s)),
        wrapCompartment.reconfigure(wrapExtension(s)),
      ],
    });
  };

  useEffect(() => {
    if (!hostRef.current || viewRef.current) return;
    const empty = EditorState.create({
      doc: "",
      extensions: buildExtensions(
        "__empty",
        settingsRef.current,
        fontOffsetRef.current,
        (k) => dirtyFn.current?.(k),
        (l, c) => cursorFn.current?.(l, c),
        (w) => gotoFn.current?.(w)
      ),
    });
    const view = new EditorView({ state: empty, parent: hostRef.current });
    viewRef.current = view;

    const setCtrlNav = (on: boolean) => {
      view.dom.classList.toggle("cm-ctrl-nav", on);
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Control" || e.key === "Meta") setCtrlNav(true);
    };
    const onKeyUp = (e: KeyboardEvent) => {
      if (e.key === "Control" || e.key === "Meta") setCtrlNav(false);
    };
    const onBlur = () => setCtrlNav(false);
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    window.addEventListener("blur", onBlur);

    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
      window.removeEventListener("blur", onBlur);
      view.destroy();
      viewRef.current = null;
      statesRef.current.clear();
    };
  }, []);

  useImperativeHandle(ref, () => ({
    openDocument(key, text) {
      const state = EditorState.create({
        doc: text,
        extensions: buildExtensions(
          key,
          settingsRef.current,
          fontOffsetRef.current,
          (k) => dirtyFn.current?.(k),
          (l, c) => cursorFn.current?.(l, c),
          (w) => gotoFn.current?.(w)
        ),
      });
      statesRef.current.set(key, state);
      setLiveDocument(key, text);
      syncHostDir(key);
    },
    activate(key) {
      const view = viewRef.current;
      if (!view) return;
      if (activeKeyRef.current === key) {
        view.focus();
        return;
      }
      if (activeKeyRef.current) {
        statesRef.current.set(activeKeyRef.current, view.state);
        setLiveDocument(activeKeyRef.current, view.state.doc.toString());
        saveScroll(activeKeyRef.current);
      }
      let state = statesRef.current.get(key);
      if (!state) {
        state = EditorState.create({
          doc: "",
          extensions: buildExtensions(
            key,
            settingsRef.current,
            fontOffsetRef.current,
            (k) => dirtyFn.current?.(k),
            (l, c) => cursorFn.current?.(l, c),
            (w) => gotoFn.current?.(w)
          ),
        });
        statesRef.current.set(key, state);
      }
      view.setState(state);
      activeKeyRef.current = key;
      // סנכרון מראה/טאב/גלישה להעדפות הנוכחיות (טאבים שנוצרו קודם עלולים להיות מיושנים)
      reconfigureView(view, key);
      setLiveDocument(key, state.doc.toString());
      syncHostDir(key);
      view.focus();
      const bmLines = bookmarkLinesRef.current.get(key) ?? [];
      view.dispatch({ effects: setBookmarkLinesEffect.of(bmLines) });
      restoreScroll(key);
    },
    closeDocument(key) {
      statesRef.current.delete(key);
      scrollRef.current.delete(key);
      bookmarkLinesRef.current.delete(key);
      clearLiveDocument(key);
      if (activeKeyRef.current === key) {
        activeKeyRef.current = null;
        syncHostDir(null);
        viewRef.current?.setState(
          EditorState.create({
            doc: "",
            extensions: buildExtensions("__empty", settingsRef.current, fontOffsetRef.current),
          })
        );
      }
    },
    getText(key) {
      if (activeKeyRef.current === key && viewRef.current) {
        return viewRef.current.state.doc.toString();
      }
      return statesRef.current.get(key)?.doc.toString() ?? null;
    },
    setText(key, text) {
      const view = viewRef.current;
      if (activeKeyRef.current === key && view) {
        view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
        statesRef.current.set(key, view.state);
        return;
      }
      const state = EditorState.create({
        doc: text,
        extensions: buildExtensions(
          key,
          settingsRef.current,
          fontOffsetRef.current,
          (k) => dirtyFn.current?.(k),
          (l, c) => cursorFn.current?.(l, c),
          (w) => gotoFn.current?.(w)
        ),
      });
      statesRef.current.set(key, state);
    },
    revealLine(line) {
      const view = viewRef.current;
      if (!view) return;
      const doc = view.state.doc;
      const n = Math.min(Math.max(1, line), doc.lines);
      const l = doc.line(n);
      view.dispatch({ selection: { anchor: l.from }, scrollIntoView: true });
    },
    getIdentifierAtCursor() {
      const view = viewRef.current;
      if (!view) return null;
      const pos = view.state.selection.main.head;
      const line = view.state.doc.lineAt(pos);
      return identifierAt(line.text, pos - line.from);
    },
    focus() {
      viewRef.current?.focus();
    },
    layout() {
      viewRef.current?.requestMeasure();
    },
    undo() {
      const v = viewRef.current;
      if (v) undo(v);
    },
    redo() {
      const v = viewRef.current;
      if (v) redo(v);
    },
    selectAll() {
      const v = viewRef.current;
      if (v) selectAll(v);
    },
    openFind() {
      const v = viewRef.current;
      if (!v) return;
      v.dom.classList.remove("cm-yod-prefer-replace");
      openSearchPanel(v);
    },
    openReplace() {
      const v = viewRef.current;
      if (!v) return;
      v.dom.classList.add("cm-yod-prefer-replace");
      openSearchPanel(v);
      requestAnimationFrame(() => {
        const panel = v.dom.querySelector(".cm-yod-search");
        panel?.classList.add("cm-yod-search--replace");
        const replace = panel?.querySelector<HTMLInputElement>("[data-yod-replace]");
        if (replace) {
          replace.focus();
          replace.select();
        }
        v.dom.classList.remove("cm-yod-prefer-replace");
      });
    },
    toggleComment() {
      const v = viewRef.current;
      if (v) toggleLineComment(v);
    },
    duplicateLine() {
      const v = viewRef.current;
      if (v) duplicateLine(v);
    },
    jumpToMatchingPair() {
      const v = viewRef.current;
      if (v) jumpToYodPair(v);
    },
    foldCode() {
      const v = viewRef.current;
      if (v) {
        foldCode(v);
        v.focus();
      }
    },
    unfoldCode() {
      const v = viewRef.current;
      if (v) {
        unfoldCode(v);
        v.focus();
      }
    },
    foldAll() {
      const v = viewRef.current;
      if (v) {
        foldAll(v);
        v.focus();
      }
    },
    unfoldAll() {
      const v = viewRef.current;
      if (v) {
        unfoldAll(v);
        v.focus();
      }
    },
    zoom(delta) {
      const s = settingsRef.current;
      const nextEff = effectiveFont(s, fontOffsetRef.current + delta);
      fontOffsetRef.current = nextEff - s.fontSize;
      const view = viewRef.current;
      if (view) reconfigureView(view, activeKeyRef.current);
    },
    zoomReset() {
      fontOffsetRef.current = 0;
      const view = viewRef.current;
      if (view) reconfigureView(view, activeKeyRef.current);
    },
    getFontSize() {
      return effectiveFont(settingsRef.current, fontOffsetRef.current);
    },
    setDiagnostics(diags) {
      const v = viewRef.current;
      if (!v) return;
      v.dispatch({ effects: setEditorDiagnostics.of(diags) });
    },
    setBookmarkLines(key, lines) {
      bookmarkLinesRef.current.set(key, lines);
      const v = viewRef.current;
      if (v && activeKeyRef.current === key) {
        v.dispatch({ effects: setBookmarkLinesEffect.of(lines) });
      }
    },
    applySettings(settings) {
      settingsRef.current = settings;
      const view = viewRef.current;
      if (view) reconfigureView(view, activeKeyRef.current);
    },
  }));

  return (
    <div
      ref={hostRef}
      className={`${className ?? "cm-host"}${contentDir === "ltr" ? " cm-host--ltr" : " cm-host--rtl"}`}
      dir={contentDir}
    />
  );
});
