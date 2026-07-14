import type * as monaco from "monaco-editor";

const KEYWORDS = [
  "משתנה",
  "קבוע",
  "פונקציה",
  "אם",
  "אחרת",
  "אחרת_אם",
  "כל עוד",
  "עבור",
  "מ",
  "עד",
  "בצע",
  "החזר",
  "שבור",
  "המשך",
  "סוף",
  "כלול",
  "יבא",
  "מתוך",
  "יצא",
  "מודול",
  "מחלקה",
  "חדש",
  "זה",
  "הורה",
  "פרטי",
  "ציבורי",
  "אמת",
  "שקר",
  "ריק",
  "ו",
  "או",
  "לא",
  "משימה",
  "המתן",
  "במקביל",
  "נסה",
  "תפוס",
  "לבסוף",
  "זרוק",
];

/** רישום שפת יוד ב־Monaco — tokenizer מלא לקובץ (מהיר ב־worker). */
export function registerYodLanguage(m: typeof monaco) {
  if (m.languages.getLanguages().some((l) => l.id === "yod")) return;

  m.languages.register({ id: "yod", extensions: [".יוד", ".yod"], aliases: ["יוד", "Yod"] });

  m.languages.setLanguageConfiguration("yod", {
    comments: { lineComment: "//", blockComment: ["/*", "*/"] },
    brackets: [
      ["[", "]"],
      ["(", ")"],
      ["{", "}"],
    ],
    autoClosingPairs: [
      { open: "[", close: "]" },
      { open: "(", close: ")" },
      { open: "{", close: "}" },
      { open: '"', close: '"' },
      { open: "`", close: "`" },
    ],
    surroundingPairs: [
      { open: "[", close: "]" },
      { open: "(", close: ")" },
      { open: '"', close: '"' },
    ],
  });

  m.languages.setMonarchTokensProvider("yod", {
    defaultToken: "",
    tokenPostfix: ".yod",
    keywords: KEYWORDS,
    tokenizer: {
      root: [
        [/\/\*/, "comment", "@comment"],
        [/\/\/.*$/, "comment"],
        [/`/, "string", "@template"],
        [/"([^"\\]|\\.)*$/, "string.invalid"],
        [/"/, "string", "@string"],
        [/\d+(\.\d+)?/, "number"],
        [
          /[א-תA-Za-z_][א-תA-Za-z0-9_]*/,
          {
            cases: {
              "@keywords": "keyword",
              "@default": "identifier",
            },
          },
        ],
        [/[{}()\[\]]/, "@brackets"],
        [/[;,.]/, "delimiter"],
        [/[+\-*/%=<>!&|]+/, "operator"],
        [/\s+/, "white"],
      ],
      comment: [
        [/[^/*]+/, "comment"],
        [/\*\//, "comment", "@pop"],
        [/[/*]/, "comment"],
      ],
      string: [
        [/[^\\"]+/, "string"],
        [/\\./, "string.escape"],
        [/"/, "string", "@pop"],
      ],
      template: [
        [/[^\\`$]+/, "string"],
        [/\$\{/, { token: "delimiter.bracket", next: "@interpolated" }],
        [/`/, "string", "@pop"],
      ],
      interpolated: [
        [/[^}]+/, "identifier"],
        [/\}/, { token: "delimiter.bracket", next: "@pop" }],
      ],
    },
  });

  m.languages.registerCompletionItemProvider("yod", {
    triggerCharacters: [".", '"'],
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };
      const suggestions: monaco.languages.CompletionItem[] = KEYWORDS.map((k) => ({
        label: k,
        kind: m.languages.CompletionItemKind.Keyword,
        insertText: k,
        range,
      }));
      const builtins = ["הדפס", "קלט", "אורך", "טווח", "אקראי"];
      for (const b of builtins) {
        suggestions.push({
          label: b,
          kind: m.languages.CompletionItemKind.Function,
          insertText: b,
          range,
        });
      }
      return { suggestions };
    },
  });

  // ערכת צבעים בסגנון Dark+
  m.editor.defineTheme("yod-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "comment.yod", foreground: "6A9955" },
      { token: "keyword.yod", foreground: "C586C0" },
      { token: "string.yod", foreground: "CE9178" },
      { token: "number.yod", foreground: "B5CEA8" },
      { token: "identifier.yod", foreground: "D4D4D4" },
      { token: "operator.yod", foreground: "D4D4D4" },
    ],
    colors: {
      "editor.background": "#1e1e1e",
      "editor.foreground": "#d4d4d4",
      "editorLineNumber.foreground": "#858585",
      "editorCursor.foreground": "#aeafad",
      "editor.selectionBackground": "#264f78",
      "editor.inactiveSelectionBackground": "#3a3d41",
      "editorIndentGuide.background": "#404040",
      "editor.lineHighlightBackground": "#2a2d2e",
    },
  });
}

export function langForPath(filePath: string): string {
  const lower = filePath.toLowerCase();
  if (lower.endsWith(".יוד") || lower.endsWith(".yod")) return "yod";
  if (lower.endsWith(".json")) return "json";
  if (lower.endsWith(".html") || lower.endsWith(".htm")) return "html";
  if (lower.endsWith(".css")) return "css";
  if (lower.endsWith(".js") || lower.endsWith(".mjs") || lower.endsWith(".cjs")) return "javascript";
  if (lower.endsWith(".ts") || lower.endsWith(".tsx")) return "typescript";
  if (lower.endsWith(".md")) return "markdown";
  if (lower.endsWith(".go")) return "go";
  return "plaintext";
}
