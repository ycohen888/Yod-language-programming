import { useEffect, useRef } from "react";
import { MergeView } from "@codemirror/merge";
import { EditorState, type Extension } from "@codemirror/state";
import { EditorView, lineNumbers } from "@codemirror/view";
import { makeSyntaxHighlighting, paletteForTheme } from "../lib/yodHighlightStyle";
import { yodStreamLanguage } from "../lib/yodCmLanguage";
import { yodBidiIsolates } from "../lib/yodBidi";
import { languageSupportForPath } from "../lib/languageModes";
import type { Settings } from "../types";

type Props = {
  title: string;
  /** תוכן הגרסה שעל הדיסק (צד המקור, לקריאה) */
  diskText: string;
  /** תוכן הבאפר הנוכחי (הצד הניתן לעריכה) */
  bufferText: string;
  settings: Settings;
  yod: boolean;
  /** החזרת התוכן הערוך לעורך */
  onApply: (text: string) => void;
  onClose: () => void;
};

/** בונה את סט התוספים לצד בעורך ההשוואה. */
function sideExtensions(settings: Settings, yod: boolean, fileName: string, editable: boolean): Extension {
  const p = paletteForTheme(settings.theme);
  const dir = yod ? "rtl" : "ltr";
  const textAlign = yod ? "right" : "left";
  const langExt: Extension[] = yod
    ? [yodStreamLanguage, yodBidiIsolates()]
    : (() => {
        const l = languageSupportForPath(fileName);
        return l ? [l] : [];
      })();
  return [
    lineNumbers(),
    makeSyntaxHighlighting(p),
    ...langExt,
    EditorState.readOnly.of(!editable),
    EditorView.editable.of(editable),
    EditorView.editorAttributes.of({ dir, lang: yod ? "he" : "en" }),
    EditorView.contentAttributes.of({ dir, lang: yod ? "he" : "en" }),
    EditorView.theme(
      {
        "&": {
          fontSize: `${settings.fontSize}px`,
          backgroundColor: p.background,
          color: p.foreground,
        },
        ".cm-scroller": { fontFamily: settings.fontFamily, direction: dir },
        ".cm-content": { direction: dir, textAlign, caretColor: p.foreground },
        ".cm-line": { direction: dir, textAlign },
        ".cm-gutters": {
          backgroundColor: p.background,
          color: p.lineNumber,
          border: "none",
        },
      },
      { dark: p.dark }
    ),
  ];
}

/** תצוגת השוואה (Diff) — באפר נוכחי מול הגרסה שעל הדיסק, עם עריכה ומיזוג. */
export function DiffView({ title, diskText, bufferText, settings, yod, onApply, onClose }: Props) {
  const hostRef = useRef<HTMLDivElement>(null);
  const mergeRef = useRef<MergeView | null>(null);

  useEffect(() => {
    if (!hostRef.current || mergeRef.current) return;
    const mv = new MergeView({
      parent: hostRef.current,
      // a = דיסק (מקור, לקריאה); b = באפר (ניתן לעריכה)
      orientation: yod ? "b-a" : "a-b",
      revertControls: "a-to-b",
      highlightChanges: true,
      gutter: true,
      a: {
        doc: diskText,
        extensions: sideExtensions(settings, yod, title, false),
      },
      b: {
        doc: bufferText,
        extensions: sideExtensions(settings, yod, title, true),
      },
    });
    mergeRef.current = mv;
    return () => {
      mv.destroy();
      mergeRef.current = null;
    };
    // נבנה פעם אחת בפתיחה; שינוי הגדרות בזמן פתיחת ההשוואה נדיר
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const apply = () => {
    const text = mergeRef.current?.b.state.doc.toString() ?? bufferText;
    onApply(text);
  };

  return (
    <div className="diff-overlay" dir="rtl">
      <div className="diff-window" role="dialog" aria-label="השוואה מול הדיסק">
        <div className="diff-header">
          <span className="diff-title">{title}</span>
          <span className="diff-sub">עורך (ניתן לעריכה) ↔ דיסק</span>
          <div className="diff-actions">
            <button type="button" className="diff-apply" onClick={apply}>
              החל שינויים
            </button>
            <button type="button" onClick={onClose}>
              סגור
            </button>
          </div>
        </div>
        <div className="diff-body" ref={hostRef} />
      </div>
    </div>
  );
}
