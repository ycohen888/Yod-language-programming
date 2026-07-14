import { RangeSet, RangeSetBuilder, StateEffect, StateField } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  GutterMarker,
  gutter,
  type DecorationSet,
} from "@codemirror/view";

export type EditorDiagnostic = {
  line: number;
  severity: "error" | "warning" | "info";
  message: string;
};

export const setEditorDiagnostics = StateEffect.define<EditorDiagnostic[]>();

class DiagMarker extends GutterMarker {
  constructor(
    readonly severity: EditorDiagnostic["severity"],
    readonly message: string
  ) {
    super();
  }
  eq(other: DiagMarker) {
    return this.severity === other.severity && this.message === other.message;
  }
  toDOM() {
    const el = document.createElement("div");
    el.className = `cm-diag-mark cm-diag-${this.severity}`;
    el.title = this.message;
    return el;
  }
}

type DiagState = {
  gutters: RangeSet<GutterMarker>;
  lines: DecorationSet;
};

const empty: DiagState = {
  gutters: RangeSet.empty,
  lines: Decoration.none,
};

function buildDiagState(
  doc: { lines: number; line: (n: number) => { from: number } },
  diags: EditorDiagnostic[]
): DiagState {
  const byLine = new Map<number, EditorDiagnostic>();
  for (const d of diags) {
    if (!d.line || d.line < 1) continue;
    const prev = byLine.get(d.line);
    if (!prev || severityRank(d.severity) > severityRank(prev.severity)) {
      byLine.set(d.line, d);
    }
  }

  const gBuilder = new RangeSetBuilder<GutterMarker>();
  const lBuilder = new RangeSetBuilder<Decoration>();
  const sorted = [...byLine.entries()].sort((a, b) => a[0] - b[0]);
  for (const [line, d] of sorted) {
    if (line > doc.lines) continue;
    const from = doc.line(line).from;
    gBuilder.add(from, from, new DiagMarker(d.severity, d.message));
    lBuilder.add(
      from,
      from,
      Decoration.line({ class: `cm-diag-line cm-diag-line-${d.severity}` })
    );
  }
  return { gutters: gBuilder.finish(), lines: lBuilder.finish() };
}

function severityRank(s: EditorDiagnostic["severity"]) {
  if (s === "error") return 3;
  if (s === "warning") return 2;
  return 1;
}

export const diagnosticsField = StateField.define<DiagState>({
  create: () => empty,
  update(value, tr) {
    for (const e of tr.effects) {
      if (e.is(setEditorDiagnostics)) {
        return buildDiagState(tr.state.doc, e.value);
      }
    }
    if (tr.docChanged && value !== empty) {
      return {
        gutters: value.gutters.map(tr.changes),
        lines: value.lines.map(tr.changes),
      };
    }
    return value;
  },
  provide: (f) => [
    EditorView.decorations.from(f, (v) => v.lines),
    gutter({
      class: "cm-diag-gutter",
      markers: (view) => view.state.field(f).gutters,
    }),
  ],
});

export const yodDiagnosticsExt = [diagnosticsField];
