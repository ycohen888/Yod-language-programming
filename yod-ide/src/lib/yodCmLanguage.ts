import { StreamLanguage } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";
import {
  yodAfterClass,
  yodAfterFunction,
  yodAfterVar,
  yodBuiltinSet,
  yodConstantSet,
  yodControlSet,
  yodDeclKeywordSet,
} from "./yodKeywords";

type Expect = "function" | "class" | "variable" | null;

type State = {
  inComment: boolean;
  inString: boolean;
  inTemplate: boolean;
  expect: Expect;
};

/**
 * שפת יוד ל־CodeMirror — סיווג טוקנים כמו yod/internal/highlight (PHP Dark+).
 */
export const yodStreamLanguage = StreamLanguage.define({
  name: "yod",
  startState: (): State => ({
    inComment: false,
    inString: false,
    inTemplate: false,
    expect: null,
  }),
  token(stream, state) {
    if (state.inComment) {
      if (stream.match(/\*\//)) {
        state.inComment = false;
        return "comment";
      }
      stream.next();
      return "comment";
    }
    if (state.inString) {
      if (stream.match(/\\./)) return "string";
      if (stream.match('"') || stream.match("'")) {
        state.inString = false;
        return "string";
      }
      stream.next();
      return "string";
    }
    if (state.inTemplate) {
      if (stream.match(/\\./)) return "string";
      if (stream.match("`")) {
        state.inTemplate = false;
        return "string";
      }
      if (stream.match("${")) return "punctuation";
      stream.next();
      return "string";
    }

    if (stream.eatSpace()) return null;

    if (stream.match("//")) {
      stream.skipToEnd();
      return "comment";
    }
    if (stream.match("/*")) {
      state.inComment = true;
      return "comment";
    }
    if (stream.match('"') || stream.match("'")) {
      state.inString = true;
      return "string";
    }
    if (stream.match("`")) {
      state.inTemplate = true;
      return "string";
    }
    if (stream.match(/^\d+(\.\d+)?/)) {
      state.expect = null;
      return "number";
    }

    // מזהה / מילת מפתח (כולל _ בעברית: כל_עוד)
    if (stream.match(/^[א-תA-Za-z_־][א-תA-Za-z0-9_־]*/)) {
      const w = stream.current();

      if (yodControlSet.has(w)) {
        state.expect = null;
        return "controlKeyword";
      }
      if (yodConstantSet.has(w)) {
        state.expect = null;
        return "atom";
      }
      if (yodDeclKeywordSet.has(w)) {
        if (yodAfterFunction.has(w)) state.expect = "function";
        else if (yodAfterClass.has(w)) state.expect = "class";
        else if (yodAfterVar.has(w)) state.expect = "variable";
        else state.expect = null;
        return "keyword";
      }

      if (state.expect === "function") {
        state.expect = null;
        return "functionName";
      }
      if (state.expect === "class") {
        state.expect = null;
        return "typeName";
      }
      if (state.expect === "variable") {
        state.expect = null;
        return "variableName";
      }

      // קריאה: שם(
      if (stream.match(/^\s*\(/, false) || yodBuiltinSet.has(w)) {
        return "functionName";
      }
      return "variableName";
    }

    // אופרטורים דו־תוויים
    if (
      stream.match("==") ||
      stream.match("!=") ||
      stream.match("<=") ||
      stream.match(">=") ||
      stream.match("&&") ||
      stream.match("||") ||
      stream.match("+=") ||
      stream.match("-=") ||
      stream.match("*=") ||
      stream.match("/=") ||
      stream.match("%=") ||
      stream.match("**") ||
      stream.match("??") ||
      stream.match("->")
    ) {
      state.expect = null;
      return "operator";
    }

    if (stream.match(/^[+\-*/%=<>!&|]+/)) {
      state.expect = null;
      return "operator";
    }
    if (stream.match(/^[()[\]{};,.:]/)) {
      state.expect = null;
      return "punctuation";
    }

    stream.next();
    state.expect = null;
    return null;
  },
  tokenTable: {
    comment: t.comment,
    string: t.string,
    number: t.number,
    controlKeyword: t.controlKeyword,
    keyword: t.keyword,
    atom: t.atom,
    functionName: t.function(t.variableName),
    variableName: t.variableName,
    typeName: t.typeName,
    operator: t.operator,
    punctuation: t.punctuation,
  },
  languageData: {
    commentTokens: { line: "//", block: { open: "/*", close: "*/" } },
  },
});
