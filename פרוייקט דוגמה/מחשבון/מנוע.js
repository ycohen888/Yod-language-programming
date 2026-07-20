/**
 * מנוע מחשבון — חישוב שרשרת, פונקציות, היסטוריה ומקלדת
 */
(function () {
  "use strict";

  var display = document.getElementById("value");
  var exprEl = document.getElementById("expr");
  var pad = document.getElementById("pad");
  var historyPanel = document.getElementById("history-panel");
  var historyList = document.getElementById("history-list");
  var app = document.getElementById("app");

  var state = {
    display: "0",
    expression: "",
    acc: null,
    op: null,
    fresh: true,
    history: []
  };

  function format(n) {
    if (!isFinite(n)) return "שגיאה";
    var s = Number(n.toPrecision(12)).toString();
    if (s.indexOf("e") >= 0) return s;
    if (s.indexOf(".") >= 0) s = s.replace(/\.?0+$/, "");
    return s || "0";
  }

  function render() {
    display.textContent = state.display;
    display.classList.toggle("error", state.display === "שגיאה");
    exprEl.innerHTML = state.expression ? state.expression : "&nbsp;";
    display.classList.remove("flash");
    void display.offsetWidth;
    display.classList.add("flash");
  }

  function setDisplay(v) {
    state.display = v;
    render();
  }

  function current() {
    var n = parseFloat(state.display.replace(",", "."));
    return isNaN(n) ? 0 : n;
  }

  function applyOp(a, op, b) {
    switch (op) {
      case "+": return a + b;
      case "−": return a - b;
      case "×": return a * b;
      case "÷": return b === 0 ? NaN : a / b;
      default: return b;
    }
  }

  function commitPending() {
    if (state.op == null || state.acc == null) return current();
    var r = applyOp(state.acc, state.op, current());
    state.acc = null;
    state.op = null;
    return r;
  }

  function pushHistory(expr, result) {
    state.history.unshift({ expr: expr, result: result });
    if (state.history.length > 30) state.history.pop();
    paintHistory();
  }

  function paintHistory() {
    historyList.innerHTML = "";
    if (!state.history.length) {
      var empty = document.createElement("div");
      empty.className = "history-empty";
      empty.textContent = "אין חישובים עדיין";
      historyList.appendChild(empty);
      return;
    }
    state.history.forEach(function (item) {
      var li = document.createElement("li");
      li.innerHTML =
        '<span class="h-expr">' +
        escapeHtml(item.expr) +
        '</span><span class="h-val">' +
        escapeHtml(item.result) +
        "</span>";
      li.addEventListener("click", function () {
        setDisplay(item.result);
        state.expression = item.expr;
        state.fresh = true;
        state.acc = null;
        state.op = null;
        historyPanel.hidden = true;
      });
      historyList.appendChild(li);
    });
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function inputDigit(d) {
    if (state.display === "שגיאה") {
      state.display = "0";
      state.expression = "";
    }
    if (state.fresh) {
      state.display = d === "." ? "0." : d;
      state.fresh = false;
    } else {
      if (d === "." && state.display.indexOf(".") >= 0) return;
      if (state.display === "0" && d !== ".") state.display = d;
      else state.display += d;
    }
    render();
  }

  function inputOp(op) {
    if (state.display === "שגיאה") return;
    var cur = current();
    if (state.op != null && !state.fresh) {
      var mid = applyOp(state.acc, state.op, cur);
      if (!isFinite(mid)) {
        fail();
        return;
      }
      state.acc = mid;
      setDisplay(format(mid));
    } else {
      state.acc = cur;
    }
    state.op = op;
    state.expression = format(state.acc) + " " + op;
    state.fresh = true;
    render();
  }

  function equals() {
    if (state.display === "שגיאה") return;
    if (state.op == null) {
      state.expression = format(current()) + " =";
      render();
      return;
    }
    var a = state.acc;
    var op = state.op;
    var b = current();
    var expr = format(a) + " " + op + " " + format(b);
    var r = applyOp(a, op, b);
    if (!isFinite(r)) {
      fail();
      return;
    }
    var out = format(r);
    state.expression = expr + " =";
    state.display = out;
    state.acc = null;
    state.op = null;
    state.fresh = true;
    pushHistory(expr, out);
    render();
  }

  function fail() {
    state.display = "שגיאה";
    state.expression = "";
    state.acc = null;
    state.op = null;
    state.fresh = true;
    render();
  }

  function clearAll() {
    state.display = "0";
    state.expression = "";
    state.acc = null;
    state.op = null;
    state.fresh = true;
    render();
  }

  function clearEntry() {
    state.display = "0";
    state.fresh = true;
    render();
  }

  function backspace() {
    if (state.fresh || state.display === "שגיאה") {
      clearEntry();
      return;
    }
    if (state.display.length <= 1 || (state.display.length === 2 && state.display[0] === "-")) {
      state.display = "0";
      state.fresh = true;
    } else {
      state.display = state.display.slice(0, -1);
    }
    render();
  }

  function negate() {
    if (state.display === "שגיאה" || state.display === "0") return;
    if (state.display[0] === "-") state.display = state.display.slice(1);
    else state.display = "-" + state.display;
    state.fresh = false;
    render();
  }

  function percent() {
    if (state.display === "שגיאה") return;
    var cur = current();
    var r;
    if (state.acc != null && (state.op === "+" || state.op === "−")) {
      r = state.acc * (cur / 100);
    } else {
      r = cur / 100;
    }
    setDisplay(format(r));
    state.fresh = true;
  }

  function unary(kind) {
    if (state.display === "שגיאה") return;
    var cur = current();
    var r;
    var label;
    switch (kind) {
      case "√":
        if (cur < 0) return fail();
        r = Math.sqrt(cur);
        label = "√(" + format(cur) + ")";
        break;
      case "x²":
        r = cur * cur;
        label = "sqr(" + format(cur) + ")";
        break;
      case "1/x":
        if (cur === 0) return fail();
        r = 1 / cur;
        label = "1/(" + format(cur) + ")";
        break;
      default:
        return;
    }
    if (!isFinite(r)) return fail();
    var out = format(r);
    state.expression = label;
    state.display = out;
    state.fresh = true;
    pushHistory(label, out);
    render();
  }

  function press(k) {
    var btn = pad.querySelector('[data-k="' + cssEscape(k) + '"]');
    if (btn) {
      btn.classList.add("pressed");
      setTimeout(function () { btn.classList.remove("pressed"); }, 120);
    }

    if (/^[0-9.]$/.test(k)) return inputDigit(k);
    if (k === "+" || k === "−" || k === "×" || k === "÷") return inputOp(k);
    if (k === "=") return equals();
    if (k === "C") return clearAll();
    if (k === "CE") return clearEntry();
    if (k === "⌫") return backspace();
    if (k === "±") return negate();
    if (k === "%") return percent();
    if (k === "√" || k === "x²" || k === "1/x") return unary(k);
  }

  function cssEscape(s) {
    if (window.CSS && CSS.escape) return CSS.escape(s);
    return s.replace(/"/g, '\\"');
  }

  pad.addEventListener("click", function (e) {
    var t = e.target.closest("[data-k]");
    if (!t) return;
    press(t.getAttribute("data-k"));
  });

  document.getElementById("btn-theme").addEventListener("click", function () {
    var cur = document.documentElement.getAttribute("data-theme");
    var next = cur === "light" ? "dark" : "light";
    if (next === "dark") document.documentElement.removeAttribute("data-theme");
    else document.documentElement.setAttribute("data-theme", "light");
    try { localStorage.setItem("yod-calc-theme", next); } catch (e) {}
  });

  document.getElementById("btn-history").addEventListener("click", function () {
    historyPanel.hidden = !historyPanel.hidden;
    if (!historyPanel.hidden) paintHistory();
  });

  document.getElementById("btn-clear-history").addEventListener("click", function () {
    state.history = [];
    paintHistory();
  });

  document.addEventListener("click", function (e) {
    if (historyPanel.hidden) return;
    if (historyPanel.contains(e.target)) return;
    if (e.target.id === "btn-history") return;
    historyPanel.hidden = true;
  });

  document.addEventListener("keydown", function (e) {
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    var k = e.key;
    var map = {
      "0": "0", "1": "1", "2": "2", "3": "3", "4": "4",
      "5": "5", "6": "6", "7": "7", "8": "8", "9": "9",
      ".": ".", ",": ".",
      "+": "+", "-": "−", "*": "×", "/": "÷",
      Enter: "=", "=": "=",
      Backspace: "⌫", Escape: "C", Delete: "CE",
      "%": "%"
    };
    if (map[k] != null) {
      e.preventDefault();
      press(map[k]);
    }
  });

  try {
    var saved = localStorage.getItem("yod-calc-theme");
    if (saved === "light") document.documentElement.setAttribute("data-theme", "light");
  } catch (e) {}

  render();
})();
