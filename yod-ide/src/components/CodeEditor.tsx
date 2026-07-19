import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { EditorState, Prec, type Extension } from "@codemirror/state";
import {
  EditorView,
  keymap,
  highlightActiveLine,
  lineNumbers,
  drawSelection,
  dropCursor,
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
import { bracketMatching } from "@codemirror/language";
import { yodStreamLanguage } from "../lib/yodCmLanguage";
import { yodSyntaxHighlighting, yodPhpDarkColors } from "../lib/yodHighlightStyle";
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
  zoom: (delta: number) => void;
  zoomReset: () => void;
  getFontSize: () => number;
  getIdentifierAtCursor: () => string | null;
  setDiagnostics: (diags: EditorDiagnostic[]) => void;
};

type Props = {
  onDirty?: (key: string) => void;
  onCursor?: (line: number, col: number) => void;
  /** Ctrl/Cmd+לחיצה על מזהה → מעבר להגדרה */
  onGotoDefinition?: (word: string) => void;
  className?: string;
};

const BASE_FONT = 14;

/** תמיד מחזיר true — מונע מ־defaultKeymap להפעיל הזחה על Ctrl+[ / Ctrl+] */
function jumpToYodPairAlways(view: EditorView): boolean {
  jumpToYodPair(view);
  return true;
}

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
  fontSize: number,
  onDirty?: (key: string) => void,
  onCursor?: (line: number, col: number) => void,
  onGotoDefinition?: (word: string) => void
): Extension {
  const yodMode = key !== "__empty" && isYodFamilyFile(key);
  const dir = yodMode ? "rtl" : "ltr";
  const lang = yodMode ? "he" : "en";
  const textAlign = yodMode ? "right" : "left";

  const yodOnly: Extension[] = yodMode
    ? [
        ...yodDiagnosticsExt,
        yodSyntaxHighlighting,
        yodStreamLanguage,
        yodBidiIsolates(),
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
        const extras: Extension[] = [...yodDiagnosticsExt, yodSyntaxHighlighting];
        if (langExt) extras.push(langExt);
        return extras;
      })();

  return [
    lineNumbers(),
    highlightActiveLine(),
    drawSelection(),
    dropCursor(),
    history(),
    bracketMatching(),
    highlightSelectionMatches(),
    search({ top: true, createPanel: createYodSearchPanel }),
    ...yodOnly,
    keymap.of([indentWithTab, ...defaultKeymap, ...historyKeymap, ...searchKeymap]),
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
    EditorView.theme(
      {
        "&": {
          height: "100%",
          width: "100%",
          fontSize: `${fontSize}px`,
          backgroundColor: yodPhpDarkColors.background,
          color: yodPhpDarkColors.foreground,
        },
        "&.cm-ctrl-nav .cm-content": {
          cursor: "pointer",
        },
        ".cm-scroller": {
          overflow: "auto",
          fontFamily: 'Consolas, "Courier New", "Noto Sans Hebrew", monospace',
          direction: dir,
        },
        ".cm-content": {
          direction: dir,
          textAlign,
          caretColor: yodPhpDarkColors.foreground,
          padding: "8px 0",
          color: yodPhpDarkColors.foreground,
        },
        ".cm-line": {
          direction: dir,
          textAlign,
        },
        ".cm-gutters": {
          backgroundColor: yodPhpDarkColors.background,
          color: yodPhpDarkColors.lineNumber,
          border: "none",
          ...(yodMode
            ? { borderLeft: "1px solid #3c3c3c" }
            : { borderRight: "1px solid #3c3c3c" }),
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
        ".cm-activeLineGutter": { backgroundColor: "#2a2d2e" },
        ".cm-activeLine": { backgroundColor: "#2a2d2e" },
        "&.cm-focused .cm-cursor": { borderLeftColor: yodPhpDarkColors.foreground },
        "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
          backgroundColor: "#264f78 !important",
        },
      },
      { dark: true }
    ),
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
  const scrollRef = useRef<Map<string, { top: number; left: number }>>(new Map());
  const activeKeyRef = useRef<string | null>(null);
  const fontSizeRef = useRef(BASE_FONT);
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

  const saveScroll = (key: string | null) => {
    const view = viewRef.current;
    if (!view || !key) return;
    scrollRef.current.set(key, {
      top: view.scrollDOM.scrollTop,
      left: view.scrollDOM.scrollLeft,
    });
  };

  const restoreScroll = (key: string) => {
    const view = viewRef.current;
    if (!view) return;
    const pos = scrollRef.current.get(key);
    const apply = () => {
      const v = viewRef.current;
      if (!v) return;
      v.scrollDOM.scrollTop = pos ? pos.top : 0;
      v.scrollDOM.scrollLeft = pos ? pos.left : 0;
    };
    // מיד + אחרי מדידה מחדש, כי setState מרנדר מחדש את התוכן
    apply();
    view.requestMeasure({ read: () => null, write: apply });
  };

  const recreateActiveWithFont = (size: number) => {
    const view = viewRef.current;
    const key = activeKeyRef.current;
    if (!view || !key) return;
    const text = view.state.doc.toString();
    const sel = view.state.selection;
    saveScroll(key);
    const state = EditorState.create({
      doc: text,
      selection: sel,
      extensions: buildExtensions(
        key,
        size,
        (k) => dirtyFn.current?.(k),
        (l, c) => cursorFn.current?.(l, c),
        (w) => gotoFn.current?.(w)
      ),
    });
    statesRef.current.set(key, state);
    view.setState(state);
    restoreScroll(key);
  };

  useEffect(() => {
    if (!hostRef.current || viewRef.current) return;
    const empty = EditorState.create({
      doc: "",
      extensions: buildExtensions(
        "__empty",
        fontSizeRef.current,
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
          fontSizeRef.current,
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
            fontSizeRef.current,
            (k) => dirtyFn.current?.(k),
            (l, c) => cursorFn.current?.(l, c),
            (w) => gotoFn.current?.(w)
          ),
        });
        statesRef.current.set(key, state);
      }
      view.setState(state);
      activeKeyRef.current = key;
      setLiveDocument(key, state.doc.toString());
      syncHostDir(key);
      view.focus();
      restoreScroll(key);
    },
    closeDocument(key) {
      statesRef.current.delete(key);
      scrollRef.current.delete(key);
      clearLiveDocument(key);
      if (activeKeyRef.current === key) {
        activeKeyRef.current = null;
        syncHostDir(null);
        viewRef.current?.setState(
          EditorState.create({
            doc: "",
            extensions: buildExtensions("__empty", fontSizeRef.current),
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
          fontSizeRef.current,
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
    zoom(delta) {
      fontSizeRef.current = Math.min(28, Math.max(10, fontSizeRef.current + delta));
      recreateActiveWithFont(fontSizeRef.current);
    },
    zoomReset() {
      fontSizeRef.current = BASE_FONT;
      recreateActiveWithFont(BASE_FONT);
    },
    getFontSize() {
      return fontSizeRef.current;
    },
    setDiagnostics(diags) {
      const v = viewRef.current;
      if (!v) return;
      v.dispatch({ effects: setEditorDiagnostics.of(diags) });
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
