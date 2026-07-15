import type { Panel, ViewUpdate } from "@codemirror/view";
import { EditorView, runScopeHandlers } from "@codemirror/view";
import {
  SearchQuery,
  closeSearchPanel,
  findNext,
  findPrevious,
  getSearchQuery,
  replaceAll,
  replaceNext,
  selectMatches,
  setSearchQuery,
} from "@codemirror/search";

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  attrs: Record<string, string | boolean | ((e: Event) => void) | undefined> = {},
  children: (Node | string)[] = []
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v == null || v === false) continue;
    if (typeof v === "function") {
      node.addEventListener(k.replace(/^on/, "").toLowerCase(), v as EventListener);
    } else if (v === true) {
      node.setAttribute(k, "");
    } else {
      node.setAttribute(k, v);
    }
  }
  for (const c of children) {
    node.appendChild(typeof c === "string" ? document.createTextNode(c) : c);
  }
  return node;
}

function countMatches(view: EditorView, query: SearchQuery): { current: number; total: number } {
  if (!query.valid || !query.search) return { current: 0, total: 0 };
  const sel = view.state.selection.main;
  let total = 0;
  let current = 0;
  const cursor = query.getCursor(view.state);
  for (let item = cursor.next(); !item.done; item = cursor.next()) {
    total++;
    const m = item.value;
    if (m.from === sel.from && m.to === sel.to) current = total;
  }
  return { current, total };
}

/** פאנל חיפוש/החלפה בסגנון VS Code — RTL ומקצועי */
export function createYodSearchPanel(view: EditorView): Panel {
  let query = getSearchQuery(view.state);
  let replaceOpen = false;

  const searchInput = el("input", {
    type: "text",
    class: "cm-yod-search-input",
    placeholder: "חיפוש",
    "aria-label": "חיפוש",
    "main-field": "true",
    value: query.search,
    spellcheck: "false",
  }) as HTMLInputElement;
  searchInput.value = query.search;

  const replaceInput = el("input", {
    type: "text",
    class: "cm-yod-search-input",
    placeholder: "החלף ב…",
    "aria-label": "החלפה",
    "data-yod-replace": "true",
    value: query.replace,
    spellcheck: "false",
  }) as HTMLInputElement;
  replaceInput.value = query.replace;

  const matchInfo = el("span", { class: "cm-yod-search-count", "aria-live": "polite" }, ["אין תוצאות"]);

  function mkToggle(title: string, label: string, active: boolean, onClick: () => void) {
    const btn = el("button", {
      type: "button",
      class: "cm-yod-search-toggle" + (active ? " active" : ""),
      title,
      "aria-label": title,
      "aria-pressed": active ? "true" : "false",
    }, [label]) as HTMLButtonElement;
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      onClick();
    });
    return btn;
  }

  const caseBtn = mkToggle("התאם רישיות (Aa)", "Aa", query.caseSensitive, () => {
    caseBtn.classList.toggle("active");
    caseBtn.setAttribute("aria-pressed", caseBtn.classList.contains("active") ? "true" : "false");
    commit();
  });
  const wordBtn = mkToggle("מילה שלמה", "אב", query.wholeWord, () => {
    wordBtn.classList.toggle("active");
    wordBtn.setAttribute("aria-pressed", wordBtn.classList.contains("active") ? "true" : "false");
    commit();
  });
  const reBtn = mkToggle("ביטוי רגולרי", ".*", query.regexp, () => {
    reBtn.classList.toggle("active");
    reBtn.setAttribute("aria-pressed", reBtn.classList.contains("active") ? "true" : "false");
    commit();
  });

  function mkIconBtn(title: string, cls: string, content: string, onClick: () => void) {
    const btn = el("button", {
      type: "button",
      class: "cm-yod-search-btn " + cls,
      title,
      "aria-label": title,
    }, [content]) as HTMLButtonElement;
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      onClick();
    });
    return btn;
  }

  const expandBtn = mkIconBtn("הצג החלפה", "cm-yod-search-expand", "▾", () => {
    setReplaceOpen(!replaceOpen);
  });

  const prevBtn = mkIconBtn("הקודם (Shift+Enter)", "", "▴", () => findPrevious(view));
  const nextBtn = mkIconBtn("הבא (Enter)", "", "▾", () => findNext(view));
  const closeBtn = mkIconBtn("סגור (Esc)", "cm-yod-search-close", "✕", () => closeSearchPanel(view));

  const replaceOneBtn = el("button", {
    type: "button",
    class: "cm-yod-search-action",
    title: "החלף",
  }, ["החלף"]) as HTMLButtonElement;
  replaceOneBtn.addEventListener("click", (e) => {
    e.preventDefault();
    replaceNext(view);
    refreshCount();
  });

  const replaceAllBtn = el("button", {
    type: "button",
    class: "cm-yod-search-action",
    title: "החלף הכל",
  }, ["החלף הכל"]) as HTMLButtonElement;
  replaceAllBtn.addEventListener("click", (e) => {
    e.preventDefault();
    replaceAll(view);
    refreshCount();
  });

  const selectAllBtn = el("button", {
    type: "button",
    class: "cm-yod-search-action cm-yod-search-action-ghost",
    title: "בחר הכל",
  }, ["בחר הכל"]) as HTMLButtonElement;
  selectAllBtn.addEventListener("click", (e) => {
    e.preventDefault();
    selectMatches(view);
  });

  const findLine = el("div", { class: "cm-yod-search-find-line" }, [
    el("div", { class: "cm-yod-search-field-wrap" }, [searchInput]),
    matchInfo,
    el("div", { class: "cm-yod-search-toggles" }, [caseBtn, wordBtn, reBtn]),
    el("div", { class: "cm-yod-search-nav" }, [prevBtn, nextBtn]),
  ]);

  const replaceLine = el("div", { class: "cm-yod-search-replace-line" }, [
    el("div", { class: "cm-yod-search-field-wrap" }, [replaceInput]),
    el("div", { class: "cm-yod-search-replace-actions" }, [replaceOneBtn, replaceAllBtn, selectAllBtn]),
  ]);

  const fieldsCol = el("div", { class: "cm-yod-search-fields" }, [findLine, replaceLine]);

  const dom = el("div", {
    class: "cm-search cm-yod-search",
    role: "search",
    "aria-label": "חיפוש והחלפה",
  }, [expandBtn, fieldsCol, closeBtn]);

  function setReplaceOpen(open: boolean) {
    replaceOpen = open;
    dom.classList.toggle("cm-yod-search--replace", open);
    expandBtn.textContent = open ? "▴" : "▾";
    expandBtn.title = open ? "הסתר החלפה" : "הצג החלפה";
    expandBtn.setAttribute("aria-label", expandBtn.title);
  }

  function commit() {
    const next = new SearchQuery({
      search: searchInput.value,
      replace: replaceInput.value,
      caseSensitive: caseBtn.classList.contains("active"),
      wholeWord: wordBtn.classList.contains("active"),
      regexp: reBtn.classList.contains("active"),
    });
    if (!next.eq(query)) {
      query = next;
      view.dispatch({ effects: setSearchQuery.of(next) });
    }
    refreshCount();
  }

  function refreshCount() {
    const q = getSearchQuery(view.state);
    if (!q.search) {
      matchInfo.textContent = "";
      matchInfo.classList.remove("cm-yod-search-count--none");
      return;
    }
    if (!q.valid) {
      matchInfo.textContent = "ביטוי לא תקין";
      matchInfo.classList.add("cm-yod-search-count--none");
      return;
    }
    const { current, total } = countMatches(view, q);
    if (total === 0) {
      matchInfo.textContent = "אין תוצאות";
      matchInfo.classList.add("cm-yod-search-count--none");
    } else {
      matchInfo.textContent = current > 0 ? `${current} מתוך ${total}` : `${total} תוצאות`;
      matchInfo.classList.remove("cm-yod-search-count--none");
    }
  }

  function onInput() {
    commit();
  }

  searchInput.addEventListener("input", onInput);
  replaceInput.addEventListener("input", onInput);

  function keydown(e: KeyboardEvent) {
    if (runScopeHandlers(view, e, "search-panel")) {
      e.preventDefault();
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      if (e.target === replaceInput) {
        if (e.ctrlKey || e.metaKey) replaceAll(view);
        else replaceNext(view);
      } else {
        (e.shiftKey ? findPrevious : findNext)(view);
      }
      refreshCount();
    }
  }

  dom.addEventListener("keydown", keydown);

  // אם נפתח מ־Ctrl+H — הסימון מופיע מיד אחרי mount
  if (view.dom.classList.contains("cm-yod-prefer-replace")) {
    setReplaceOpen(true);
  }

  return {
    dom,
    top: true,
    mount() {
      const prefer = view.dom.classList.contains("cm-yod-prefer-replace");
      view.dom.classList.remove("cm-yod-prefer-replace");
      if (prefer) setReplaceOpen(true);
      refreshCount();
      if (prefer) {
        replaceInput.focus();
        replaceInput.select();
      } else {
        searchInput.focus();
        searchInput.select();
      }
    },
    update(update: ViewUpdate) {
      for (const tr of update.transactions) {
        for (const effect of tr.effects) {
          if (effect.is(setSearchQuery) && !effect.value.eq(query)) {
            query = effect.value;
            searchInput.value = query.search;
            replaceInput.value = query.replace;
            caseBtn.classList.toggle("active", query.caseSensitive);
            wordBtn.classList.toggle("active", query.wholeWord);
            reBtn.classList.toggle("active", query.regexp);
          }
        }
      }
      if (update.docChanged || update.selectionSet || update.transactions.some((t) => t.effects.some((e) => e.is(setSearchQuery)))) {
        refreshCount();
      }
    },
  };
}
