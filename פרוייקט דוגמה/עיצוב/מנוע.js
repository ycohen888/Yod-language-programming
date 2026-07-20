/**
 * מנוע עיצוב — גשר יוד↔DOM עם תור פקודות עד DOM מוכן
 */
(function (global) {
  "use strict";

  var pendingCommands = [];
  var domReady = false;
  var controls = Object.create(null);
  var radioSeq = 0;
  var checkSeq = 0;
  var pendingShowTargets = Object.create(null);

  /* ——— תרגום (i18n) ——— */
  var i18n = {
    lang: "he",
    gender: "",
    dir: "rtl",
    dict: Object.create(null)
  };
  global.__יוד_i18n = i18n;

  function i18nT(s, ctx) {
    if (s == null) return "";
    s = String(s);
    if (!s) return s;
    var key = ctx ? ctx + "\x04" + s : s;
    if (i18n.dict[key] != null && i18n.dict[key] !== "") return String(i18n.dict[key]);
    if (ctx && i18n.dict[s] != null && i18n.dict[s] !== "") return String(i18n.dict[s]);
    return s;
  }
  global.t = i18nT;

  function i18nTN(singular, plural, n) {
    var form = n === 1 ? singular : plural;
    var out = i18nT(form);
    return String(out).replace(/%d/g, String(n));
  }
  global.tn = i18nTN;

  function applyDocumentDir(dir) {
    var rtl = dir !== "ltr";
    document.documentElement.setAttribute("dir", rtl ? "rtl" : "ltr");
    if (document.body) {
      document.body.setAttribute("dir", rtl ? "rtl" : "ltr");
    }
    var rtlLink = document.getElementById("bs-rtl");
    var ltrLink = document.getElementById("bs-ltr");
    if (rtlLink && ltrLink) {
      rtlLink.disabled = !rtl;
      ltrLink.disabled = rtl;
      if ("media" in rtlLink) {
        rtlLink.media = rtl ? "all" : "none";
        ltrLink.media = rtl ? "none" : "all";
      }
    }
  }

  function setI18nCatalog(v) {
    i18n.lang = v.שפה != null ? String(v.שפה) : i18n.lang;
    i18n.gender = v.מגדר != null ? String(v.מגדר) : i18n.gender;
    i18n.dir = v.כיוון != null ? String(v.כיוון) : i18n.dir;
    i18n.dict = Object.create(null);
    var d = v.מילון || v.dict || {};
    if (d && typeof d === "object") {
      Object.keys(d).forEach(function (k) {
        i18n.dict[k] = d[k] != null ? String(d[k]) : "";
      });
    }
    document.documentElement.setAttribute("lang", i18n.lang || "he");
    applyDocumentDir(i18n.dir);
    refreshAllI18n();
  }

  function refreshAllI18n() {
    Object.keys(controls).forEach(function (id) {
      var c = controls[id];
      if (!c) return;
      try {
        if (typeof c.applyI18n === "function") {
          c.applyI18n();
          return;
        }
        if (c.sourceText != null && typeof c.setText === "function") {
          c.setText(c.sourceText);
        } else if (c.sourceText != null && c.el) {
          var display = i18nT(c.sourceText);
          c.text = display;
          if (c.labelEl) c.labelEl.textContent = display;
          else if (c.brandTitleEl) c.brandTitleEl.textContent = display;
        }
      } catch (e) {}
    });
    var splash = document.getElementById("yod-splash-sub");
    if (splash) splash.textContent = i18nT("טוען…");
    var splashRoot = document.getElementById("yod-splash");
    if (splashRoot) splashRoot.setAttribute("aria-label", i18nT("טוען"));
  }

  function post(obj) {
    var s;
    try {
      s = JSON.stringify(obj);
    } catch (e) {
      return;
    }
    // ערוץ אחד בלבד — כמה ערוצים גרמו ללחיצה כפולה/רביעית
    try {
      if (global.__יוד_גשר) {
        if (global.fetch) {
          global
            .fetch(global.__יוד_גשר, {
              method: "POST",
              body: s,
              mode: "cors",
              keepalive: true,
              cache: "no-store"
            })
            .catch(function () {});
        } else {
          var img = new Image();
          img.src = global.__יוד_גשר + "?" + encodeURIComponent(s) + "&_=" + Date.now();
        }
        return;
      }
    } catch (e0) {}
    try {
      if (global.chrome && global.chrome.webview && global.chrome.webview.postMessage) {
        global.chrome.webview.postMessage(s);
      }
    } catch (e2) {}
  }

  function elById(id) {
    return document.getElementById(id);
  }

  function root() {
    return document.getElementById("תוכן") || document.body;
  }

  function markHasControls() {
    var r = root();
    if (r) r.classList.add("has-controls");
    var hint = document.getElementById("רמז_ריק");
    if (hint) hint.style.display = "none";
  }

  /* ——— בסיס Control ——— */
  function Control(opts) {
    this.id = opts.מזהה || opts.id;
    if (!this.id) {
      this.id = "yod_auto_" + ++checkSeq;
    }
    this.type = opts.סוג || opts.type || "פקד";
    var raw = opts.טקסט != null ? String(opts.טקסט) : "";
    this.sourceText = raw;
    this.text = i18nT(raw);
    this.parentId = opts.הורה || opts.parent || "";
    this.el = null;
  }

  function isLogFooterEl(c) {
    return (
      c &&
      (c.classList.contains("yod-box-log") ||
        c.getAttribute("data-yod-box") === "יומן")
    );
  }

  function isAdBarEl(c) {
    return (
      c &&
      (c.classList.contains("yod-adbar") ||
        c.getAttribute("data-yod-box") === "פרסומת")
    );
  }

  function placeChromeLayout() {
    var r = root();
    if (!r) return;
    var top = r.querySelector(':scope > [data-yod-type="סרגל_עליון"]');
    var scroll = document.getElementById("yod-main-scroll");
    var log = r.querySelector(':scope > .yod-box-log, :scope > [data-yod-box="יומן"]');
    var ad = r.querySelector(':scope > .yod-adbar, :scope > [data-yod-box="פרסומת"]');
    if (top) {
      r.insertBefore(top, r.firstChild);
    }
    if (scroll) {
      if (top) {
        r.insertBefore(scroll, top.nextSibling);
      } else {
        r.insertBefore(scroll, r.firstChild);
      }
    }
    if (log) {
      r.appendChild(log);
    }
    // הכרזה נעוצה בתחתית ממש — אחרי היומן
    if (ad) {
      r.appendChild(ad);
    }
  }

  function ensureMainScroll() {
    var r = root();
    if (!r) return null;
    var scroll = document.getElementById("yod-main-scroll");
    if (!scroll) {
      scroll = document.createElement("div");
      scroll.id = "yod-main-scroll";
      scroll.className = "yod-main-scroll";
      var move = [];
      for (var i = 0; i < r.children.length; i++) {
        var c = r.children[i];
        if (!c || c.id === "רמז_ריק" || c.id === "yod-main-scroll") continue;
        if (c.getAttribute("data-yod-type") === "סרגל_עליון") continue;
        if (isLogFooterEl(c)) continue;
        if (isAdBarEl(c)) continue;
        move.push(c);
      }
      for (var j = 0; j < move.length; j++) {
        scroll.appendChild(move[j]);
      }
      r.appendChild(scroll);
    }
    placeChromeLayout();
    return scroll;
  }

  Control.prototype.resolveMountParent = function (parent) {
    if (parent) return parent;
    if (this.parentId) {
      var host = controls[this.parentId];
      if (host && host.el) {
        if (typeof host.mountTarget === "function") {
          return host.mountTarget() || host.el;
        }
        return host.el;
      }
      var byId = elById(this.parentId);
      if (byId) return byId;
    }
    var r = root();
    if (this.type === "סרגל_עליון") {
      return r;
    }
    if (this.type === "מכל" && this.kind === "יומן") {
      return r;
    }
    if (
      r &&
      (r.querySelector('[data-yod-type="סרגל_עליון"]') ||
        r.querySelector(".yod-box-log"))
    ) {
      return ensureMainScroll() || r;
    }
    var existing = document.getElementById("yod-main-scroll");
    if (existing) return existing;
    return r;
  };

  Control.prototype.mount = function (parent) {
    if (!this.el) {
      this.el = this.createElement();
      this.el.id = this.id;
      this.el.classList.add("yod-control");
      this.el.setAttribute("data-yod-id", this.id);
      this.el.setAttribute("data-yod-type", this.type);
      this.bindEvents();
    }
    this.resolveMountParent(parent).appendChild(this.el);
    markHasControls();
    controls[this.id] = this;
    return this;
  };

  Control.prototype.createElement = function () {
    var d = document.createElement("div");
    d.textContent = this.text;
    return d;
  };

  Control.prototype.bindEvents = function () {};

  Control.prototype.setText = function (raw) {
    this.sourceText = raw == null ? "" : String(raw);
    this.text = i18nT(this.sourceText);
    if (this.el) this.applyText();
  };

  Control.prototype.applyI18n = function () {
    if (this.sourceText != null) {
      this.text = i18nT(this.sourceText);
      if (this.el && typeof this.applyText === "function") this.applyText();
    }
    if (this.sourcePlaceholder != null) {
      this.placeholder = i18nT(this.sourcePlaceholder);
      if (this.el && this.el.placeholder !== undefined) this.el.placeholder = this.placeholder;
      if (this.input) this.input.placeholder = this.placeholder;
    }
    if (this.sourceTitle != null) {
      this.title = i18nT(this.sourceTitle);
      if (this.brandTitleEl) this.brandTitleEl.textContent = this.title || i18nT("תפריט");
      else if (this.titleEl) this.titleEl.textContent = this.title;
    }
    if (this.sourceBody != null) {
      this.body = i18nT(this.sourceBody);
      if (typeof this._syncBodyText === "function") this._syncBodyText();
    }
    if (this.sourceActionLabel != null) {
      this.actionLabel = i18nT(this.sourceActionLabel);
      if (typeof this._syncActionBtn === "function") this._syncActionBtn();
    }
    if (this.sourceSubtitle != null) {
      this.subtitle = i18nT(this.sourceSubtitle);
      if (this.subtitleEl) this.subtitleEl.textContent = this.subtitle;
    }
    if (typeof this.render === "function" && (this.columns || this.items || this.nodes || this.rawItems || this.slides)) {
      this.render();
    }
    if (this.el) {
      var closeBtns = this.el.querySelectorAll("[data-yod-modal-close='1'], .yod-popover-close");
      for (var ci = 0; ci < closeBtns.length; ci++) {
        var cb = closeBtns[ci];
        if (cb.getAttribute("aria-label") != null) {
          cb.setAttribute("aria-label", i18nT("סגור"));
        }
        if (cb.classList && !cb.classList.contains("btn-close") && cb.tagName === "BUTTON") {
          cb.textContent = i18nT("סגור");
        }
      }
    }
  };

  Control.prototype.applyText = function () {
    if (!this.el) return;
    if (this.labelEl) {
      this.labelEl.textContent = this.text;
      return;
    }
    if (this.brandTitleEl) {
      this.brandTitleEl.textContent = this.text;
      return;
    }
    // אל תמחקו מבנה DOM עם ילדים (מכל / כרטיס / סרגל / …)
    if (this.el.childElementCount > 0) {
      return;
    }
    this.el.textContent = this.text;
  };

  Control.prototype.setVisible = function (v) {
    if (!this.el) return;
    this.el.classList.toggle("d-none", !v);
  };

  Control.prototype.setDisabled = function (v) {
    if (!this.el) return;
    if ("disabled" in this.el) {
      this.el.disabled = !!v;
    } else {
      this.el.setAttribute("aria-disabled", v ? "true" : "false");
      this.el.classList.toggle("disabled", !!v);
    }
  };

  Control.prototype.remove = function () {
    if (this.el && this.el.parentNode) {
      this.el.parentNode.removeChild(this.el);
    }
    delete controls[this.id];
    this.el = null;
  };

  /* ——— כפתור ——— */
  var STYLE_MAP = {
    ראשי: "btn-primary",
    משני: "btn-secondary",
    סכנה: "btn-danger",
    הצלחה: "btn-success",
    אזהרה: "btn-warning",
    מידע: "btn-info",
    קווי: "btn-outline-primary"
  };

  function Button(opts) {
    Control.call(this, opts);
    this.type = "כפתור";
    this.styleName = opts.סגנון || opts.style || "ראשי";
    this.showTarget =
      opts.יעד_הצגה ||
      opts.showTarget ||
      pendingShowTargets[this.id] ||
      "";
    if (this.showTarget && pendingShowTargets[this.id]) {
      delete pendingShowTargets[this.id];
    }
  }

  Button.prototype = Object.create(Control.prototype);
  Button.prototype.constructor = Button;

  Button.prototype.createElement = function () {
    var b = document.createElement("button");
    b.type = "button";
    b.className = "btn " + (STYLE_MAP[this.styleName] || STYLE_MAP["ראשי"]);
    decorateButtonLabel(b, this.text);
    b.style.position = "relative";
    b.style.zIndex = "2";
    return b;
  };

  Button.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    if (!this.showTarget && pendingShowTargets[this.id]) {
      this.showTarget = pendingShowTargets[this.id];
      delete pendingShowTargets[this.id];
    }
    return this;
  };

  Button.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      var tid = self.showTarget || pendingShowTargets[self.id] || "";
      if (tid) {
        var tgt = controls[tid];
        if (tgt && typeof tgt.show === "function") {
          try {
            tgt.show();
          } catch (e) {}
        }
      }
      post({
        סוג: "אירוע",
        שם: "לחיצה",
        ערכים: { מזהה: self.id, סוג: self.type || "כפתור" }
      });
    });
  };

  Button.prototype.setShowTarget = function (id) {
    this.showTarget = id == null ? "" : String(id);
    if (this.id) pendingShowTargets[this.id] = this.showTarget;
  };

  Button.prototype.applyText = function () {
    if (this.el) decorateButtonLabel(this.el, this.text);
  };

  Button.prototype.setStyle = function (name) {
    this.styleName = name || "ראשי";
    if (!this.el) return;
    this.el.className = "btn yod-control " + (STYLE_MAP[this.styleName] || STYLE_MAP["ראשי"]);
  };

  /* ——— שדה (קלט) ——— */
  var INPUT_TYPE_MAP = {
    טקסט: "text",
    אימייל: "email",
    סיסמה: "password",
    מספר: "number",
    חיפוש: "search",
    טלפון: "tel",
    כתובת: "url"
  };

  function Field(opts) {
    Control.call(this, opts);
    this.type = "שדה";
    this.sourcePlaceholder = opts.רמז != null ? String(opts.רמז) : "";
    this.placeholder = i18nT(this.sourcePlaceholder);
    this.fieldKind = opts.סוג_שדה || opts.סוג_קלט || "טקסט";
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    // ערך שדה — נתון משתמש, לא msgid
    this.sourceText = this.value;
    this.text = this.value;
  }

  Field.prototype = Object.create(Control.prototype);
  Field.prototype.constructor = Field;

  Field.prototype.createElement = function () {
    var inp = document.createElement("input");
    inp.type = INPUT_TYPE_MAP[this.fieldKind] || "text";
    inp.className = "form-control";
    inp.placeholder = this.placeholder;
    inp.value = this.value;
    return inp;
  };

  Field.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("input", function () {
      self.value = self.el.value;
      post({
        סוג: "אירוע",
        שם: "שינוי",
        ערכים: { מזהה: self.id, סוג: "שדה", ערך: self.value }
      });
    });
    this.el.addEventListener("keydown", function (ev) {
      if (ev.key === "Enter") {
        self.value = self.el.value;
        post({
          סוג: "אירוע",
          שם: "כניסה",
          ערכים: { מזהה: self.id, סוג: "שדה", ערך: self.value }
        });
      }
    });
  };

  Field.prototype.applyText = function () {
    if (this.el) this.el.value = this.text;
    this.value = this.text;
  };

  Field.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.text = this.value;
    if (this.el) this.el.value = this.value;
  };

  Field.prototype.setText = function (raw) {
    this.sourceText = raw == null ? "" : String(raw);
    this.text = this.sourceText;
    this.value = this.text;
    if (this.el) this.applyText();
  };

  Field.prototype.applyI18n = function () {
    if (this.sourcePlaceholder != null) {
      this.placeholder = i18nT(this.sourcePlaceholder);
      if (this.el) this.el.placeholder = this.placeholder;
    }
  };

  Field.prototype.setPlaceholder = function (p) {
    this.sourcePlaceholder = p == null ? "" : String(p);
    this.placeholder = i18nT(this.sourcePlaceholder);
    if (this.el) this.el.placeholder = this.placeholder;
  };

  /* ——— אזור טקסט ——— */
  function TextArea(opts) {
    Control.call(this, opts);
    this.type = "אזור_טקסט";
    this.sourcePlaceholder = opts.רמז != null ? String(opts.רמז) : "";
    this.placeholder = i18nT(this.sourcePlaceholder);
    this.rows = opts.שורות != null ? Number(opts.שורות) || 4 : 4;
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.sourceText = this.value;
    this.text = this.value;
  }

  TextArea.prototype = Object.create(Control.prototype);
  TextArea.prototype.constructor = TextArea;

  TextArea.prototype.createElement = function () {
    var ta = document.createElement("textarea");
    ta.className = "form-control";
    ta.rows = this.rows;
    ta.placeholder = this.placeholder;
    ta.value = this.value;
    return ta;
  };

  TextArea.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("input", function () {
      self.value = self.el.value;
      post({
        סוג: "אירוע",
        שם: "שינוי",
        ערכים: { מזהה: self.id, סוג: "אזור_טקסט", ערך: self.value }
      });
    });
  };

  TextArea.prototype.applyText = function () {
    if (this.el) this.el.value = this.text;
    this.value = this.text;
  };

  TextArea.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.text = this.value;
    if (this.el) this.el.value = this.value;
  };

  TextArea.prototype.setText = function (raw) {
    this.sourceText = raw == null ? "" : String(raw);
    this.text = this.sourceText;
    this.value = this.text;
    if (this.el) this.applyText();
  };

  TextArea.prototype.applyI18n = function () {
    if (this.sourcePlaceholder != null) {
      this.placeholder = i18nT(this.sourcePlaceholder);
      if (this.el) this.el.placeholder = this.placeholder;
    }
  };

  TextArea.prototype.setPlaceholder = function (p) {
    this.sourcePlaceholder = p == null ? "" : String(p);
    this.placeholder = i18nT(this.sourcePlaceholder);
    if (this.el) this.el.placeholder = this.placeholder;
  };

  TextArea.prototype.setRows = function (n) {
    this.rows = n > 0 ? n : 4;
    if (this.el) this.el.rows = this.rows;
  };

  /* ——— תווית ——— */
  var LABEL_STYLE = {
    רגיל: "form-label yod-label",
    כותרת: "h6 yod-label mb-1 fw-semibold",
    משני: "form-text yod-label yod-label-sub d-block small mb-1",
    תת_כותרת: "form-text yod-label yod-label-sub d-block small mb-1",
    הצלחה: "form-label yod-label text-success",
    סכנה: "form-label yod-label text-danger",
    מידע: "form-label yod-label text-info"
  };

  function normalizeLabelStyle(name) {
    var n = name || "רגיל";
    if (n === "תת כותרת" || n === "תת-כותרת") n = "תת_כותרת";
    if (LABEL_STYLE[n]) return n;
    return "רגיל";
  }

  function Label(opts) {
    Control.call(this, opts);
    this.type = "תווית";
    this.styleName = normalizeLabelStyle(opts.סגנון || "רגיל");
  }

  Label.prototype = Object.create(Control.prototype);
  Label.prototype.constructor = Label;

  Label.prototype.createElement = function () {
    var tag = this.styleName === "כותרת" ? "h5" : "label";
    var el = document.createElement(tag);
    el.className = LABEL_STYLE[this.styleName] || LABEL_STYLE["רגיל"];
    this._paintLabel(el);
    return el;
  };

  Label.prototype._paintLabel = function (el) {
    el = el || this.el;
    if (!el) return;
    var t = this.text == null ? "" : String(this.text);
    var icoKey = "";
    if (this.styleName === "הצלחה") icoKey = "check";
    else if (this.styleName === "סכנה") icoKey = "x-circle";
    else if (LABEL_ICO[t]) icoKey = LABEL_ICO[t];
    if (icoKey && ICO[icoKey]) {
      el.classList.add("yod-label-with-ico");
      if (this.styleName === "הצלחה") {
        el.classList.add("yod-label-status", "yod-label-status-ok");
        el.classList.remove("yod-label-status-fail");
      } else if (this.styleName === "סכנה") {
        el.classList.add("yod-label-status", "yod-label-status-fail");
        el.classList.remove("yod-label-status-ok");
      } else {
        el.classList.remove("yod-label-status", "yod-label-status-ok", "yod-label-status-fail");
      }
      el.innerHTML =
        svgIcon(icoKey, "yod-ico yod-label-ico") +
        '<span class="yod-label-ico-text">' +
        escapeHtml(t) +
        "</span>";
    } else {
      el.classList.remove(
        "yod-label-with-ico",
        "yod-label-status",
        "yod-label-status-ok",
        "yod-label-status-fail"
      );
      el.textContent = t;
    }
  };

  Label.prototype.applyText = function () {
    this._paintLabel(this.el);
  };

  Label.prototype.setStyle = function (name) {
    this.styleName = normalizeLabelStyle(name || "רגיל");
    if (!this.el) return;
    var wantH = this.styleName === "כותרת";
    var isH = this.el.tagName.toLowerCase() === "h5";
    if (wantH !== isH) {
      var neu = document.createElement(wantH ? "h5" : "label");
      neu.id = this.id;
      neu.className = "yod-control " + (LABEL_STYLE[this.styleName] || LABEL_STYLE["רגיל"]);
      neu.setAttribute("data-yod-id", this.id);
      neu.setAttribute("data-yod-type", this.type);
      if (this.el.parentNode) {
        this.el.parentNode.replaceChild(neu, this.el);
      }
      this.el = neu;
    } else {
      this.el.className = "yod-control " + (LABEL_STYLE[this.styleName] || LABEL_STYLE["רגיל"]);
    }
    this._paintLabel(this.el);
  };

  function normItems(items) {
    if (!items || !items.length) return [];
    var out = [];
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (typeof it === "string" || typeof it === "number") {
        out.push({ טקסט: String(it), ערך: String(it) });
      } else if (it && typeof it === "object") {
        var t =
          it.טקסט != null
            ? String(it.טקסט)
            : it.label != null
              ? String(it.label)
              : String(it.ערך != null ? it.ערך : it.value != null ? it.value : "");
        var val =
          it.ערך != null ? String(it.ערך) : it.value != null ? String(it.value) : t;
        var row = { טקסט: t, ערך: val };
        if (it.התקדמות != null && it.התקדמות !== "") {
          var p = Number(it.התקדמות);
          if (!isNaN(p)) {
            if (p < 0) p = 0;
            if (p > 100) p = 100;
            row.התקדמות = p;
          }
        }
        if (it.סוג != null) row.סוג = String(it.סוג);
        out.push(row);
      }
    }
    return out;
  }

  // נרמול פריטי תפריט (סרגל עליון/צד) — קינון, מפריד, ריק, איקון, רוחב מלא
  function normMenuItems(items) {
    if (!items || !items.length) return [];
    var out = [];
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (it == null) continue;
      if (typeof it === "string" || typeof it === "number") {
        var s = String(it);
        var st = s.trim();
        if (st === "-" || st === "—" || st === "מפריד") {
          out.push({ סוג: "מפריד", טקסט: "", ערך: "", ילדים: [] });
          continue;
        }
        if (st === "" || st === "ריק") {
          out.push({ סוג: "ריק", טקסט: "", ערך: "", ילדים: [] });
          continue;
        }
        out.push({ סוג: "פריט", טקסט: s, ערך: s, ילדים: [], איקון: "", רוחב_מלא: false, מושבת: false });
        continue;
      }
      if (typeof it !== "object") continue;
      var kind = it.סוג != null ? String(it.סוג) : it.kind != null ? String(it.kind) : "פריט";
      if (kind === "separator" || kind === "מפריד") {
        out.push({ סוג: "מפריד", טקסט: "", ערך: "", ילדים: [] });
        continue;
      }
      if (kind === "spacer" || kind === "ריק" || kind === "empty") {
        out.push({ סוג: "ריק", טקסט: "", ערך: "", ילדים: [] });
        continue;
      }
      if (kind === "header" || kind === "כותרת") {
        var ht =
          it.טקסט != null
            ? String(it.טקסט)
            : it.label != null
              ? String(it.label)
              : "";
        out.push({ סוג: "כותרת", טקסט: ht, ערך: "", ילדים: [], איקון: "" });
        continue;
      }
      var t2 =
        it.טקסט != null
          ? String(it.טקסט)
          : it.label != null
            ? String(it.label)
            : String(it.ערך != null ? it.ערך : it.value != null ? it.value : "");
      var val2 =
        it.ערך != null ? String(it.ערך) : it.value != null ? String(it.value) : t2;
      var kidsRaw = it.ילדים || it.פריטים || it.children || [];
      var kids = Array.isArray(kidsRaw) ? normMenuItems(kidsRaw) : [];
      var icon =
        it.איקון != null
          ? String(it.איקון)
          : it.icon != null
            ? String(it.icon)
            : "";
      out.push({
        סוג: "פריט",
        טקסט: t2,
        ערך: val2,
        איקון: icon,
        ילדים: kids,
        רוחב_מלא: !!(it.רוחב_מלא || it.fullWidth || it.mega),
        מושבת: !!(it.מושבת || it.disabled)
      });
    }
    return out;
  }

  function docIsRtl() {
    return (document.documentElement.getAttribute("dir") || "rtl") !== "ltr";
  }

  // תת־תפריט: RTL→שמאל, LTR→ימין; אם חורג מהמסך — הופכים צד
  function positionSubmenuFlip(wrap, sub) {
    if (!wrap || !sub) return;
    wrap.classList.remove("yod-sub-flip");
    var rect = sub.getBoundingClientRect();
    var pad = 8;
    var overflow = docIsRtl() ? rect.left < pad : rect.right > window.innerWidth - pad;
    if (overflow) {
      wrap.classList.add("yod-sub-flip");
    }
  }

  function closeAllYodMenus() {
    var nodes = document.querySelectorAll(".yod-menu-root.open");
    for (var i = 0; i < nodes.length; i++) {
      nodes[i].classList.remove("open");
      var tog = nodes[i].querySelector(":scope > .yod-menu-toggle");
      if (tog) tog.setAttribute("aria-expanded", "false");
    }
    var megas = document.querySelectorAll(".yod-menu-mega.show");
    for (var j = 0; j < megas.length; j++) {
      megas[j].classList.remove("show");
    }
    var subs = document.querySelectorAll(".yod-menu-item.open-sub, .yod-menu-item.yod-sub-flip");
    for (var k = 0; k < subs.length; k++) {
      subs[k].classList.remove("open-sub");
      subs[k].classList.remove("yod-sub-flip");
    }
  }

  function menuIconHtml(iconKey, text) {
    var key = iconKey || "";
    if (!key && text) key = LABEL_ICO[String(text)] || "";
    if (!key) return "";
    if (ICO[key]) return svgIcon(key, "yod-menu-icon");
    // מפתח עברי → LABEL_ICO
    if (LABEL_ICO[key] && ICO[LABEL_ICO[key]]) {
      return svgIcon(LABEL_ICO[key], "yod-menu-icon");
    }
    var low = key.toLowerCase();
    if (
      low.indexOf("data:") === 0 ||
      low.indexOf("http://") === 0 ||
      low.indexOf("https://") === 0 ||
      low.indexOf("./") === 0 ||
      low.indexOf("/") === 0 ||
      /\.(png|jpg|jpeg|gif|svg|webp)(\?|$)/i.test(key)
    ) {
      return (
        '<img class="yod-menu-icon yod-menu-icon-img" src="' +
        key.replace(/"/g, "&quot;") +
        '" alt="" />'
      );
    }
    return "";
  }

  function decorateMenuLabel(el, text, iconKey) {
    if (!el) return;
    var t = text == null ? "" : String(text);
    var ico = menuIconHtml(iconKey, t);
    if (!ico && LABEL_ICO[t]) {
      ico = svgIcon(LABEL_ICO[t], "yod-menu-icon");
    }
    // תמיד משבצת איקון קבועה — כדי שטקסט עם/בלי איקון יתחיל באותה עמודה
    if (!ico) {
      ico = '<span class="yod-menu-icon yod-menu-icon-slot" aria-hidden="true"></span>';
    }
    el.innerHTML = ico + '<span class="yod-menu-label">' + escapeHtml(t) + "</span>";
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function postMenuClick(ctrl, value) {
    ctrl.value = value == null ? "" : String(value);
    post({
      סוג: "אירוע",
      שם: "לחיצה",
      ערכים: { מזהה: ctrl.id, סוג: ctrl.type, ערך: ctrl.value }
    });
    postChange(ctrl.id, ctrl.type, ctrl.value);
  }

  function buildMenuPanel(items, ctrl, depth) {
    depth = depth || 0;
    var panel = document.createElement("div");
    panel.className = depth === 0 ? "yod-menu" : "yod-menu yod-menu-sub";
    panel.setAttribute("role", "menu");
    panel.addEventListener("click", function (ev) {
      ev.stopPropagation();
    });
    for (var i = 0; i < items.length; i++) {
      (function (it) {
        if (it.סוג === "מפריד") {
          var sep = document.createElement("div");
          sep.className = "yod-menu-sep";
          sep.setAttribute("role", "separator");
          panel.appendChild(sep);
          return;
        }
        if (it.סוג === "ריק") {
          var sp = document.createElement("div");
          sp.className = "yod-menu-spacer";
          sp.setAttribute("aria-hidden", "true");
          panel.appendChild(sp);
          return;
        }
        if (it.סוג === "כותרת") {
          var hd = document.createElement("div");
          hd.className = "yod-menu-header";
          hd.textContent = i18nT(it.טקסט || "");
          panel.appendChild(hd);
          return;
        }
        var hasKids = it.ילדים && it.ילדים.length > 0;
        if (hasKids) {
          var wrap = document.createElement("div");
          wrap.className = "yod-menu-item has-sub";
          var btn = document.createElement("button");
          btn.type = "button";
          btn.className = "yod-menu-btn";
          btn.setAttribute("role", "menuitem");
          if (it.מושבת) btn.disabled = true;
          decorateMenuLabel(btn, i18nT(it.טקסט || ""), it.איקון);
          var chev = document.createElement("span");
          chev.className = "yod-menu-chevron";
          chev.setAttribute("aria-hidden", "true");
          btn.appendChild(chev);
          var sub = buildMenuPanel(it.ילדים, ctrl, depth + 1);
          wrap.appendChild(btn);
          wrap.appendChild(sub);
          var openSub = function () {
            var sibs = wrap.parentNode ? wrap.parentNode.querySelectorAll(":scope > .yod-menu-item.open-sub") : [];
            for (var s = 0; s < sibs.length; s++) {
              if (sibs[s] !== wrap) {
                sibs[s].classList.remove("open-sub");
                sibs[s].classList.remove("yod-sub-flip");
              }
            }
            wrap.classList.add("open-sub");
            wrap.classList.remove("yod-sub-flip");
            // מדידה אחרי הצגה — אם יוצא מהמסך, הופכים צד
            requestAnimationFrame(function () {
              positionSubmenuFlip(wrap, sub);
            });
          };
          btn.addEventListener("mouseenter", openSub);
          btn.addEventListener("focus", openSub);
          btn.addEventListener("click", function (ev) {
            ev.preventDefault();
            ev.stopPropagation();
            if (wrap.classList.contains("open-sub")) {
              wrap.classList.remove("open-sub");
            } else {
              openSub();
            }
          });
          panel.appendChild(wrap);
          return;
        }
        var leaf = document.createElement("button");
        leaf.type = "button";
        leaf.className = "yod-menu-btn yod-menu-item";
        leaf.setAttribute("role", "menuitem");
        if (it.מושבת) leaf.disabled = true;
        decorateMenuLabel(leaf, i18nT(it.טקסט || ""), it.איקון);
        leaf.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          closeAllYodMenus();
          postMenuClick(ctrl, it.ערך);
          if (typeof ctrl.render === "function") ctrl.render();
        });
        panel.appendChild(leaf);
      })(items[i]);
    }
    return panel;
  }

  function postChange(id, type, value, checked) {
    var vals = { מזהה: id, סוג: type, ערך: value };
    if (checked !== undefined) vals.מסומן = !!checked;
    post({ סוג: "אירוע", שם: "שינוי", ערכים: vals });
  }

  function copyTextToClipboard(text) {
    text = text == null ? "" : String(text);
    if (!text) return;
    try {
      if (global.navigator && global.navigator.clipboard && global.navigator.clipboard.writeText) {
        global.navigator.clipboard.writeText(text).catch(function () {
          fallbackCopyText(text);
        });
        return;
      }
    } catch (e) {}
    fallbackCopyText(text);
  }

  function fallbackCopyText(text) {
    try {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.setAttribute("readonly", "");
      ta.style.position = "fixed";
      ta.style.left = "-9999px";
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
    } catch (e2) {}
  }

  function hideYodContextMenu() {
    var m = document.getElementById("yod-ctx-menu");
    if (m && m.parentNode) m.parentNode.removeChild(m);
  }

  function showYodContextMenu(x, y, items) {
    hideYodContextMenu();
    var menu = document.createElement("div");
    menu.id = "yod-ctx-menu";
    menu.className = "yod-ctx-menu";
    menu.setAttribute("role", "menu");
    // יורש dir מהמסמך (RTL/LTR) — לא נועל rtl
    menu.removeAttribute("dir");
    for (var i = 0; i < items.length; i++) {
      (function (it) {
        var btn = document.createElement("button");
        btn.type = "button";
        btn.className = "yod-ctx-item";
        btn.setAttribute("role", "menuitem");
        btn.textContent = it.label;
        if (it.disabled) {
          btn.disabled = true;
          btn.classList.add("disabled");
        }
        btn.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          hideYodContextMenu();
          if (it.onClick) it.onClick();
        });
        menu.appendChild(btn);
      })(items[i]);
    }
    document.body.appendChild(menu);
    var rect = menu.getBoundingClientRect();
    var left = x;
    var top = y;
    if (left + rect.width > window.innerWidth - 8) left = window.innerWidth - rect.width - 8;
    if (top + rect.height > window.innerHeight - 8) top = window.innerHeight - rect.height - 8;
    if (left < 8) left = 8;
    if (top < 8) top = 8;
    menu.style.left = left + "px";
    menu.style.top = top + "px";
  }

  if (!global.__yodCtxMenuBound) {
    global.__yodCtxMenuBound = true;
    document.addEventListener("click", function () {
      hideYodContextMenu();
      closeAllYodMenus();
    });
    document.addEventListener("keydown", function (ev) {
      if (ev.key === "Escape") {
        hideYodContextMenu();
        closeAllYodMenus();
      }
    });
    window.addEventListener("blur", function () {
      hideYodContextMenu();
      closeAllYodMenus();
    });
    window.addEventListener("resize", function () {
      hideYodContextMenu();
      closeAllYodMenus();
    });
    window.addEventListener("scroll", function () {
      hideYodContextMenu();
      closeAllYodMenus();
    }, true);
  }

  var ICO = {
    plus: '<path d="M12 5v14M5 12h14"/>',
    trash: '<path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6M10 11v6M14 11v6"/>',
    sync: '<path d="M21 12a9 9 0 0 1-15.5 6.4M3 12a9 9 0 0 1 15.5-6.4M3 20v-4h4M21 4v4h-4"/>',
    save: '<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2zM17 21v-8H7v8M7 3v5h8"/>',
    folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    broom: '<path d="M5 22h14M9 22V12l-4-2 2-7h10l2 7-4 2v10M12 5V3"/>',
    list: '<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>',
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>',
    "scroll-text": '<path d="M15 12h-5M15 8h-5M19 17V5a2 2 0 0 0-2-2H4"/><path d="M8 21h12a2 2 0 0 0 2-2v-1a1 1 0 0 0-1-1H11a1 1 0 0 0-1 1v1a2 2 0 1 1-4 0V5a2 2 0 1 1 4 0v2a1 1 0 0 0 1 1h3"/>',
    edit: '<path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
    clear: '<path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"/>',
    check: '<circle cx="12" cy="12" r="10"/><path d="M9 12l2 2 4-4"/>',
    "x-circle": '<circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/>',
    clock: '<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>',
    pause: '<circle cx="12" cy="12" r="10"/><path d="M10 8v8M14 8v8"/>',
    globe: '<circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 0 1 0 20M12 2a15 15 0 0 0 0 20"/>',
    alert: '<path d="M10.3 3.9L1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0zM12 9v4M12 17h.01"/>',
    moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
    dots: '<circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/>',
    clipboard: '<path d="M9 4h6a1 1 0 0 1 1 1v1h1a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h1V5a1 1 0 0 1 1-1z"/><path d="M9 13l2 2 4-4"/>'
  };

  var LABEL_ICO = {
    "חדש": "plus",
    "מחק": "trash",
    "סנכרן": "sync",
    "שמור": "save",
    "נקה": "clear",
    "יומן אירועים": "scroll-text",
    "בחר A…": "folder",
    "בחר B…": "folder",
    "בחר מקור": "folder",
    "בחר יעד": "folder",
    "תיקיה": "folder",
    "ערוך": "edit",
    "רשימה": "list",
    "מדריך": "list"
  };

  function svgIcon(name, cls) {
    var d = ICO[name];
    if (!d) return "";
    return (
      '<svg class="yod-ico ' +
      (cls || "") +
      '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
      d +
      "</svg>"
    );
  }

  function decorateButtonLabel(el, text) {
    if (!el) return;
    var t = text == null ? "" : String(text);
    var key = LABEL_ICO[t];
    if (!key) {
      el.textContent = t;
      return;
    }
    el.innerHTML = svgIcon(key, "yod-ico-btn") + "<span>" + t + "</span>";
  }

  function logKindFromText(t) {
    var m = /\]\s*([^:]+):/.exec(String(t || ""));
    return m ? m[1].trim() : "";
  }

  /* ——— תיבת סימון / מתג ——— */
  function CheckLike(opts, asSwitch) {
    Control.call(this, opts);
    this.type = asSwitch ? "מתג" : "תיבת_סימון";
    this.checked = !!(opts.מסומן || opts.checked);
    this.input = null;
    this.labelEl = null;
    this.asSwitch = !!asSwitch;
    // id ASCII ייחודי לקלט — לא לשתף בין פקדים
    this.inputId = "chk_" + ++checkSeq;
  }

  CheckLike.prototype = Object.create(Control.prototype);
  CheckLike.prototype.constructor = CheckLike;

  CheckLike.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = this.asSwitch
      ? "form-check form-switch yod-check"
      : "form-check yod-check";
    wrap.setAttribute("data-check-id", this.inputId);

    var inp = document.createElement("input");
    inp.className = "form-check-input";
    inp.type = "checkbox";
    if (this.asSwitch) inp.setAttribute("role", "switch");
    inp.id = this.inputId;
    inp.name = this.inputId; // name ייחודי — לא קיבוץ בין פקדים
    inp.checked = this.checked;
    inp.setAttribute("data-yod-input", this.id || this.inputId);

    var lab = document.createElement("label");
    lab.className = "form-check-label";
    lab.setAttribute("for", this.inputId);
    lab.textContent = this.text;

    wrap.appendChild(inp);
    wrap.appendChild(lab);
    this.input = inp;
    this.labelEl = lab;
    return wrap;
  };

  CheckLike.prototype.bindEvents = function () {
    var self = this;
    if (!this.input) return;
    this.input.addEventListener("change", function (ev) {
      ev.stopPropagation();
      self.checked = !!self.input.checked;
      postChange(self.id, self.type, self.checked, self.checked);
    });
    this.input.addEventListener("click", function (ev) {
      ev.stopPropagation();
    });
  };

  CheckLike.prototype.applyText = function () {
    if (this.labelEl) this.labelEl.textContent = this.text;
  };

  CheckLike.prototype.setDisabled = function (v) {
    if (this.input) this.input.disabled = !!v;
  };

  CheckLike.prototype.setChecked = function (v) {
    this.checked = !!v;
    if (this.input) this.input.checked = this.checked;
  };

  /* ——— לחצן אפשרויות ——— */
  function RadioGroup(opts) {
    Control.call(this, opts);
    this.type = "לחצן_אפשרויות";
    this.items = normItems(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    // name ASCII ייחודי לכל קבוצה (עברית ב־name מאחדת קבוצות ב־WebView)
    this.groupName = "rg_" + ++radioSeq;
  }

  RadioGroup.prototype = Object.create(Control.prototype);
  RadioGroup.prototype.constructor = RadioGroup;

  RadioGroup.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-radio-group";
    wrap.setAttribute("data-radio-group", this.groupName);
    this.renderItems(wrap);
    return wrap;
  };

  RadioGroup.prototype.renderItems = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    var self = this;
    var gname = this.groupName;
    for (var i = 0; i < this.items.length; i++) {
      var it = this.items[i];
      var row = document.createElement("div");
      row.className = "form-check";
      var inp = document.createElement("input");
      inp.className = "form-check-input";
      inp.type = "radio";
      inp.name = gname;
      inp.id = gname + "_r" + i;
      inp.value = it.ערך;
      if (this.value === it.ערך) inp.checked = true;
      var lab = document.createElement("label");
      lab.className = "form-check-label";
      lab.htmlFor = inp.id;
      lab.textContent = i18nT(it.טקסט);
      (function (input) {
        input.addEventListener("change", function () {
          if (input.checked) {
            self.value = input.value;
            postChange(self.id, self.type, self.value);
          }
        });
      })(inp);
      row.appendChild(inp);
      row.appendChild(lab);
      wrap.appendChild(row);
    }
  };

  RadioGroup.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    if (!this.el) return;
    var inputs = this.el.querySelectorAll('input[type="radio"]');
    for (var i = 0; i < inputs.length; i++) {
      inputs[i].checked = inputs[i].value === this.value;
    }
  };

  RadioGroup.prototype.setItems = function (items) {
    this.items = normItems(items);
    this.renderItems();
  };

  RadioGroup.prototype.setDisabled = function (v) {
    if (!this.el) return;
    var inputs = this.el.querySelectorAll("input");
    for (var i = 0; i < inputs.length; i++) inputs[i].disabled = !!v;
  };

  /* ——— תיבת בחירה ——— */
  function SelectBox(opts) {
    Control.call(this, opts);
    this.type = "תיבת_בחירה";
    this.items = normItems(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
  }

  SelectBox.prototype = Object.create(Control.prototype);
  SelectBox.prototype.constructor = SelectBox;

  SelectBox.prototype.createElement = function () {
    var sel = document.createElement("select");
    sel.className = "form-select";
    this.fillOptions(sel);
    if (this.value) sel.value = this.value;
    return sel;
  };

  SelectBox.prototype.fillOptions = function (sel) {
    sel = sel || this.el;
    if (!sel) return;
    sel.innerHTML = "";
    for (var i = 0; i < this.items.length; i++) {
      var it = this.items[i];
      var opt = document.createElement("option");
      opt.value = it.ערך;
      opt.textContent = i18nT(it.טקסט);
      sel.appendChild(opt);
    }
    if (this.value) sel.value = this.value;
  };

  SelectBox.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("change", function () {
      self.value = self.el.value;
      postChange(self.id, self.type, self.value);
    });
  };

  SelectBox.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    if (this.el) this.el.value = this.value;
  };

  SelectBox.prototype.setItems = function (items) {
    this.items = normItems(items);
    this.fillOptions();
  };

  /* ——— השלמה אוטומטית ——— */
  function Autocomplete(opts) {
    Control.call(this, opts);
    this.type = "השלמה_אוטומטית";
    this.items = normItems(opts.פריטים);
    this.sourcePlaceholder = opts.רמז != null ? String(opts.רמז) : "הקלד לחיפוש…";
    this.placeholder = i18nT(this.sourcePlaceholder);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.sourceText = this.value;
    this.text = this.value;
    this.input = null;
    this.listEl = null;
    this._open = false;
  }
  Autocomplete.prototype = Object.create(Control.prototype);
  Autocomplete.prototype.constructor = Autocomplete;
  Autocomplete.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-autocomplete dropdown";
    var inp = document.createElement("input");
    inp.type = "text";
    inp.className = "form-control";
    inp.placeholder = this.placeholder;
    inp.value = this.value;
    inp.setAttribute("autocomplete", "off");
    inp.setAttribute("role", "combobox");
    inp.setAttribute("aria-autocomplete", "list");
    inp.setAttribute("aria-expanded", "false");
    var list = document.createElement("div");
    list.className = "dropdown-menu yod-ac-menu w-100";
    list.setAttribute("role", "listbox");
    wrap.appendChild(inp);
    wrap.appendChild(list);
    this.input = inp;
    this.listEl = list;
    return wrap;
  };
  Autocomplete.prototype.bindEvents = function () {
    var self = this;
    if (!this.input) return;
    this.input.addEventListener("input", function () {
      self.value = self.input.value;
      self.renderList(self.value);
      self.showList(true);
      postChange(self.id, self.type, self.value);
    });
    this.input.addEventListener("focus", function () {
      self.renderList(self.input.value);
      self.showList(true);
    });
    this.input.addEventListener("keydown", function (ev) {
      if (ev.key === "Escape") {
        self.showList(false);
      } else if (ev.key === "Enter") {
        var first = self.listEl && self.listEl.querySelector(".dropdown-item");
        if (self._open && first) {
          ev.preventDefault();
          first.click();
        }
      }
    });
    document.addEventListener("click", function (ev) {
      if (!self.el) return;
      if (!self.el.contains(ev.target)) self.showList(false);
    });
  };
  Autocomplete.prototype.showList = function (on) {
    this._open = !!on;
    if (!this.listEl) return;
    this.listEl.classList.toggle("show", this._open);
    if (this.input) this.input.setAttribute("aria-expanded", this._open ? "true" : "false");
  };
  Autocomplete.prototype.renderList = function (q) {
    if (!this.listEl) return;
    this.listEl.innerHTML = "";
    q = (q || "").toString().toLowerCase();
    var self = this;
    var count = 0;
    for (var i = 0; i < this.items.length; i++) {
      (function (it) {
        var t = String(it.טקסט || "");
        var v = String(it.ערך || t);
        if (q && t.toLowerCase().indexOf(q) < 0 && v.toLowerCase().indexOf(q) < 0) {
          return;
        }
        count++;
        if (count > 12) return;
        var a = document.createElement("button");
        a.type = "button";
        a.className = "dropdown-item";
        a.setAttribute("role", "option");
        a.textContent = i18nT(t);
        a.addEventListener("click", function () {
          self.value = v;
          if (self.input) self.input.value = t;
          self.showList(false);
          postChange(self.id, self.type, self.value);
        });
        self.listEl.appendChild(a);
      })(this.items[i]);
    }
    if (count === 0) {
      var empty = document.createElement("div");
      empty.className = "dropdown-item disabled text-secondary";
      empty.textContent = i18nT("אין התאמות");
      this.listEl.appendChild(empty);
    }
  };
  Autocomplete.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    if (this.input) this.input.value = this.value;
  };
  Autocomplete.prototype.setItems = function (items) {
    this.items = normItems(items);
    if (this._open) this.renderList(this.input ? this.input.value : "");
  };
  Autocomplete.prototype.setPlaceholder = function (p) {
    this.sourcePlaceholder = p == null ? "" : String(p);
    this.placeholder = i18nT(this.sourcePlaceholder);
    if (this.input) this.input.placeholder = this.placeholder;
  };
  Autocomplete.prototype.applyI18n = function () {
    if (this.sourcePlaceholder != null) {
      this.placeholder = i18nT(this.sourcePlaceholder);
      if (this.input) this.input.placeholder = this.placeholder;
    }
    if (this._open) this.renderList(this.input ? this.input.value : "");
  };

  /* ——— כפתור מפוצל ——— */
  function SplitButton(opts) {
    Control.call(this, opts);
    this.type = "כפתור_מפוצל";
    this.styleName = opts.סגנון || "ראשי";
    this.items = normItems(opts.פריטים);
    this._menuOpen = false;
    this.mainBtn = null;
    this.toggleBtn = null;
    this.menuEl = null;
  }
  SplitButton.prototype = Object.create(Control.prototype);
  SplitButton.prototype.constructor = SplitButton;
  SplitButton.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "btn-group yod-split-btn";
    wrap.setAttribute("role", "group");
    var cls = STYLE_MAP[this.styleName] || STYLE_MAP["ראשי"];
    var main = document.createElement("button");
    main.type = "button";
    main.className = "btn " + cls;
    main.textContent = i18nT(this.sourceText || "פעולה");
    var tog = document.createElement("button");
    tog.type = "button";
    tog.className = "btn " + cls + " dropdown-toggle dropdown-toggle-split";
    tog.setAttribute("aria-expanded", "false");
    tog.setAttribute("aria-label", i18nT("תפריט"));
    var menu = document.createElement("ul");
    menu.className = "dropdown-menu dropdown-menu-end";
    wrap.appendChild(main);
    wrap.appendChild(tog);
    wrap.appendChild(menu);
    this.mainBtn = main;
    this.toggleBtn = tog;
    this.menuEl = menu;
    this.renderMenu();
    return wrap;
  };
  SplitButton.prototype.renderMenu = function () {
    if (!this.menuEl) return;
    this.menuEl.innerHTML = "";
    var self = this;
    for (var i = 0; i < this.items.length; i++) {
      (function (it) {
        var li = document.createElement("li");
        var a = document.createElement("button");
        a.type = "button";
        a.className = "dropdown-item";
        a.textContent = i18nT(it.טקסט);
        a.addEventListener("click", function () {
          self.value = it.ערך != null ? String(it.ערך) : String(it.טקסט || "");
          self.showMenu(false);
          post({
            סוג: "אירוע",
            שם: "לחיצה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: self.value }
          });
          postChange(self.id, self.type, self.value);
        });
        li.appendChild(a);
        self.menuEl.appendChild(li);
      })(this.items[i]);
    }
  };
  SplitButton.prototype.showMenu = function (on) {
    this._menuOpen = !!on;
    if (this.menuEl) this.menuEl.classList.toggle("show", this._menuOpen);
    if (this.toggleBtn) this.toggleBtn.setAttribute("aria-expanded", this._menuOpen ? "true" : "false");
    if (this.el) this.el.classList.toggle("show", this._menuOpen);
  };
  SplitButton.prototype.bindEvents = function () {
    var self = this;
    if (this.mainBtn) {
      this.mainBtn.addEventListener("click", function (ev) {
        ev.preventDefault();
        self.showMenu(false);
        post({
          סוג: "אירוע",
          שם: "לחיצה",
          ערכים: { מזהה: self.id, סוג: self.type, ערך: self.text || "" }
        });
      });
    }
    if (this.toggleBtn) {
      this.toggleBtn.addEventListener("click", function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        self.showMenu(!self._menuOpen);
      });
    }
    document.addEventListener("click", function (ev) {
      if (!self.el) return;
      if (!self.el.contains(ev.target)) self.showMenu(false);
    });
  };
  SplitButton.prototype.applyText = function () {
    if (this.mainBtn) this.mainBtn.textContent = i18nT(this.sourceText || "פעולה");
  };
  SplitButton.prototype.setStyle = function (name) {
    this.styleName = name || "ראשי";
    var cls = STYLE_MAP[this.styleName] || STYLE_MAP["ראשי"];
    if (this.mainBtn) this.mainBtn.className = "btn " + cls;
    if (this.toggleBtn) {
      this.toggleBtn.className = "btn " + cls + " dropdown-toggle dropdown-toggle-split";
    }
  };
  SplitButton.prototype.setItems = function (items) {
    this.items = normItems(items);
    this.renderMenu();
  };

  /* ——— סרגל ——— */
  function Slider(opts) {
    Control.call(this, opts);
    this.type = "סרגל";
    this.min = opts.מינ != null ? Number(opts.מינ) : 0;
    this.max = opts.מקס != null ? Number(opts.מקס) : 100;
    this.value = opts.ערך != null ? Number(opts.ערך) : this.min;
  }

  Slider.prototype = Object.create(Control.prototype);
  Slider.prototype.constructor = Slider;

  Slider.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-slider";
    var inp = document.createElement("input");
    inp.type = "range";
    inp.className = "form-range";
    inp.min = this.min;
    inp.max = this.max;
    inp.value = this.value;
    var lab = document.createElement("div");
    lab.className = "form-text";
    lab.textContent = String(this.value);
    wrap.appendChild(inp);
    wrap.appendChild(lab);
    this.input = inp;
    this.valueLabel = lab;
    return wrap;
  };

  Slider.prototype.bindEvents = function () {
    var self = this;
    this.input.addEventListener("input", function () {
      self.value = Number(self.input.value);
      if (self.valueLabel) self.valueLabel.textContent = String(self.value);
      postChange(self.id, self.type, self.value);
    });
  };

  Slider.prototype.setValue = function (v) {
    this.value = Number(v);
    if (this.input) this.input.value = this.value;
    if (this.valueLabel) this.valueLabel.textContent = String(this.value);
  };

  Slider.prototype.setDisabled = function (v) {
    if (this.input) this.input.disabled = !!v;
  };

  /* ——— בורר צבע ——— */
  function ColorPicker(opts) {
    Control.call(this, opts);
    this.type = "בורר_צבע";
    this.value = opts.ערך != null ? String(opts.ערך) : "#0d6efd";
  }

  ColorPicker.prototype = Object.create(Control.prototype);
  ColorPicker.prototype.constructor = ColorPicker;

  ColorPicker.prototype.createElement = function () {
    var inp = document.createElement("input");
    inp.type = "color";
    inp.className = "form-control form-control-color";
    inp.value = this.value;
    inp.title = this.value;
    return inp;
  };

  ColorPicker.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("input", function () {
      self.value = self.el.value;
      self.el.title = self.value;
      postChange(self.id, self.type, self.value);
    });
  };

  ColorPicker.prototype.setValue = function (v) {
    this.value = v == null ? "#000000" : String(v);
    if (this.el) {
      this.el.value = this.value;
      this.el.title = this.value;
    }
  };

  /* ——— בורר קבצים ——— */
  function FilePicker(opts) {
    Control.call(this, opts);
    this.type = "בורר_קבצים";
    this.hint = opts.רמז != null ? String(opts.רמז) : "בחר קובץ…";
    this.accept = opts.מקבל != null ? String(opts.מקבל) : "";
    this.multiple = !!(opts.מרובה || opts.multiple);
    this.value = "";
    this.names = [];
    this.input = null;
    this.labelEl = null;
  }
  FilePicker.prototype = Object.create(Control.prototype);
  FilePicker.prototype.constructor = FilePicker;
  FilePicker.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-file-picker input-group";
    var inp = document.createElement("input");
    inp.type = "file";
    inp.className = "form-control";
    inp.id = (this.id || "fp") + "_file";
    if (this.accept) inp.accept = this.accept;
    inp.multiple = this.multiple;
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-outline-secondary";
    btn.textContent = this.hint || "בחר קובץ…";
    var lab = document.createElement("span");
    lab.className = "form-control text-secondary yod-file-names";
    lab.textContent = "לא נבחר קובץ";
    lab.setAttribute("aria-live", "polite");
    // input מוסתר ויזואלית אבל זמין; הכפתור פותח אותו
    inp.style.position = "absolute";
    inp.style.width = "1px";
    inp.style.height = "1px";
    inp.style.opacity = "0";
    inp.style.overflow = "hidden";
    wrap.appendChild(inp);
    wrap.appendChild(lab);
    wrap.appendChild(btn);
    this.input = inp;
    this.labelEl = lab;
    this.buttonEl = btn;
    return wrap;
  };
  FilePicker.prototype.bindEvents = function () {
    var self = this;
    if (this.buttonEl) {
      this.buttonEl.addEventListener("click", function (ev) {
        ev.preventDefault();
        self.open();
      });
    }
    if (!this.input) return;
    this.input.addEventListener("change", function () {
      var files = self.input.files;
      var names = [];
      if (files) {
        for (var i = 0; i < files.length; i++) names.push(files[i].name);
      }
      self.names = names;
      self.value = names.join(", ");
      if (self.labelEl) {
        self.labelEl.textContent = self.value || "לא נבחר קובץ";
        self.labelEl.classList.toggle("text-secondary", !self.value);
      }
      post({
        סוג: "אירוע",
        שם: "שינוי",
        ערכים: {
          מזהה: self.id,
          סוג: self.type,
          ערך: self.value,
          שמות: names
        }
      });
    });
  };
  FilePicker.prototype.open = function () {
    if (this.input) this.input.click();
  };
  FilePicker.prototype.setAccept = function (a) {
    this.accept = a == null ? "" : String(a);
    if (this.input) this.input.accept = this.accept;
  };
  FilePicker.prototype.setMultiple = function (v) {
    this.multiple = !!v;
    if (this.input) this.input.multiple = this.multiple;
  };
  FilePicker.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    if (this.labelEl) {
      this.labelEl.textContent = this.value || "לא נבחר קובץ";
    }
  };
  FilePicker.prototype.applyText = function () {
    if (this.buttonEl) this.buttonEl.textContent = this.text || this.hint || "בחר קובץ…";
  };

  /* ——— דוגם צבע (מינימלי: ריבוע לחיץ + hex) ——— */
  function ColorSampler(opts) {
    Control.call(this, opts);
    this.type = "דוגם_צבע";
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    if (!this.value) this.value = "#0d6efd";
    this.swatchEl = null;
    this.valueEl = null;
    this.colorInput = null;
  }
  ColorSampler.prototype = Object.create(Control.prototype);
  ColorSampler.prototype.constructor = ColorSampler;
  ColorSampler.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-eyedropper";
    wrap.title = this.text || i18nT("בחירת צבע");

    var sw = document.createElement("button");
    sw.type = "button";
    sw.className = "yod-eyedropper-swatch";
    sw.setAttribute("aria-label", this.text || i18nT("בחירת צבע"));
    sw.style.background = this.value;

    var val = document.createElement("code");
    val.className = "yod-eyedropper-hex";
    val.textContent = this.value;

    var cin = document.createElement("input");
    cin.type = "color";
    cin.value = /^#[0-9A-Fa-f]{6}$/.test(this.value) ? this.value : "#0d6efd";
    cin.className = "yod-eyedropper-native";
    cin.tabIndex = -1;
    cin.setAttribute("aria-hidden", "true");

    wrap.appendChild(sw);
    wrap.appendChild(val);
    wrap.appendChild(cin);
    this.swatchEl = sw;
    this.valueEl = val;
    this.colorInput = cin;
    return wrap;
  };
  ColorSampler.prototype.bindEvents = function () {
    var self = this;
    if (this.swatchEl) {
      this.swatchEl.addEventListener("click", function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        self.sample();
      });
    }
    if (this.valueEl) {
      this.valueEl.addEventListener("click", function (ev) {
        ev.preventDefault();
        self.sample();
      });
    }
    if (this.colorInput) {
      this.colorInput.addEventListener("input", function () {
        self._applySample(self.colorInput.value);
      });
      this.colorInput.addEventListener("change", function () {
        self._applySample(self.colorInput.value);
      });
    }
  };
  ColorSampler.prototype.applyText = function () {
    if (this.swatchEl) {
      this.swatchEl.setAttribute("aria-label", this.text || i18nT("בחירת צבע"));
    }
    if (this.el) this.el.title = this.text || i18nT("בחירת צבע");
  };
  ColorSampler.prototype.setValue = function (v) {
    this.value = v == null || v === "" ? "#0d6efd" : String(v);
    if (this.swatchEl) this.swatchEl.style.background = this.value;
    if (this.valueEl) this.valueEl.textContent = this.value;
    if (this.colorInput && /^#[0-9A-Fa-f]{6}$/.test(this.value)) {
      this.colorInput.value = this.value;
    }
  };
  ColorSampler.prototype._applySample = function (hex) {
    hex = hex == null ? "" : String(hex);
    this.setValue(hex);
    postChange(this.id, this.type, hex);
  };
  ColorSampler.prototype._openNativeColor = function () {
    var inp = this.colorInput;
    if (!inp) return;
    if (this.value && /^#[0-9A-Fa-f]{6}$/.test(this.value)) {
      inp.value = this.value;
    }
    try {
      inp.click();
    } catch (e) {}
  };
  ColorSampler.prototype.sample = function () {
    var self = this;
    var Eye = global.EyeDropper;
    var canEye =
      typeof Eye === "function" && global.isSecureContext === true;
    if (!canEye) {
      self._openNativeColor();
      return;
    }
    try {
      var eye = new Eye();
      eye
        .open()
        .then(function (result) {
          var hex = result && result.sRGBHex ? result.sRGBHex : "";
          if (hex) self._applySample(hex);
          else self._openNativeColor();
        })
        .catch(function () {
          self._openNativeColor();
        });
    } catch (e) {
      self._openNativeColor();
    }
  };

  /* ——— בורר תאריך ——— */
  var DATE_TYPE_MAP = {
    תאריך: "date",
    זמן: "time",
    שעה: "time",
    תאריך_זמן: "datetime-local",
    תאריך_שעה: "datetime-local",
    date: "date",
    time: "time",
    datetime: "datetime-local"
  };

  function isoToDmy(s) {
    s = s == null ? "" : String(s).trim();
    if (!s) return "";
    var m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s);
    if (m) return m[3] + "/" + m[2] + "/" + m[1];
    return s;
  }

  function dmyToIso(s) {
    s = s == null ? "" : String(s).trim();
    var m = /^(\d{1,2})[\/.\-](\d{1,2})[\/.\-](\d{4})/.exec(s);
    if (!m) return "";
    return m[3] + "-" + ("0" + m[2]).slice(-2) + "-" + ("0" + m[1]).slice(-2);
  }

  function pad2(n) {
    return ("0" + n).slice(-2);
  }

  function todayDmy() {
    var d = new Date();
    return pad2(d.getDate()) + "/" + pad2(d.getMonth() + 1) + "/" + d.getFullYear();
  }

  function todayIso() {
    var d = new Date();
    return d.getFullYear() + "-" + pad2(d.getMonth() + 1) + "-" + pad2(d.getDate());
  }

  function normalizeDmy(s) {
    s = s == null ? "" : String(s).trim();
    if (!s) return "";
    var m = /^(\d{1,2})[\/.\-](\d{1,2})[\/.\-](\d{4})(.*)$/.exec(s);
    if (!m) return s;
    var d = ("0" + m[1]).slice(-2);
    var mo = ("0" + m[2]).slice(-2);
    var rest = (m[4] || "").trim();
    return d + "/" + mo + "/" + m[3] + (rest ? " " + rest : "");
  }

  function DatePicker(opts) {
    Control.call(this, opts);
    this.type = "בורר_תאריך";
    this.fieldKind = opts.סוג_שדה || "תאריך";
    var raw = opts.ערך != null ? String(opts.ערך).trim() : "";
    this.value = raw ? normalizeDmy(isoToDmy(raw)) : "";
    this.input = null;
    this.native = null;
    this.pickBtn = null;
  }

  DatePicker.prototype = Object.create(Control.prototype);
  DatePicker.prototype.constructor = DatePicker;

  DatePicker.prototype.createElement = function () {
    var kind = DATE_TYPE_MAP[this.fieldKind] || "date";
    if (kind === "time") {
      var tin = document.createElement("input");
      tin.type = "time";
      tin.className = "form-control yod-date";
      tin.value = this.value || "";
      this.input = tin;
      return tin;
    }

    var wrap = document.createElement("div");
    wrap.className = "yod-date-picker input-group";

    var inp = document.createElement("input");
    inp.type = "text";
    inp.className = "form-control yod-date";
    inp.lang = "en-GB";
    inp.dir = "ltr";
    inp.style.textAlign = "left";
    inp.setAttribute("inputmode", "numeric");
    inp.setAttribute("autocomplete", "off");
    inp.setAttribute("aria-label", i18nT("תאריך"));
    inp.title = i18nT("תאריך");
    inp.value = this.value || "";

    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-outline-secondary yod-date-btn";
    btn.setAttribute("aria-label", i18nT("בחירת תאריך"));
    btn.setAttribute("aria-haspopup", "dialog");
    btn.title = i18nT("בחירת תאריך");
    btn.innerHTML =
      '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">' +
      '<path d="M3.5 0a.5.5 0 0 1 .5.5V1h8V.5a.5.5 0 0 1 1 0V1h1a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V3a2 2 0 0 1 2-2h1V.5a.5.5 0 0 1 .5-.5zM1 4v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V4H1z"/>' +
      "</svg>";

    var native = document.createElement("input");
    native.type = "date";
    native.className = "yod-date-native";
    native.tabIndex = -1;
    native.setAttribute("aria-hidden", "true");
    native.value = dmyToIso(this.value) || "";

    wrap.appendChild(inp);
    wrap.appendChild(btn);
    wrap.appendChild(native);
    this.input = inp;
    this.pickBtn = btn;
    this.native = native;
    return wrap;
  };

  DatePicker.prototype.placeNativeNearButton = function () {
    if (!this.native || !this.pickBtn) return;
    var r = this.pickBtn.getBoundingClientRect();
    var n = this.native;
    n.style.position = "fixed";
    n.style.left = Math.round(r.left) + "px";
    n.style.top = Math.round(r.bottom - 2) + "px";
    n.style.width = Math.max(2, Math.round(r.width)) + "px";
    n.style.height = "2px";
    n.style.opacity = "0";
    n.style.pointerEvents = "none";
    n.style.border = "0";
    n.style.padding = "0";
    n.style.margin = "0";
    n.style.zIndex = "19060";
  };

  DatePicker.prototype.bindEvents = function () {
    var self = this;
    var kind = DATE_TYPE_MAP[this.fieldKind] || "date";
    if (kind === "time") {
      this.el.addEventListener("change", function () {
        self.value = self.el.value;
        postChange(self.id, self.type, self.value);
      });
      return;
    }

    function commitFromText() {
      var raw = (self.input.value || "").trim();
      self.value = raw ? normalizeDmy(raw) : "";
      self.input.value = self.value;
      if (self.native) {
        self.native.value = dmyToIso(self.value) || "";
      }
      postChange(self.id, self.type, self.value);
    }

    this.input.addEventListener("change", commitFromText);
    this.input.addEventListener("blur", function () {
      if (self.input.value !== self.value) commitFromText();
    });

    function openNative() {
      if (!self.native) return;
      self.placeNativeNearButton();
      self.native.value = dmyToIso(self.input.value) || todayIso();
      try {
        if (typeof self.native.showPicker === "function") {
          self.native.showPicker();
          return;
        }
      } catch (e) {}
      self.native.style.pointerEvents = "auto";
      self.native.click();
      self.native.style.pointerEvents = "none";
    }

    this.pickBtn.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      openNative();
    });

    this.native.addEventListener("change", function () {
      if (!self.native.value) {
        self.value = "";
        self.input.value = "";
        postChange(self.id, self.type, "");
        return;
      }
      var dmy = isoToDmy(self.native.value);
      if (kind === "datetime-local") {
        var timePart = "";
        var m = /\s+(\d{1,2}:\d{2})/.exec(self.input.value);
        if (m) timePart = " " + m[1];
        else timePart = " 00:00";
        dmy = dmy + timePart;
      }
      self.value = normalizeDmy(dmy);
      self.input.value = self.value;
      postChange(self.id, self.type, self.value);
    });
  };

  DatePicker.prototype.setValue = function (v) {
    var kind = DATE_TYPE_MAP[this.fieldKind] || "date";
    if (kind === "time") {
      this.value = v == null ? "" : String(v);
      if (this.el) this.el.value = this.value;
      return;
    }
    var raw = v == null ? "" : String(v).trim();
    this.value = raw ? normalizeDmy(isoToDmy(raw)) : "";
    if (this.input) this.input.value = this.value;
    if (this.native) this.native.value = dmyToIso(this.value) || "";
  };

  /* ——— בורר שעה ——— */
  function nowTime(withSec) {
    var d = new Date();
    var t = pad2(d.getHours()) + ":" + pad2(d.getMinutes());
    if (withSec) t += ":" + pad2(d.getSeconds());
    return t;
  }

  function normalizeTime(s, withSec) {
    s = s == null ? "" : String(s).trim();
    if (!s) return "";
    var m = /^(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?/.exec(s);
    if (!m) return s;
    var out = pad2(m[1]) + ":" + pad2(m[2]);
    if (withSec) {
      out += ":" + pad2(m[3] != null ? m[3] : 0);
    }
    return out;
  }

  function fillTimeSelect(sel, max, selected) {
    sel.innerHTML = "";
    for (var i = 0; i <= max; i++) {
      var opt = document.createElement("option");
      opt.value = pad2(i);
      opt.textContent = pad2(i);
      if (pad2(i) === selected) opt.selected = true;
      sel.appendChild(opt);
    }
  }

  function TimePicker(opts) {
    Control.call(this, opts);
    this.type = "בורר_שעה";
    this.withSeconds = !!(opts.עם_שניות === true || opts.עם_שניות === "אמת" || opts.עם_שניות === 1);
    var raw = opts.ערך != null ? String(opts.ערך).trim() : "";
    this.value = raw ? normalizeTime(raw, this.withSeconds) : "";
    this.input = null;
    this.pickBtn = null;
    this.panel = null;
    this.selH = null;
    this.selM = null;
    this.selS = null;
    this.open = false;
    this._docClose = null;
  }

  TimePicker.prototype = Object.create(Control.prototype);
  TimePicker.prototype.constructor = TimePicker;

  TimePicker.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-time-picker input-group";

    var inp = document.createElement("input");
    inp.type = "text";
    inp.className = "form-control yod-time";
    inp.lang = "en-GB";
    inp.dir = "ltr";
    inp.style.textAlign = "left";
    inp.setAttribute("inputmode", "numeric");
    inp.setAttribute("autocomplete", "off");
    inp.setAttribute("aria-label", i18nT("שעה"));
    inp.title = i18nT("שעה");
    inp.value = this.value || "";

    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-outline-secondary yod-time-btn";
    btn.setAttribute("aria-label", i18nT("בחירת שעה"));
    btn.setAttribute("aria-haspopup", "dialog");
    btn.setAttribute("aria-expanded", "false");
    btn.title = i18nT("בחירת שעה");
    btn.innerHTML =
      '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">' +
      '<path d="M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>' +
      '<path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0z"/>' +
      "</svg>";

    wrap.appendChild(inp);
    wrap.appendChild(btn);
    this.input = inp;
    this.pickBtn = btn;
    return wrap;
  };

  TimePicker.prototype.ensurePanel = function () {
    if (this.panel) return;
    var panel = document.createElement("div");
    panel.className = "yod-time-panel";
    panel.setAttribute("role", "dialog");
    panel.setAttribute("aria-label", i18nT("בחירת שעה"));
    panel.setAttribute("aria-hidden", "true");

    var body = document.createElement("div");
    body.className = "yod-time-panel-body";

    var selH = document.createElement("select");
    selH.className = "form-select form-select-sm yod-time-sel";
    selH.setAttribute("aria-label", i18nT("שעות"));
    var colon1 = document.createElement("span");
    colon1.className = "yod-time-colon";
    colon1.textContent = ":";
    var selM = document.createElement("select");
    selM.className = "form-select form-select-sm yod-time-sel";
    selM.setAttribute("aria-label", i18nT("דקות"));

    body.appendChild(selH);
    body.appendChild(colon1);
    body.appendChild(selM);

    var selS = null;
    if (this.withSeconds) {
      var colon2 = document.createElement("span");
      colon2.className = "yod-time-colon";
      colon2.textContent = ":";
      selS = document.createElement("select");
      selS.className = "form-select form-select-sm yod-time-sel";
      selS.setAttribute("aria-label", i18nT("שניות"));
      body.appendChild(colon2);
      body.appendChild(selS);
    }

    var foot = document.createElement("div");
    foot.className = "yod-time-panel-footer";
    var save = document.createElement("button");
    save.type = "button";
    save.className = "btn btn-sm yod-time-save";
    save.textContent = "שמור";
    foot.appendChild(save);

    panel.appendChild(body);
    panel.appendChild(foot);
    document.body.appendChild(panel);

    this.panel = panel;
    this.selH = selH;
    this.selM = selM;
    this.selS = selS;

    var self = this;
    save.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      self.applyFromPanel();
      self.hidePanel();
    });
    panel.addEventListener("click", function (ev) {
      ev.stopPropagation();
    });
  };

  TimePicker.prototype.syncPanelFromValue = function () {
    var t = normalizeTime(this.input ? this.input.value : this.value, this.withSeconds) || nowTime(this.withSeconds);
    var parts = t.split(":");
    fillTimeSelect(this.selH, 23, parts[0] || "00");
    fillTimeSelect(this.selM, 59, parts[1] || "00");
    if (this.selS) fillTimeSelect(this.selS, 59, parts[2] || "00");
  };

  TimePicker.prototype.applyFromPanel = function () {
    var t = (this.selH ? this.selH.value : "00") + ":" + (this.selM ? this.selM.value : "00");
    if (this.withSeconds && this.selS) t += ":" + this.selS.value;
    this.value = normalizeTime(t, this.withSeconds);
    if (this.input) this.input.value = this.value;
    postChange(this.id, this.type, this.value);
  };

  TimePicker.prototype.positionPanel = function () {
    if (!this.panel || !this.el) return;
    var r = this.el.getBoundingClientRect();
    var pw = this.panel.offsetWidth || 160;
    var ph = this.panel.offsetHeight || 100;
    var top = r.bottom + 6;
    var left = r.left;
    if (left + pw > window.innerWidth - 8) left = Math.max(8, window.innerWidth - pw - 8);
    if (top + ph > window.innerHeight - 8 && r.top - ph - 6 > 8) {
      top = r.top - ph - 6;
    }
    this.panel.style.top = Math.round(top) + "px";
    this.panel.style.left = Math.round(left) + "px";
  };

  TimePicker.prototype.showPanel = function () {
    this.ensurePanel();
    this.syncPanelFromValue();
    this.panel.classList.add("yod-time-panel-open");
    this.panel.removeAttribute("aria-hidden");
    this.open = true;
    if (this.pickBtn) this.pickBtn.setAttribute("aria-expanded", "true");
    this.positionPanel();
    var self = this;
    if (this._docClose) document.removeEventListener("click", this._docClose, true);
    this._docClose = function (ev) {
      if (!self.open) return;
      var t = ev.target;
      if (self.panel && self.panel.contains(t)) return;
      if (self.el && self.el.contains(t)) return;
      self.hidePanel();
    };
    setTimeout(function () {
      document.addEventListener("click", self._docClose, true);
    }, 0);
  };

  TimePicker.prototype.hidePanel = function () {
    if (this.panel) {
      this.panel.classList.remove("yod-time-panel-open");
      this.panel.setAttribute("aria-hidden", "true");
    }
    this.open = false;
    if (this.pickBtn) this.pickBtn.setAttribute("aria-expanded", "false");
    if (this._docClose) {
      document.removeEventListener("click", this._docClose, true);
      this._docClose = null;
    }
  };

  TimePicker.prototype.bindEvents = function () {
    var self = this;

    function commitFromText() {
      var raw = (self.input.value || "").trim();
      self.value = raw ? normalizeTime(raw, self.withSeconds) : "";
      self.input.value = self.value;
      postChange(self.id, self.type, self.value);
    }

    this.input.addEventListener("change", commitFromText);
    this.input.addEventListener("blur", function () {
      if (self.input.value !== self.value) commitFromText();
    });

    this.pickBtn.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      if (self.open) self.hidePanel();
      else self.showPanel();
    });
  };

  TimePicker.prototype.setValue = function (v) {
    var raw = v == null ? "" : String(v).trim();
    this.value = raw ? normalizeTime(raw, this.withSeconds) : "";
    if (this.input) this.input.value = this.value;
    if (this.open) this.syncPanelFromValue();
  };

  TimePicker.prototype.remove = function () {
    this.hidePanel();
    if (this.panel && this.panel.parentNode) this.panel.parentNode.removeChild(this.panel);
    this.panel = null;
    Control.prototype.remove.call(this);
  };

  /* ——— מפריד / מרווח ——— */
  function Divider(opts) {
    Control.call(this, opts);
    this.type = "מפריד";
  }
  Divider.prototype = Object.create(Control.prototype);
  Divider.prototype.constructor = Divider;
  Divider.prototype.createElement = function () {
    var hr = document.createElement("hr");
    hr.className = "yod-divider";
    return hr;
  };

  function Spacer(opts) {
    Control.call(this, opts);
    this.type = "מרווח";
    this.height = opts.גובה != null ? Number(opts.גובה) : 16;
  }
  Spacer.prototype = Object.create(Control.prototype);
  Spacer.prototype.constructor = Spacer;
  Spacer.prototype.createElement = function () {
    var d = document.createElement("div");
    d.className = "yod-spacer";
    d.style.height = this.height + "px";
    return d;
  };
  Spacer.prototype.setHeight = function (h) {
    this.height = h >= 0 ? h : 16;
    if (this.el) this.el.style.height = this.height + "px";
  };

  /* ——— תג ——— */
  var BADGE_STYLE = {
    ראשי: "text-bg-primary",
    משני: "text-bg-secondary",
    סכנה: "text-bg-danger",
    הצלחה: "text-bg-success",
    אזהרה: "text-bg-warning",
    מידע: "text-bg-info"
  };

  function Badge(opts) {
    Control.call(this, opts);
    this.type = "תג";
    this.styleName = opts.סגנון || "ראשי";
  }
  Badge.prototype = Object.create(Control.prototype);
  Badge.prototype.constructor = Badge;
  Badge.prototype.createElement = function () {
    var s = document.createElement("span");
    s.className = "badge " + (BADGE_STYLE[this.styleName] || BADGE_STYLE["ראשי"]);
    s.textContent = this.text;
    return s;
  };
  Badge.prototype.applyText = function () {
    if (this.el) this.el.textContent = this.text;
  };
  Badge.prototype.setStyle = function (name) {
    this.styleName = name || "ראשי";
    if (this.el) {
      this.el.className =
        "badge yod-control " + (BADGE_STYLE[this.styleName] || BADGE_STYLE["ראשי"]);
    }
  };

  /* ——— התרעה ——— */
  var ALERT_STYLE = {
    מידע: "alert-info",
    הצלחה: "alert-success",
    אזהרה: "alert-warning",
    סכנה: "alert-danger",
    ראשי: "alert-primary",
    משני: "alert-secondary"
  };

  function AlertBox(opts) {
    Control.call(this, opts);
    this.type = "התרעה";
    this.styleName = opts.סגנון || "מידע";
  }
  AlertBox.prototype = Object.create(Control.prototype);
  AlertBox.prototype.constructor = AlertBox;
  AlertBox.prototype.createElement = function () {
    var d = document.createElement("div");
    d.className = "alert " + (ALERT_STYLE[this.styleName] || ALERT_STYLE["מידע"]);
    d.setAttribute("role", "alert");
    d.textContent = this.text;
    return d;
  };
  AlertBox.prototype.applyText = function () {
    if (this.el) this.el.textContent = this.text;
  };
  AlertBox.prototype.setStyle = function (name) {
    this.styleName = name || "מידע";
    if (this.el) {
      this.el.className =
        "alert yod-control " + (ALERT_STYLE[this.styleName] || ALERT_STYLE["מידע"]);
    }
  };

  /* ——— פס התקדמות ——— */
  function Progress(opts) {
    Control.call(this, opts);
    this.type = "פס_התקדמות";
    this.value = opts.ערך != null ? Number(opts.ערך) : 0;
    this.bar = null;
  }
  Progress.prototype = Object.create(Control.prototype);
  Progress.prototype.constructor = Progress;
  Progress.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "progress";
    wrap.setAttribute("role", "progressbar");
    var bar = document.createElement("div");
    bar.className = "progress-bar";
    var pct = Math.max(0, Math.min(100, this.value));
    bar.style.width = pct + "%";
    bar.textContent = Math.round(pct) + "%";
    wrap.appendChild(bar);
    this.bar = bar;
    return wrap;
  };
  Progress.prototype.setValue = function (v) {
    this.value = Number(v) || 0;
    var pct = Math.max(0, Math.min(100, this.value));
    if (this.bar) {
      this.bar.style.width = pct + "%";
      this.bar.textContent = Math.round(pct) + "%";
    }
    if (this.el) {
      this.el.setAttribute("aria-valuenow", String(Math.round(pct)));
    }
  };

  /* ——— טעינה ——— */
  function Spinner(opts) {
    Control.call(this, opts);
    this.type = "טעינה";
  }
  Spinner.prototype = Object.create(Control.prototype);
  Spinner.prototype.constructor = Spinner;
  Spinner.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-spinner d-flex align-items-center gap-2";
    var sp = document.createElement("div");
    sp.className = "spinner-border spinner-border-sm";
    sp.setAttribute("role", "status");
    var lab = document.createElement("span");
    lab.textContent = this.text || "";
    wrap.appendChild(sp);
    wrap.appendChild(lab);
    this.labelEl = lab;
    return wrap;
  };
  Spinner.prototype.applyText = function () {
    if (this.labelEl) this.labelEl.textContent = this.text || "";
  };

  function tippyApi() {
    var t = global.tippy;
    if (typeof t === "function") return t;
    if (t && typeof t.default === "function") return t.default;
    return null;
  }

  function ensureBootstrapModal(el) {
    if (!el) return null;
    try {
      if (global.bootstrap && global.bootstrap.Modal) {
        return global.bootstrap.Modal.getOrCreateInstance(el);
      }
    } catch (e) {}
    return null;
  }

  function ensureBootstrapToast(el) {
    if (!el) return null;
    try {
      if (global.bootstrap && global.bootstrap.Toast) {
        return global.bootstrap.Toast.getOrCreateInstance(el, { delay: 3500, autohide: true });
      }
    } catch (e) {}
    return null;
  }

  function itemTitle(it) {
    if (typeof it === "string" || typeof it === "number") return String(it);
    if (!it || typeof it !== "object") return "";
    if (it.טקסט != null) return String(it.טקסט);
    if (it.כותרת != null) return String(it.כותרת);
    if (it.title != null) return String(it.title);
    return String(it.ערך != null ? it.ערך : it.value != null ? it.value : "");
  }

  function itemBody(it) {
    if (!it || typeof it !== "object") return "";
    if (it.גוף != null) return String(it.גוף);
    if (it.body != null) return String(it.body);
    if (it.ערך != null && it.טקסט != null) return String(it.ערך);
    return "";
  }

  function attachCssTooltip(target, text) {
    if (!target) return;
    target.setAttribute("data-yod-tip", text || "");
    target.classList.add("yod-has-tip");
    target.setAttribute("title", text || "");
  }

  /* ——— הסבר צף (tippy.js + fallback) ——— */
  function TooltipCtrl(opts) {
    Control.call(this, opts);
    this.type = "הסבר_צף";
    this.sourceContent = opts.תוכן != null ? String(opts.תוכן) : "";
    this.content = i18nT(this.sourceContent);
    this.sourceText = opts.טקסט != null ? String(opts.טקסט) : "?";
    this.text = i18nT(this.sourceText);
    this.targetId = "";
    this._tip = null;
  }
  TooltipCtrl.prototype = Object.create(Control.prototype);
  TooltipCtrl.prototype.constructor = TooltipCtrl;
  TooltipCtrl.prototype.createElement = function () {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-outline-secondary btn-sm rounded-circle";
    btn.textContent = this.text || "?";
    btn.setAttribute("aria-label", i18nT("הסבר"));
    return btn;
  };
  TooltipCtrl.prototype.bindEvents = function () {
    var self = this;
    setTimeout(function () {
      self.attachTippy(self.el);
    }, 30);
  };
  TooltipCtrl.prototype.attachTippy = function (target) {
    if (!target) return;
    var text = i18nT(this.sourceContent || this.content);
    if (!text) text = i18nT("הסבר");
    var api = tippyApi();
    if (this._tip) {
      try {
        this._tip.destroy();
      } catch (e) {}
      this._tip = null;
    }
    if (!api) {
      attachCssTooltip(target, text);
      return;
    }
    try {
      this._tip = api(target, {
        content: text,
        allowHTML: false,
        theme: "light-border",
        placement: "top",
        trigger: "mouseenter focus",
        interactive: false,
        appendTo: function () {
          return document.body;
        },
        zIndex: 99999
      });
      target.classList.remove("yod-has-tip");
      target.removeAttribute("title");
    } catch (e) {
      attachCssTooltip(target, text);
    }
  };
  TooltipCtrl.prototype.setContent = function (c) {
    this.sourceContent = c == null ? "" : String(c);
    this.content = i18nT(this.sourceContent);
    if (this._tip) {
      try {
        this._tip.setContent(this.content);
      } catch (e) {}
    } else if (this.el) {
      attachCssTooltip(this.el, this.content);
    }
  };
  TooltipCtrl.prototype.applyI18n = function () {
    this.content = i18nT(this.sourceContent || "");
    this.text = i18nT(this.sourceText || "?");
    if (this.el) {
      this.el.textContent = this.text || "?";
      this.el.setAttribute("aria-label", i18nT("הסבר"));
    }
    var t = this.targetId ? document.getElementById(this.targetId) : this.el;
    if (!t && this.targetId) {
      var c = controls[this.targetId];
      t = c && c.el ? c.el : null;
    }
    this.attachTippy(t || this.el);
  };
  TooltipCtrl.prototype.setTarget = function (id) {
    this.targetId = id || "";
    var t = this.targetId ? document.getElementById(this.targetId) : this.el;
    if (!t && this.targetId) {
      var c = controls[this.targetId];
      t = c && c.el ? c.el : null;
    }
    if (t) this.attachTippy(t);
  };
  TooltipCtrl.prototype.applyText = function () {
    if (this.el && this.el.tagName === "BUTTON") this.el.textContent = this.text;
  };
  TooltipCtrl.prototype.remove = function () {
    if (this._tip) {
      try {
        this._tip.destroy();
      } catch (e) {}
      this._tip = null;
    }
    Control.prototype.remove.call(this);
  };

  /* ——— Toast ——— */
  var TOAST_BG = {
    מידע: "text-bg-info",
    הצלחה: "text-bg-success",
    אזהרה: "text-bg-warning",
    סכנה: "text-bg-danger",
    ראשי: "text-bg-primary",
    משני: "text-bg-secondary"
  };
  function ToastCtrl(opts) {
    Control.call(this, opts);
    this.type = "הודעה_קופצת";
    this.styleName = opts.סגנון || "מידע";
    this._bs = null;
  }
  ToastCtrl.prototype = Object.create(Control.prototype);
  ToastCtrl.prototype.constructor = ToastCtrl;
  ToastCtrl.prototype.createElement = function () {
    var toast = document.createElement("div");
    toast.className =
      "toast align-items-center border-0 " +
      (TOAST_BG[this.styleName] || TOAST_BG["מידע"]);
    toast.setAttribute("role", "alert");
    toast.innerHTML =
      '<div class="d-flex"><div class="toast-body"></div>' +
      '<button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button></div>';
    toast.querySelector(".toast-body").textContent = this.text;
    return toast;
  };
  ToastCtrl.prototype.mount = function () {
    if (!this.el) {
      this.el = this.createElement();
      this.el.id = this.id;
      this.el.classList.add("yod-control");
      this.el.setAttribute("data-yod-id", this.id);
      this.el.setAttribute("data-yod-type", this.type);
      this.el.style.display = "none";
    }
    var box = document.getElementById("yod-toasts") || document.body;
    box.appendChild(this.el);
    markHasControls();
    controls[this.id] = this;
    // Bootstrap Toast נוצר רק ב־show — לא מראש
    this._bs = null;
    return this;
  };
  ToastCtrl.prototype.applyText = function () {
    var b = this.el && this.el.querySelector(".toast-body");
    if (b) b.textContent = this.text;
  };
  ToastCtrl.prototype.show = function () {
    if (!this.el) return;
    if (!this.el.parentNode) {
      var box = document.getElementById("yod-toasts") || document.body;
      box.appendChild(this.el);
    }
    // תמיד נתיב ידני — אמין גם בלחיצה הראשונה ב־WebView2
    this.el.style.display = "block";
    this.el.classList.add("show");
    this.el.classList.add("showing");
    try {
      if (!this._bs) this._bs = ensureBootstrapToast(this.el);
      if (this._bs) this._bs.show();
    } catch (e) {}
    var self = this;
    clearTimeout(this._hideTimer);
    this._hideTimer = setTimeout(function () {
      self.hide();
    }, 3500);
  };
  ToastCtrl.prototype.hide = function () {
    clearTimeout(this._hideTimer);
    if (this._bs) {
      try {
        this._bs.hide();
      } catch (e) {}
    }
    if (this.el) {
      this.el.classList.remove("show");
      this.el.classList.remove("showing");
      this.el.style.display = "none";
    }
  };

  /* ——— Modal ——— */
  function ModalCtrl(opts) {
    Control.call(this, opts);
    this.type = "חלון_מודאלי";
    this.sourceTitle = opts.כותרת != null ? String(opts.כותרת) : "";
    this.title = i18nT(this.sourceTitle);
    this.sourceBody = opts.גוף != null ? String(opts.גוף) : "";
    this.body = i18nT(this.sourceBody);
    this.sourceActionLabel = opts.כפתור_פעולה != null ? String(opts.כפתור_פעולה) : "";
    this.actionLabel = i18nT(this.sourceActionLabel);
    this._bs = null;
    this.titleEl = null;
    this.bodyEl = null;
    this.actionBtn = null;
  }
  ModalCtrl.prototype = Object.create(Control.prototype);
  ModalCtrl.prototype.constructor = ModalCtrl;
  ModalCtrl.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-modal";
    wrap.setAttribute("aria-hidden", "true");
    wrap.style.display = "none";
    wrap.innerHTML =
      '<div class="yod-modal-backdrop" data-yod-modal-close="1"></div>' +
      '<div class="yod-modal-dialog" role="dialog" aria-modal="true">' +
      '<div class="yod-modal-content">' +
      '<div class="yod-modal-header"><h5 class="modal-title"></h5>' +
      '<button type="button" class="btn-close" data-yod-modal-close="1" aria-label="' +
      i18nT("סגור") +
      '"></button></div>' +
      '<div class="modal-body yod-modal-body">' +
      '<div class="yod-modal-body-text"></div>' +
      '<div class="yod-modal-body-content"></div>' +
      "</div>" +
      '<div class="yod-modal-footer">' +
      '<button type="button" class="btn btn-primary yod-modal-action" style="display:none"></button>' +
      '<button type="button" class="btn btn-secondary" data-yod-modal-close="1">' +
      i18nT("סגור") +
      "</button></div>" +
      "</div></div>";
    this.titleEl = wrap.querySelector(".modal-title");
    this.textEl = wrap.querySelector(".yod-modal-body-text");
    this.contentEl = wrap.querySelector(".yod-modal-body-content");
    this.bodyEl = wrap.querySelector(".yod-modal-body");
    this.actionBtn = wrap.querySelector(".yod-modal-action");
    this.titleEl.textContent = this.title;
    this._syncBodyText();
    this._syncActionBtn();
    return wrap;
  };
  ModalCtrl.prototype.mountTarget = function () {
    return this.contentEl || this.bodyEl || this.el;
  };
  ModalCtrl.prototype._syncBodyText = function () {
    if (!this.textEl) return;
    var t = this.body || "";
    this.textEl.textContent = t;
    this.textEl.style.display = t ? "" : "none";
    if (this.contentEl && this.contentEl.childNodes.length) {
      if (this.el && this.el.querySelector) {
        var dlg = this.el.querySelector(".yod-modal-dialog");
        if (dlg) dlg.classList.add("yod-modal-wide");
      }
    }
  };
  ModalCtrl.prototype._syncActionBtn = function () {
    if (!this.actionBtn) return;
    var lab = this.actionLabel || "";
    if (lab) {
      this.actionBtn.textContent = lab;
      this.actionBtn.style.display = "";
    } else {
      this.actionBtn.style.display = "none";
    }
  };
  ModalCtrl.prototype.mount = function () {
    if (!this.el) {
      this.el = this.createElement();
      this.el.id = this.id;
      this.el.setAttribute("data-yod-id", this.id);
      this.el.setAttribute("data-yod-type", this.type);
      this.bindEvents();
    }
    document.body.appendChild(this.el);
    controls[this.id] = this;
    return this;
  };
  ModalCtrl.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("click", function (ev) {
      var t = ev.target;
      if (t && t.classList && t.classList.contains("yod-modal-action")) {
        ev.preventDefault();
        post({ סוג: "אירוע", שם: "פעולה", ערכים: { מזהה: self.id, סוג: "חלון_מודאלי" } });
        return;
      }
      if (t && t.getAttribute && t.getAttribute("data-yod-modal-close") === "1") {
        self.hide();
      }
    });
  };
  ModalCtrl.prototype.hide = function () {
    if (!this.el) return;
    this.el.classList.remove("yod-modal-open");
    this.el.setAttribute("aria-hidden", "true");
    this.el.style.setProperty("display", "none", "important");
    this.el.style.setProperty("pointer-events", "none", "important");
    document.body.classList.remove("yod-modal-open-body");
    post({ סוג: "אירוע", שם: "סגירה", ערכים: { מזהה: this.id, סוג: "חלון_מודאלי" } });
  };
  ModalCtrl.prototype.setTitle = function (raw) {
    this.sourceTitle = raw == null ? "" : String(raw);
    this.title = i18nT(this.sourceTitle);
    if (this.titleEl) this.titleEl.textContent = this.title;
  };
  ModalCtrl.prototype.setBody = function (raw) {
    this.sourceBody = raw == null ? "" : String(raw);
    this.body = i18nT(this.sourceBody);
    this._syncBodyText();
  };
  ModalCtrl.prototype.setActionLabel = function (raw) {
    this.sourceActionLabel = raw == null ? "" : String(raw);
    this.actionLabel = i18nT(this.sourceActionLabel);
    this._syncActionBtn();
  };
  ModalCtrl.prototype.show = function () {
    if (!this.el) return;
    if (!this.el.parentNode) document.body.appendChild(this.el);
    this._syncBodyText();
    if (this.contentEl && this.contentEl.childNodes.length) {
      var dlg = this.el.querySelector(".yod-modal-dialog");
      if (dlg) dlg.classList.add("yod-modal-wide");
    }
    this.el.classList.add("yod-modal-open");
    this.el.removeAttribute("aria-hidden");
    this.el.style.setProperty("display", "flex", "important");
    this.el.style.setProperty("pointer-events", "auto", "important");
    document.body.classList.add("yod-modal-open-body");
  };
  ModalCtrl.prototype.remove = function () {
    if (this.el && this.el.classList.contains("yod-modal-open")) {
      this.el.style.display = "none";
      this.el.classList.remove("yod-modal-open");
      document.body.classList.remove("yod-modal-open-body");
    }
    Control.prototype.remove.call(this);
  };

  /* ——— חלון צף (Popover) ——— */
  function PopoverCtrl(opts) {
    Control.call(this, opts);
    this.type = "חלון_צף";
    this.sourceTitle = opts.כותרת != null ? String(opts.כותרת) : "";
    this.title = i18nT(this.sourceTitle);
    this.body = opts.גוף != null ? String(opts.גוף) : "";
    this.text = opts.טקסט != null ? String(opts.טקסט) : "מידע";
    this.targetId = "";
    this.open = false;
    this.panel = null;
    this.titleEl = null;
    this.bodyEl = null;
    this._docClose = null;
  }
  PopoverCtrl.prototype = Object.create(Control.prototype);
  PopoverCtrl.prototype.constructor = PopoverCtrl;
  PopoverCtrl.prototype.createElement = function () {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-outline-info btn-sm";
    btn.textContent = this.text || "מידע";
    btn.setAttribute("aria-haspopup", "dialog");
    btn.setAttribute("aria-expanded", "false");
    return btn;
  };
  PopoverCtrl.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    this.ensurePanel();
    return this;
  };
  PopoverCtrl.prototype.ensurePanel = function () {
    if (this.panel) return;
    var panel = document.createElement("div");
    panel.className = "yod-popover";
    panel.setAttribute("role", "dialog");
    panel.setAttribute("aria-hidden", "true");
    panel.innerHTML =
      '<div class="yod-popover-arrow"></div>' +
      '<div class="yod-popover-header">' +
      '<strong class="yod-popover-title"></strong>' +
      '<button type="button" class="btn-close btn-close-white yod-popover-close" aria-label="' +
      i18nT("סגור") +
      '"></button>' +
      "</div>" +
      '<div class="yod-popover-body"></div>';
    this.titleEl = panel.querySelector(".yod-popover-title");
    this.bodyEl = panel.querySelector(".yod-popover-body");
    this.titleEl.textContent = this.title;
    this.bodyEl.textContent = this.body;
    document.body.appendChild(panel);
    this.panel = panel;
    var self = this;
    panel.querySelector(".yod-popover-close").addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      self.hide();
    });
    panel.addEventListener("click", function (ev) {
      ev.stopPropagation();
    });
  };
  PopoverCtrl.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      if (self.open) self.hide();
      else self.show();
      post({
        סוג: "אירוע",
        שם: "לחיצה",
        ערכים: { מזהה: self.id, סוג: self.type }
      });
    });
  };
  PopoverCtrl.prototype.getAnchor = function () {
    if (this.targetId) {
      var t = document.getElementById(this.targetId);
      if (!t) {
        var c = controls[this.targetId];
        t = c && c.el ? c.el : null;
      }
      if (t) return t;
    }
    return this.el;
  };
  PopoverCtrl.prototype.position = function () {
    if (!this.panel) return;
    var anchor = this.getAnchor();
    if (!anchor) return;
    var r = anchor.getBoundingClientRect();
    var pw = this.panel.offsetWidth || 240;
    var ph = this.panel.offsetHeight || 80;
    var top = r.bottom + 10;
    var left = r.left + r.width / 2 - pw / 2;
    var place = "bottom";
    if (top + ph > window.innerHeight - 8 && r.top - ph - 10 > 8) {
      top = r.top - ph - 10;
      place = "top";
    }
    if (left < 8) left = 8;
    if (left + pw > window.innerWidth - 8) left = Math.max(8, window.innerWidth - pw - 8);
    this.panel.style.top = Math.round(top) + "px";
    this.panel.style.left = Math.round(left) + "px";
    this.panel.setAttribute("data-place", place);
    var arrow = this.panel.querySelector(".yod-popover-arrow");
    if (arrow) {
      var ax = r.left + r.width / 2 - left;
      arrow.style.left = Math.max(12, Math.min(pw - 12, ax)) + "px";
    }
  };
  PopoverCtrl.prototype.show = function () {
    this.ensurePanel();
    if (!this.panel) return;
    this.panel.classList.add("yod-popover-open");
    this.panel.removeAttribute("aria-hidden");
    this.open = true;
    if (this.el) this.el.setAttribute("aria-expanded", "true");
    this.position();
    var self = this;
    if (this._docClose) {
      document.removeEventListener("click", this._docClose, true);
    }
    this._docClose = function (ev) {
      if (!self.open) return;
      var t = ev.target;
      if (self.panel && self.panel.contains(t)) return;
      if (self.el && self.el.contains(t)) return;
      var a = self.getAnchor();
      if (a && a !== self.el && a.contains && a.contains(t)) return;
      self.hide();
    };
    setTimeout(function () {
      document.addEventListener("click", self._docClose, true);
    }, 0);
    window.addEventListener("resize", this._onReposition || (this._onReposition = function () {
      if (self.open) self.position();
    }));
  };
  PopoverCtrl.prototype.hide = function () {
    var wasOpen = this.open;
    if (this.panel) {
      this.panel.classList.remove("yod-popover-open");
      this.panel.setAttribute("aria-hidden", "true");
    }
    this.open = false;
    if (this.el) this.el.setAttribute("aria-expanded", "false");
    if (this._docClose) {
      document.removeEventListener("click", this._docClose, true);
      this._docClose = null;
    }
    if (wasOpen) {
      post({ סוג: "אירוע", שם: "סגירה", ערכים: { מזהה: this.id, סוג: "חלון_צף" } });
    }
  };
  PopoverCtrl.prototype.setTitle = function (raw) {
    this.sourceTitle = raw == null ? "" : String(raw);
    this.title = i18nT(this.sourceTitle);
    if (this.titleEl) this.titleEl.textContent = this.title;
  };
  PopoverCtrl.prototype.setBody = function (t) {
    this.body = t == null ? "" : String(t);
    if (this.bodyEl) this.bodyEl.textContent = this.body;
  };
  PopoverCtrl.prototype.setContent = function (c) {
    this.setBody(c);
  };
  PopoverCtrl.prototype.setTarget = function (id) {
    this.targetId = id == null ? "" : String(id);
    if (this.open) this.position();
  };
  PopoverCtrl.prototype.applyText = function () {
    if (this.el && this.el.tagName === "BUTTON") this.el.textContent = this.text;
  };
  PopoverCtrl.prototype.remove = function () {
    if (this.open) this.hide();
    if (this.panel && this.panel.parentNode) this.panel.parentNode.removeChild(this.panel);
    this.panel = null;
    Control.prototype.remove.call(this);
  };

  /* ——— דירוג ——— */
  function Rating(opts) {
    Control.call(this, opts);
    this.type = "דירוג";
    this.max = opts.מקס != null ? Number(opts.מקס) || 5 : 5;
    this.value = opts.ערך != null ? Number(opts.ערך) : 0;
  }
  Rating.prototype = Object.create(Control.prototype);
  Rating.prototype.constructor = Rating;
  Rating.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-rating";
    this.render(wrap);
    return wrap;
  };
  Rating.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    var self = this;
    for (var i = 1; i <= this.max; i++) {
      (function (n) {
        var b = document.createElement("button");
        b.type = "button";
        b.className = "btn btn-link p-0 px-1 text-decoration-none";
        b.textContent = n <= self.value ? "★" : "☆";
        b.style.color = n <= self.value ? "#ffc107" : "#6c757d";
        b.addEventListener("click", function () {
          self.setValue(n);
          postChange(self.id, self.type, self.value);
        });
        wrap.appendChild(b);
      })(i);
    }
  };
  Rating.prototype.setValue = function (v) {
    this.value = Number(v) || 0;
    this.render();
  };

  /* ——— כפתור איקון ——— */
  function IconButton(opts) {
    Button.call(this, opts);
    this.type = "כפתור_איקון";
  }
  IconButton.prototype = Object.create(Button.prototype);
  IconButton.prototype.constructor = IconButton;
  IconButton.prototype.createElement = function () {
    var b = document.createElement("button");
    b.type = "button";
    b.className = "btn " + (STYLE_MAP[this.styleName] || STYLE_MAP["ראשי"]);
    b.textContent = this.text || "★";
    b.setAttribute("aria-label", this.text || i18nT("איקון"));
    return b;
  };

  /* ——— קבוצת כפתורים ——— */
  function ButtonGroup(opts) {
    Control.call(this, opts);
    this.type = "קבוצת_כפתורים";
    this.items = normItems(opts.פריטים);
    this.value = "";
  }
  ButtonGroup.prototype = Object.create(Control.prototype);
  ButtonGroup.prototype.constructor = ButtonGroup;
  ButtonGroup.prototype.createElement = function () {
    var g = document.createElement("div");
    g.className = "btn-group";
    g.setAttribute("role", "group");
    this.render(g);
    return g;
  };
  ButtonGroup.prototype.render = function (g) {
    g = g || this.el;
    if (!g) return;
    g.innerHTML = "";
    var self = this;
    for (var i = 0; i < this.items.length; i++) {
      (function (it) {
        var b = document.createElement("button");
        b.type = "button";
        b.className = "btn btn-outline-primary";
        b.textContent = i18nT(it.טקסט);
        b.addEventListener("click", function () {
          var sibs = g.querySelectorAll("button");
          for (var j = 0; j < sibs.length; j++) {
            sibs[j].classList.remove("active");
          }
          b.classList.add("active");
          self.value = it.ערך;
          post({
            סוג: "אירוע",
            שם: "לחיצה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: self.value }
          });
          postChange(self.id, self.type, self.value);
        });
        g.appendChild(b);
      })(this.items[i]);
    }
  };

  /* ——— כרטיס ——— */
  function Card(opts) {
    Control.call(this, opts);
    this.type = "כרטיס";
    this.sourceTitle = opts.כותרת != null ? String(opts.כותרת) : "";
    this.title = i18nT(this.sourceTitle);
    this.body = opts.גוף != null ? String(opts.גוף) : "";
    this.titleEl = null;
    this.bodyEl = null;
    this.contentEl = null;
  }
  Card.prototype = Object.create(Control.prototype);
  Card.prototype.constructor = Card;
  Card.prototype.createElement = function () {
    var c = document.createElement("div");
    c.className = "card yod-section-card";
    c.innerHTML =
      '<div class="card-body">' +
      '<h5 class="card-title"></h5>' +
      '<p class="card-text yod-card-sub"></p>' +
      '<div class="yod-card-content"></div>' +
      "</div>";
    this.titleEl = c.querySelector(".card-title");
    this.bodyEl = c.querySelector(".card-text");
    this.contentEl = c.querySelector(".yod-card-content");
    this.titleEl.textContent = this.title;
    if (this.body) {
      this.bodyEl.textContent = this.body;
    } else {
      this.bodyEl.style.display = "none";
    }
    return c;
  };
  Card.prototype.mountTarget = function () {
    return this.contentEl || (this.el && this.el.querySelector(".yod-card-content")) || this.el;
  };
  Card.prototype.setTitle = function (raw) {
    this.sourceTitle = raw == null ? "" : String(raw);
    this.title = i18nT(this.sourceTitle);
    if (this.titleEl) this.titleEl.textContent = this.title;
  };
  Card.prototype.setBody = function (t) {
    this.body = t == null ? "" : String(t);
    if (this.bodyEl) {
      this.bodyEl.textContent = this.body;
      this.bodyEl.style.display = this.body ? "" : "none";
    }
  };

  /* ——— טבלה ——— */
  function normColumns(cols) {
    var out = [];
    if (!cols) return out;
    for (var i = 0; i < cols.length; i++) {
      var c = cols[i];
      if (typeof c === "string" || typeof c === "number") {
        out.push({ טקסט: String(c), מפתח: String(c) });
      } else if (c && typeof c === "object") {
        var label = c.טקסט != null ? String(c.טקסט) : c.כותרת != null ? String(c.כותרת) : String(c.מפתח || c.key || i);
        var key = c.מפתח != null ? String(c.מפתח) : c.key != null ? String(c.key) : label;
        out.push({ טקסט: label, מפתח: key });
      }
    }
    return out;
  }

  function normRows(rows, columns) {
    var out = [];
    if (!rows) return out;
    for (var i = 0; i < rows.length; i++) {
      var r = rows[i];
      var cells = [];
      var id = String(i);
      if (Array.isArray(r)) {
        for (var j = 0; j < r.length; j++) cells.push(r[j] == null ? "" : String(r[j]));
        id = cells[0] || id;
      } else if (r && typeof r === "object") {
        if (r.מזהה != null) id = String(r.מזהה);
        else if (r.id != null) id = String(r.id);
        if (Array.isArray(r.תאים) || Array.isArray(r.cells)) {
          var arr = r.תאים || r.cells;
          for (var k = 0; k < arr.length; k++) cells.push(arr[k] == null ? "" : String(arr[k]));
        } else {
          for (var c = 0; c < columns.length; c++) {
            var key = columns[c].מפתח;
            var v = r[key];
            if (v == null && r.ערכים) v = r.ערכים[key];
            cells.push(v == null ? "" : String(v));
          }
        }
        if (!id && cells[0]) id = cells[0];
      } else {
        cells.push(String(r));
        id = String(r);
      }
      out.push({ מזהה: id, תאים: cells });
    }
    return out;
  }

  function DataTable(opts) {
    Control.call(this, opts);
    this.type = "טבלה";
    this.columns = normColumns(opts.עמודות);
    this.rows = normRows(opts.שורות, this.columns);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.headEl = null;
    this.bodyEl = null;
  }
  DataTable.prototype = Object.create(Control.prototype);
  DataTable.prototype.constructor = DataTable;
  DataTable.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "table-responsive yod-table";
    var table = document.createElement("table");
    table.className = "table table-hover table-sm align-middle mb-0";
    var thead = document.createElement("thead");
    var tbody = document.createElement("tbody");
    table.appendChild(thead);
    table.appendChild(tbody);
    wrap.appendChild(table);
    this.headEl = thead;
    this.bodyEl = tbody;
    this.render();
    return wrap;
  };
  DataTable.prototype.render = function () {
    if (!this.headEl || !this.bodyEl) return;
    this.headEl.innerHTML = "";
    this.bodyEl.innerHTML = "";
    var trh = document.createElement("tr");
    for (var i = 0; i < this.columns.length; i++) {
      var th = document.createElement("th");
      th.scope = "col";
      th.textContent = i18nT(this.columns[i].טקסט);
      trh.appendChild(th);
    }
    this.headEl.appendChild(trh);
    var self = this;
    for (var r = 0; r < this.rows.length; r++) {
      (function (row) {
        var tr = document.createElement("tr");
        if (self.value && self.value === row.מזהה) tr.classList.add("table-active");
        tr.style.cursor = "pointer";
        for (var c = 0; c < self.columns.length; c++) {
          var td = document.createElement("td");
          td.textContent = row.תאים[c] != null ? row.תאים[c] : "";
          tr.appendChild(td);
        }
        tr.addEventListener("click", function () {
          self.value = row.מזהה;
          self.render();
          post({
            סוג: "אירוע",
            שם: "לחיצה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: row.מזהה }
          });
          postChange(self.id, self.type, row.מזהה);
        });
        self.bodyEl.appendChild(tr);
      })(this.rows[r]);
    }
  };
  DataTable.prototype.setColumns = function (cols) {
    this.columns = normColumns(cols);
    this.rows = normRows(
      this.rows.map(function (r) {
        return r.תאים;
      }),
      this.columns
    );
    this.render();
  };
  DataTable.prototype.setRows = function (rows) {
    this.rows = normRows(rows, this.columns);
    this.render();
  };
  DataTable.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.render();
  };

  /* ——— רשת נתונים (Tabulator) ——— */
  var GRID_ALIGN = {
    ימין: "right",
    שמאל: "left",
    מרכז: "center",
    אמצע: "center",
    right: "right",
    left: "left",
    center: "center"
  };

  // מיפוי ערך סמנטי → מחלקת צבע לפי הנושא
  var GRID_SEMANTIC = {
    הצלחה: "yod-cell-ok",
    הצליח: "yod-cell-ok",
    תקין: "yod-cell-ok",
    ok: "yod-cell-ok",
    success: "yod-cell-ok",
    שגיאה: "yod-cell-danger",
    נכשל: "yod-cell-danger",
    תקלה: "yod-cell-danger",
    error: "yod-cell-danger",
    danger: "yod-cell-danger",
    fail: "yod-cell-danger",
    אזהרה: "yod-cell-warn",
    warn: "yod-cell-warn",
    warning: "yod-cell-warn",
    מידע: "yod-cell-info",
    info: "yod-cell-info"
  };

  function gridEscape(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function gridNumbers(v) {
    if (Array.isArray(v)) {
      return v
        .map(function (n) {
          return Number(n);
        })
        .filter(function (n) {
          return !isNaN(n);
        });
    }
    if (typeof v === "string") {
      return v
        .split(/[,;\s]+/)
        .map(function (n) {
          return Number(n);
        })
        .filter(function (n) {
          return !isNaN(n);
        });
    }
    if (typeof v === "number") return [v];
    return [];
  }

  // ——— גרף זעיר בתא (sparkline) כ-SVG inline ———
  function gridSparkline(v, colOpt) {
    var vals = gridNumbers(v);
    if (!vals.length) return "";
    var type = colOpt.סוג_גרף != null ? String(colOpt.סוג_גרף) : "קו";
    var color = colOpt.צבע != null ? String(colOpt.צבע) : "var(--yod-accent)";
    var w = 82,
      h = 22,
      pad = 2;
    var min, max;
    if (Array.isArray(colOpt.טווח) && colOpt.טווח.length >= 2) {
      min = Number(colOpt.טווח[0]);
      max = Number(colOpt.טווח[1]);
    } else {
      min = Math.min.apply(null, vals);
      max = Math.max.apply(null, vals);
    }
    if (max === min) max = min + 1;
    var n = vals.length;
    function yv(val) {
      return h - pad - ((val - min) / (max - min)) * (h - 2 * pad);
    }
    var svg =
      '<svg class="yod-spark" width="' +
      w +
      '" height="' +
      h +
      '" viewBox="0 0 ' +
      w +
      " " +
      h +
      '" preserveAspectRatio="none">';
    if (type === "עמודות" || type === "bar" || type === "bars") {
      var slot = (w - 2 * pad) / n;
      var bw = slot * 0.7;
      for (var i = 0; i < n; i++) {
        var bx = pad + i * slot + (slot - bw) / 2;
        var by = yv(vals[i]);
        var bh = Math.max(0.5, h - pad - by);
        svg +=
          '<rect x="' +
          bx.toFixed(1) +
          '" y="' +
          by.toFixed(1) +
          '" width="' +
          bw.toFixed(1) +
          '" height="' +
          bh.toFixed(1) +
          '" fill="' +
          color +
          '" rx="0.5"/>';
      }
    } else {
      var xstep = n > 1 ? (w - 2 * pad) / (n - 1) : 0;
      var pts = [];
      for (var j = 0; j < n; j++) {
        pts.push((pad + j * xstep).toFixed(1) + "," + yv(vals[j]).toFixed(1));
      }
      if (type === "שטח" || type === "area") {
        var area =
          "M" +
          pad +
          "," +
          (h - pad) +
          " L" +
          pts.join(" L") +
          " L" +
          (pad + (n - 1) * xstep).toFixed(1) +
          "," +
          (h - pad) +
          " Z";
        svg += '<path d="' + area + '" fill="' + color + '" opacity="0.22"/>';
      }
      svg +=
        '<polyline points="' +
        pts.join(" ") +
        '" fill="none" stroke="' +
        color +
        '" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/>';
    }
    svg += "</svg>";
    return (
      '<span class="yod-spark-wrap" title="' + gridEscape(vals.join(", ")) + '">' + svg + "</span>"
    );
  }

  function gridBadge(v, colOpt) {
    var map = colOpt.צבעים || {};
    var sval = v == null ? "" : String(v);
    var cls = "";
    var style = "";
    var explicit = map[sval] != null ? String(map[sval]) : null;
    var key = explicit != null ? explicit : sval;
    if (GRID_SEMANTIC[key]) {
      cls = " " + GRID_SEMANTIC[key];
    } else if (explicit != null) {
      style = ' style="background:' + gridEscape(explicit) + ';color:#fff"';
    }
    return '<span class="yod-grid-badge' + cls + '"' + style + ">" + gridEscape(i18nT(sval)) + "</span>";
  }

  function gridProgress(v, colOpt) {
    if (v == null || v === "") return "";
    var min = colOpt.מינימום != null ? Number(colOpt.מינימום) : 0;
    var max = colOpt.מקסימום != null ? Number(colOpt.מקסימום) : 100;
    var num = Number(v);
    if (isNaN(num)) num = 0;
    var pct = max === min ? 0 : ((num - min) / (max - min)) * 100;
    pct = Math.max(0, Math.min(100, pct));
    var color = colOpt.צבע != null ? String(colOpt.צבע) : "var(--yod-accent)";
    return (
      '<div class="yod-grid-progress"><div class="yod-grid-progress-bar" style="width:' +
      pct.toFixed(1) +
      "%;background:" +
      color +
      '"></div><span class="yod-grid-progress-label">' +
      gridEscape(num) +
      "</span></div>"
    );
  }

  function gridStars(v, colOpt) {
    var maxs = colOpt.מקסימום != null ? Number(colOpt.מקסימום) : 5;
    var n = Math.round(Number(v));
    if (isNaN(n)) n = 0;
    var s = "";
    for (var i = 1; i <= maxs; i++) {
      s += '<span class="yod-star ' + (i <= n ? "on" : "off") + '">★</span>';
    }
    return '<span class="yod-grid-stars">' + s + "</span>";
  }

  function gridTick(v) {
    var on = v === true || v === 1 || v === "1" || v === "אמת" || v === "true" || v === "כן";
    return (
      '<span class="yod-grid-tick ' + (on ? "on" : "off") + '">' + (on ? "✓" : "✗") + "</span>"
    );
  }

  function gridLink(v, colOpt) {
    var url = v == null ? "" : String(v);
    var label = colOpt.תווית != null ? String(colOpt.תווית) : url;
    return (
      '<a href="' +
      gridEscape(url) +
      '" target="_blank" rel="noopener" class="yod-grid-link">' +
      gridEscape(i18nT(label)) +
      "</a>"
    );
  }

  function gridImage(v, colOpt) {
    var h = colOpt.גובה != null ? String(colOpt.גובה) : "24px";
    return (
      '<img class="yod-grid-img" src="' +
      gridEscape(v == null ? "" : v) +
      '" style="height:' +
      gridEscape(h) +
      '" alt=""/>'
    );
  }

  function gridIsTruthy(v) {
    return v === true || v === 1 || v === "1" || v === "אמת" || v === "true" || v === "כן";
  }

  // תיבת סימון אמיתית (לחיצה מחליפה מצב דרך אירוע "עריכה")
  function gridCheckbox(v) {
    var on = gridIsTruthy(v);
    return (
      '<span class="yod-grid-check' +
      (on ? " on" : "") +
      '" role="checkbox" aria-checked="' +
      (on ? "true" : "false") +
      '" tabindex="0"></span>'
    );
  }

  // עמודת כפתורי פעולה (למשל מחיקה) — לחיצה שולחת אירוע "תפריט"
  function gridActions(colOpt) {
    var btns = Array.isArray(colOpt.כפתורים) ? colOpt.כפתורים : [{ פעולה: "מחק", איקון: "trash" }];
    var html = '<div class="yod-grid-actions">';
    for (var i = 0; i < btns.length; i++) {
      var b = btns[i] || {};
      var act = b.פעולה != null ? String(b.פעולה) : b.תווית != null ? String(b.תווית) : "";
      var icoKey = b.איקון != null ? String(b.איקון) : "";
      var inner =
        icoKey && ICO[icoKey]
          ? svgIcon(icoKey, "")
          : gridEscape(b.תווית != null ? String(b.תווית) : act);
      var title = b.רמז != null ? String(b.רמז) : i18nT(act);
      var cls = b.סגנון === "סכנה" || b.סגנון === "danger" ? " yod-grid-action-danger" : "";
      html +=
        '<button type="button" class="yod-grid-action' +
        cls +
        '" data-yod-action="' +
        gridEscape(act) +
        '" title="' +
        gridEscape(title) +
        '" aria-label="' +
        gridEscape(title) +
        '">' +
        inner +
        "</button>";
    }
    html += "</div>";
    return html;
  }

  function gridEvalRule(rule, value) {
    var op = rule.מתי != null ? String(rule.מתי) : "שווה";
    var target = rule.ערך;
    var num = Number(value);
    var tnum = Number(target);
    switch (op) {
      case "שווה":
      case "==":
        return String(value) === String(target);
      case "שונה":
      case "!=":
        return String(value) !== String(target);
      case "גדול":
      case ">":
        return !isNaN(num) && num > tnum;
      case "גדול_שווה":
      case ">=":
        return !isNaN(num) && num >= tnum;
      case "קטן":
      case "<":
        return !isNaN(num) && num < tnum;
      case "קטן_שווה":
      case "<=":
        return !isNaN(num) && num <= tnum;
      case "בין":
        return !isNaN(num) && num >= Number(rule.ערך) && num <= Number(rule.ערך2);
      case "מכיל":
        return String(value).indexOf(String(target)) >= 0;
      case "ריק":
        return value == null || String(value) === "";
      default:
        return false;
    }
  }

  function gridApplyConditional(el, rules, value) {
    if (!el || !rules) return;
    for (var i = 0; i < rules.length; i++) {
      var rule = rules[i];
      if (gridEvalRule(rule, value)) {
        if (rule.מחלקה) el.classList.add(String(rule.מחלקה));
        if (rule.רקע) el.style.backgroundColor = String(rule.רקע);
        if (rule.טקסט) el.style.color = String(rule.טקסט);
        break;
      }
    }
  }

  // בונה formatter מותאם לעמודה (מעצב/גרף/עיצוב מותנה); מחזיר null אם לא נדרש
  function gridFormatter(colOpt) {
    var kind = colOpt.מעצב != null ? String(colOpt.מעצב) : colOpt.תבנית != null ? String(colOpt.תבנית) : "";
    var cond = Array.isArray(colOpt.צבע_מותנה) ? colOpt.צבע_מותנה : null;
    if (!kind && !cond) return null;
    return function (cell, params, onRendered) {
      var v = cell.getValue();
      var html;
      switch (kind) {
        case "פס":
        case "progress":
          html = gridProgress(v, colOpt);
          break;
        case "כוכבים":
        case "star":
          html = gridStars(v, colOpt);
          break;
        case "צ׳קבוקס":
        case "צ'קבוקס":
        case "checkbox":
        case "tick":
          html = gridTick(v);
          break;
        case "סימון":
        case "checkbox_input":
          html = gridCheckbox(v);
          break;
        case "פעולות":
        case "actions":
          html = gridActions(colOpt);
          break;
        case "קישור":
        case "link":
          html = gridLink(v, colOpt);
          break;
        case "תמונה":
        case "image":
          html = gridImage(v, colOpt);
          break;
        case "מצב":
        case "תג":
        case "badge":
          html = gridBadge(v, colOpt);
          break;
        case "גרף":
        case "spark":
        case "sparkline":
          html = gridSparkline(v, colOpt);
          break;
        default:
          html = gridEscape(v == null ? "" : String(v));
      }
      if (cond) {
        onRendered(function () {
          gridApplyConditional(cell.getElement(), cond, v);
        });
      }
      return html;
    };
  }

  function gridDirClass(colOpt, sug) {
    var d = colOpt.כיוון != null ? String(colOpt.כיוון) : "";
    if (d === "שמאל" || d === "ltr" || d === "שמאל_לימין") return "yod-cell-ltr";
    if (d === "ימין" || d === "rtl" || d === "ימין_לשמאל") return "yod-cell-rtl";
    // אוטומטי / לא צוין — מספרים/תאריך/מטבע → LTR כדי שספרות יוצגו נכון ב-RTL
    if (sug === "מספר" || sug === "number" || sug === "תאריך" || sug === "date" || sug === "מטבע") {
      return "yod-cell-ltr";
    }
    return "";
  }

  function gridSorter(sug) {
    if (sug === "מספר" || sug === "number" || sug === "מטבע") return "number";
    if (sug === "תאריך" || sug === "date") return "alphanum";
    if (sug === "בוליאני" || sug === "boolean") return "boolean";
    return "string";
  }

  function gridEditor(colOpt, sug) {
    var e = colOpt.עריכה;
    if (!e) return null;
    var kind = e === true ? sug || "טקסט" : String(e);
    switch (kind) {
      case "מספר":
      case "number":
      case "מטבע":
        return { editor: "number" };
      case "טקסט_ארוך":
      case "textarea":
        return { editor: "textarea" };
      case "בחירה":
      case "list":
        return { editor: "list", editorParams: { values: colOpt.אפשרויות || [] } };
      case "תאריך":
      case "date":
        return { editor: "date" };
      case "בוליאני":
      case "checkbox":
      case "tick":
        return { editor: "tickCross" };
      default:
        return { editor: "input" };
    }
  }

  function gridHeaderFilter(colOpt, sug) {
    if (!colOpt.סינון) return null;
    if (sug === "מספר" || sug === "number" || sug === "מטבע") return { headerFilter: "number" };
    if (sug === "בחירה" || sug === "list") {
      return { headerFilter: "list", headerFilterParams: { values: colOpt.אפשרויות || [], clearable: true } };
    }
    return { headerFilter: "input" };
  }

  var GRID_CALC = {
    סכום: "sum",
    ממוצע: "avg",
    מונה: "count",
    מקסימום: "max",
    מינימום: "min",
    sum: "sum",
    avg: "avg",
    count: "count",
    max: "max",
    min: "min"
  };

  function gridColumns(cols) {
    var out = [];
    if (!cols) return out;
    for (var i = 0; i < cols.length; i++) {
      var c = cols[i];
      var def;
      if (typeof c === "string" || typeof c === "number") {
        var s = String(c);
        def = { title: i18nT(s), field: s, headerSort: true };
        out.push(def);
        continue;
      }
      if (!c || typeof c !== "object") continue;
      var label =
        c.טקסט != null
          ? String(c.טקסט)
          : c.כותרת != null
          ? String(c.כותרת)
          : String(c.מפתח || c.key || i);
      var key = c.מפתח != null ? String(c.מפתח) : c.key != null ? String(c.key) : label;
      var sug = c.סוג != null ? String(c.סוג) : "";
      def = { title: i18nT(label), field: key };
      def.headerSort = c.מיון === false ? false : true;
      def.sorter = gridSorter(sug);
      var w = c.רוחב != null ? c.רוחב : c.width;
      if (w != null) def.width = w;
      var al = c.יישור != null ? String(c.יישור) : c.align != null ? String(c.align) : "";
      if (al) def.hozAlign = GRID_ALIGN[al] || al;
      if (c.הקפא === true || c.frozen === true) def.frozen = true;
      var fmt = gridFormatter(c);
      if (fmt) def.formatter = fmt;
      var ed = gridEditor(c, sug);
      if (ed) {
        def.editor = ed.editor;
        if (ed.editorParams) def.editorParams = ed.editorParams;
      }
      var hf = gridHeaderFilter(c, sug);
      if (hf) {
        def.headerFilter = hf.headerFilter;
        if (hf.headerFilterParams) def.headerFilterParams = hf.headerFilterParams;
      }
      if (c.חישוב != null && GRID_CALC[String(c.חישוב)]) def.bottomCalc = GRID_CALC[String(c.חישוב)];
      var dirCls = gridDirClass(c, sug);
      if (dirCls) def.cssClass = (def.cssClass ? def.cssClass + " " : "") + dirCls;
      out.push(def);
    }
    return out;
  }

  function gridSelectionCol() {
    return {
      formatter: "rowSelection",
      titleFormatter: "rowSelection",
      hozAlign: "center",
      headerHozAlign: "center",
      headerSort: false,
      width: 42,
      frozen: true,
      cssClass: "yod-grid-select-col"
    };
  }

  function gridRows(rows, colDefs) {
    var out = [];
    if (!rows) return out;
    var fields = [];
    for (var f = 0; f < colDefs.length; f++) {
      if (colDefs[f].field) fields.push(colDefs[f].field);
    }
    for (var i = 0; i < rows.length; i++) {
      var r = rows[i];
      var obj = {};
      var id = String(i);
      if (Array.isArray(r)) {
        for (var j = 0; j < r.length; j++) {
          if (j < fields.length) obj[fields[j]] = r[j] == null ? "" : r[j];
        }
        if (r[0] != null && String(r[0]) !== "") id = String(r[0]);
      } else if (r && typeof r === "object") {
        if (r.מזהה != null) id = String(r.מזהה);
        else if (r.id != null) id = String(r.id);
        var arr = Array.isArray(r.תאים) ? r.תאים : Array.isArray(r.cells) ? r.cells : null;
        if (arr) {
          for (var k = 0; k < arr.length; k++) {
            if (k < fields.length) obj[fields[k]] = arr[k] == null ? "" : arr[k];
          }
        } else {
          for (var c = 0; c < fields.length; c++) {
            var fkey = fields[c];
            var v = r[fkey];
            if (v == null && r.ערכים) v = r.ערכים[fkey];
            obj[fkey] = v == null ? "" : v;
          }
        }
        // שדות מטא לעיצוב שורה
        if (r._רקע != null) obj._רקע = r._רקע;
        if (r._טקסט != null) obj._טקסט = r._טקסט;
        if (r._מחלקה != null) obj._מחלקה = r._מחלקה;
        // ילדים (עץ) — נורמליזציה רקורסיבית
        if (r.ילדים != null && Array.isArray(r.ילדים)) obj.ילדים = gridRows(r.ילדים, colDefs);
      } else {
        if (fields.length) obj[fields[0]] = r == null ? "" : r;
        id = String(r);
      }
      obj.__id = id;
      out.push(obj);
    }
    return out;
  }

  function DataGrid(opts) {
    Control.call(this, opts);
    this.type = "רשת_נתונים";
    this.colSource = opts.עמודות || [];
    this.colDefs = gridColumns(this.colSource);
    this.rowData = gridRows(opts.שורות, this.colDefs);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.height = opts.גובה != null && opts.גובה !== "" ? opts.גובה : "";
    this.multiSelect = opts.בחירה_מרובה === true;
    this.paginate = opts.עימוד === true;
    this.pageSize = opts.גודל_עמוד != null ? Number(opts.גודל_עמוד) : 10;
    this.groupField = opts.קבץ_לפי != null && opts.קבץ_לפי !== "" ? String(opts.קבץ_לפי) : "";
    this.movableCols = opts.הזזת_עמודות === true;
    this.responsive = opts.רספונסיבי === true;
    this.tree = opts.עץ === true;
    this.contextItems = Array.isArray(opts.תפריט_שורה) ? opts.תפריט_שורה : null;
    this.rowColorRules = Array.isArray(opts.צבע_שורה) ? opts.צבע_שורה : null;
    this.gridTheme = opts.נושא != null ? String(opts.נושא) : "";
    this.flat = opts.שטוח === true;
    this.table = null;
    this.built = false;
  }
  DataGrid.prototype = Object.create(Control.prototype);
  DataGrid.prototype.constructor = DataGrid;
  DataGrid.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-datagrid" + (this.flat ? " yod-datagrid-flat" : "");
    return wrap;
  };
  DataGrid.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    this.applyGridTheme();
    this.buildTable();
    return this;
  };
  DataGrid.prototype.currentDir = function () {
    if (typeof i18n !== "undefined" && i18n && i18n.dir) return i18n.dir === "ltr" ? "ltr" : "rtl";
    var d = document.documentElement.getAttribute("dir");
    return d === "ltr" ? "ltr" : "rtl";
  };
  DataGrid.prototype.finalizeCols = function () {
    var defs = this.colDefs.slice();
    if (this.multiSelect) defs.unshift(gridSelectionCol());
    return defs;
  };
  DataGrid.prototype.buildContextMenu = function () {
    var self = this;
    if (!this.contextItems) return undefined;
    return this.contextItems.map(function (it) {
      return {
        label: gridEscape(i18nT(it.תווית != null ? String(it.תווית) : String(it.פעולה || ""))),
        action: function (e, row) {
          var d = row.getData();
          post({
            סוג: "אירוע",
            שם: "תפריט",
            ערכים: {
              מזהה: self.id,
              סוג: self.type,
              פעולה: it.פעולה != null ? it.פעולה : it.תווית,
              מזהה_שורה: d && d.__id != null ? String(d.__id) : ""
            }
          });
        }
      };
    });
  };
  DataGrid.prototype.rowFormatter = function (row) {
    var d = row.getData();
    var el = row.getElement();
    if (!el) return;
    if (d._רקע) el.style.backgroundColor = String(d._רקע);
    if (d._טקסט) el.style.color = String(d._טקסט);
    if (d._מחלקה) el.classList.add(String(d._מחלקה));
    var rules = this.rowColorRules;
    if (rules) {
      for (var i = 0; i < rules.length; i++) {
        var rule = rules[i];
        var field = rule.שדה != null ? String(rule.שדה) : "";
        if (gridEvalRule(rule, field ? d[field] : "")) {
          if (rule.מחלקה) el.classList.add(String(rule.מחלקה));
          if (rule.רקע) el.style.backgroundColor = String(rule.רקע);
          if (rule.טקסט) el.style.color = String(rule.טקסט);
          break;
        }
      }
    }
  };
  DataGrid.prototype.buildTable = function () {
    if (this.table || !this.el || typeof Tabulator === "undefined") return;
    var self = this;
    var options = {
      index: "__id",
      data: this.rowData,
      columns: this.finalizeCols(),
      layout: "fitColumns",
      textDirection: this.currentDir(),
      placeholder: i18nT("אין נתונים"),
      clipboard: true,
      clipboardCopyRowRange: this.multiSelect ? "selected" : "active",
      selectableRows: this.multiSelect ? true : 1,
      rowFormatter: function (row) {
        self.rowFormatter(row);
      }
    };
    if (this.height) options.height = this.height;
    if (this.paginate) {
      options.pagination = "local";
      options.paginationSize = this.pageSize > 0 ? this.pageSize : 10;
      options.paginationCounter = "rows";
    }
    if (this.groupField) options.groupBy = this.groupField;
    if (this.movableCols) options.movableColumns = true;
    if (this.responsive) options.responsiveLayout = "collapse";
    if (this.tree) {
      options.dataTree = true;
      options.dataTreeChildField = "ילדים";
      options.dataTreeStartExpanded = true;
    }
    var menu = this.buildContextMenu();
    if (menu) options.rowContextMenu = menu;
    try {
      this.table = new Tabulator(this.el, options);
    } catch (e) {
      console.warn("אתחול רשת_נתונים נכשל:", e);
      return;
    }
    this.table.on("tableBuilt", function () {
      self.built = true;
      try {
        self.table.setColumns(self.finalizeCols());
        self.table.replaceData(self.rowData);
      } catch (e) {}
      if (self.value) self.applySelection();
    });
    this.table.on("rowClick", function (e, row) {
      var d = row.getData();
      var rid = d && d.__id != null ? String(d.__id) : "";
      self.value = rid;
      post({
        סוג: "אירוע",
        שם: "לחיצה",
        ערכים: { מזהה: self.id, סוג: self.type, ערך: rid }
      });
      postChange(self.id, self.type, rid);
    });
    this.table.on("cellEdited", function (cell) {
      var d = cell.getRow().getData();
      var rid = d && d.__id != null ? String(d.__id) : "";
      if (d && d.hasOwnProperty(cell.getField())) {
        // עדכון המטמון המקומי
        for (var m = 0; m < self.rowData.length; m++) {
          if (String(self.rowData[m].__id) === rid) {
            self.rowData[m][cell.getField()] = cell.getValue();
            break;
          }
        }
      }
      post({
        סוג: "אירוע",
        שם: "עריכה",
        ערכים: {
          מזהה: self.id,
          סוג: self.type,
          שדה: cell.getField(),
          ערך: cell.getValue(),
          מזהה_שורה: rid
        }
      });
    });
    this.table.on("cellClick", function (e, cell) {
      var t = e && e.target;
      if (!t || !t.closest) return;
      var d = cell.getRow().getData();
      var rid = d && d.__id != null ? String(d.__id) : "";
      var actBtn = t.closest("[data-yod-action]");
      if (actBtn) {
        if (e.stopPropagation) e.stopPropagation();
        post({
          סוג: "אירוע",
          שם: "תפריט",
          ערכים: {
            מזהה: self.id,
            סוג: self.type,
            פעולה: actBtn.getAttribute("data-yod-action"),
            מזהה_שורה: rid
          }
        });
        return;
      }
      var chk = t.closest(".yod-grid-check");
      if (chk) {
        if (e.stopPropagation) e.stopPropagation();
        var cur = cell.getValue();
        var on = cur === true || cur === 1 || cur === "1" || cur === "אמת" || cur === "true" || cur === "כן";
        post({
          סוג: "אירוע",
          שם: "עריכה",
          ערכים: {
            מזהה: self.id,
            סוג: self.type,
            שדה: cell.getField(),
            ערך: !on,
            מזהה_שורה: rid
          }
        });
        return;
      }
    });
    if (this.multiSelect) {
      this.table.on("rowSelectionChanged", function (data) {
        var ids = data.map(function (d) {
          return d && d.__id != null ? String(d.__id) : "";
        });
        post({
          סוג: "אירוע",
          שם: "בחירה",
          ערכים: { מזהה: self.id, סוג: self.type, מזהים: ids }
        });
      });
    }
  };
  DataGrid.prototype.applyGridTheme = function () {
    if (!this.el) return;
    var t = this.gridTheme;
    if (t === "בהיר" || t === "light") this.el.setAttribute("data-yod-grid-theme", "light");
    else if (t === "כהה" || t === "dark") this.el.setAttribute("data-yod-grid-theme", "dark");
    else this.el.removeAttribute("data-yod-grid-theme");
  };
  DataGrid.prototype.setGridTheme = function (t) {
    this.gridTheme = t == null ? "" : String(t);
    this.applyGridTheme();
    if (this.table && this.built) {
      try {
        this.table.redraw(true);
      } catch (e) {}
    }
  };
  DataGrid.prototype.applySelection = function () {
    if (!this.table || !this.built) return;
    try {
      this.table.deselectRow();
      if (this.value) this.table.selectRow(this.value);
    } catch (e) {}
  };
  DataGrid.prototype.setColumns = function (cols) {
    this.colSource = cols || [];
    this.colDefs = gridColumns(this.colSource);
    if (this.table && this.built) {
      try {
        this.table.setColumns(this.finalizeCols());
      } catch (e) {}
    }
  };
  DataGrid.prototype.setRows = function (rows) {
    this.rowData = gridRows(rows, this.colDefs);
    if (this.table && this.built) {
      try {
        this.table.replaceData(this.rowData);
      } catch (e) {}
    }
  };
  DataGrid.prototype.addRow = function (row) {
    var one = gridRows([row], this.colDefs);
    if (!one.length) return;
    this.rowData.push(one[0]);
    if (this.table && this.built) {
      try {
        this.table.addData(one);
      } catch (e) {}
    }
  };
  DataGrid.prototype.clear = function () {
    this.rowData = [];
    if (this.table && this.built) {
      try {
        this.table.replaceData([]);
      } catch (e) {}
    }
  };
  DataGrid.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.applySelection();
  };
  DataGrid.prototype.setHeight = function (h) {
    this.height = h == null || h === "" ? "" : h;
    if (this.table && this.built) {
      try {
        this.table.setHeight(this.height || false);
      } catch (e) {}
    }
  };
  DataGrid.prototype.setFilter = function (field, value) {
    if (!this.table || !this.built) return;
    try {
      if (value == null || value === "") {
        this.table.clearFilter(true);
      } else {
        this.table.setFilter(String(field), "like", value);
      }
    } catch (e) {}
  };
  DataGrid.prototype.setFilterGlobal = function (text) {
    if (!this.table || !this.built) return;
    var t = String(text == null ? "" : text).toLowerCase();
    try {
      if (!t) {
        this.table.clearFilter(true);
        return;
      }
      this.table.setFilter(function (data) {
        for (var k in data) {
          if (k === "__id" || k === "_רקע" || k === "_טקסט" || k === "_מחלקה") continue;
          if (String(data[k]).toLowerCase().indexOf(t) >= 0) return true;
        }
        return false;
      });
    } catch (e) {}
  };
  DataGrid.prototype.clearFilter = function () {
    if (this.table && this.built) {
      try {
        this.table.clearFilter(true);
        this.table.clearHeaderFilter();
      } catch (e) {}
    }
  };
  DataGrid.prototype.groupBy = function (field) {
    this.groupField = field == null ? "" : String(field);
    if (this.table && this.built) {
      try {
        this.table.setGroupBy(this.groupField || false);
      } catch (e) {}
    }
  };
  DataGrid.prototype.copyGrid = function () {
    if (!this.table || !this.built) return;
    try {
      this.table.copyToClipboard(this.multiSelect ? "selected" : "active");
    } catch (e) {}
  };
  DataGrid.prototype.updateCell = function (rowId, field, value) {
    var rid = String(rowId);
    for (var m = 0; m < this.rowData.length; m++) {
      if (String(this.rowData[m].__id) === rid) {
        this.rowData[m][field] = value;
        break;
      }
    }
    if (this.table && this.built) {
      try {
        var row = this.table.getRow(rid);
        if (row) {
          var upd = { __id: rid };
          upd[field] = value;
          row.update(upd);
        }
      } catch (e) {}
    }
  };
  DataGrid.prototype.exportGrid = function (format, path) {
    var self = this;
    format = String(format || "csv").toLowerCase();
    var rows = this.table && this.built ? this.table.getData("active") : this.rowData;
    var cols = this.colDefs.filter(function (c) {
      return c.field && c.field !== "__id";
    });
    if (format === "json") {
      var arr = rows.map(function (r) {
        var o = {};
        cols.forEach(function (c) {
          o[c.field] = r[c.field];
        });
        return o;
      });
      post({
        סוג: "אירוע",
        שם: "ייצוא",
        ערכים: { מזהה: self.id, סוג: self.type, פורמט: "json", תוכן: JSON.stringify(arr, null, 2), נתיב: path || "" }
      });
    } else if (format === "xlsx") {
      var cells = [];
      cols.forEach(function (c, ci) {
        cells.push({ תא: gridColLetter(ci) + "1", ערך: c.title });
      });
      rows.forEach(function (r, ri) {
        cols.forEach(function (c, ci) {
          var val = r[c.field];
          cells.push({ תא: gridColLetter(ci) + (ri + 2), ערך: val == null ? "" : val });
        });
      });
      post({
        סוג: "אירוע",
        שם: "ייצוא",
        ערכים: { מזהה: self.id, סוג: self.type, פורמט: "xlsx", תאים: cells, נתיב: path || "" }
      });
    } else {
      var lines = [];
      lines.push(
        cols
          .map(function (c) {
            return gridCsvCell(c.title);
          })
          .join(",")
      );
      rows.forEach(function (r) {
        lines.push(
          cols
            .map(function (c) {
              return gridCsvCell(r[c.field]);
            })
            .join(",")
        );
      });
      post({
        סוג: "אירוע",
        שם: "ייצוא",
        ערכים: { מזהה: self.id, סוג: self.type, פורמט: "csv", תוכן: "\ufeff" + lines.join("\r\n"), נתיב: path || "" }
      });
    }
  };
  DataGrid.prototype.applyI18n = function () {
    Control.prototype.applyI18n.call(this);
    this.colDefs = gridColumns(this.colSource);
    if (this.table && this.built) {
      try {
        this.table.setColumns(this.finalizeCols());
        this.table.replaceData(this.rowData);
      } catch (e) {}
    }
  };
  DataGrid.prototype.remove = function () {
    try {
      if (this.table) this.table.destroy();
    } catch (e) {}
    this.table = null;
    this.built = false;
    Control.prototype.remove.call(this);
  };

  function gridCsvCell(v) {
    var s = v == null ? "" : String(v);
    if (/[",\r\n]/.test(s)) {
      s = '"' + s.replace(/"/g, '""') + '"';
    }
    return s;
  }

  function gridColLetter(idx) {
    var s = "";
    idx++;
    while (idx > 0) {
      var r = (idx - 1) % 26;
      s = String.fromCharCode(65 + r) + s;
      idx = Math.floor((idx - 1) / 26);
    }
    return s;
  }

  /* ——— תמונת פרופיל (Avatar) ——— */
  var AVATAR_SIZE = {
    קטן: "yod-avatar-sm",
    בינוני: "yod-avatar-md",
    גדול: "yod-avatar-lg",
    sm: "yod-avatar-sm",
    md: "yod-avatar-md",
    lg: "yod-avatar-lg"
  };

  function avatarInitials(name) {
    name = (name || "").toString().trim();
    if (!name) return "?";
    var parts = name.split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0].charAt(0) + parts[1].charAt(0)).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }

  function Avatar(opts) {
    Control.call(this, opts);
    this.type = "תמונת_פרופיל";
    this.name = opts.שם != null ? String(opts.שם) : this.text || "";
    this.image = opts.תמונה != null ? String(opts.תמונה) : "";
    this.sizeName = opts.גודל || "בינוני";
    this.imgEl = null;
    this.initialsEl = null;
  }
  Avatar.prototype = Object.create(Control.prototype);
  Avatar.prototype.constructor = Avatar;
  Avatar.prototype.createElement = function () {
    var wrap = document.createElement("button");
    wrap.type = "button";
    wrap.className =
      "yod-avatar " + (AVATAR_SIZE[this.sizeName] || AVATAR_SIZE["בינוני"]);
    wrap.setAttribute("aria-label", this.name || "פרופיל");
    var img = document.createElement("img");
    img.alt = this.name || "";
    img.className = "yod-avatar-img";
    var ini = document.createElement("span");
    ini.className = "yod-avatar-initials";
    ini.textContent = avatarInitials(this.name);
    wrap.appendChild(img);
    wrap.appendChild(ini);
    this.imgEl = img;
    this.initialsEl = ini;
    this.applyImage();
    return wrap;
  };
  Avatar.prototype.applyImage = function () {
    if (!this.imgEl || !this.initialsEl) return;
    if (this.image) {
      this.imgEl.src = this.image;
      this.imgEl.style.display = "block";
      this.initialsEl.style.display = "none";
      var self = this;
      this.imgEl.onerror = function () {
        self.imgEl.style.display = "none";
        self.initialsEl.style.display = "flex";
      };
    } else {
      this.imgEl.removeAttribute("src");
      this.imgEl.style.display = "none";
      this.initialsEl.style.display = "flex";
      this.initialsEl.textContent = avatarInitials(this.name);
    }
  };
  Avatar.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("click", function () {
      post({
        סוג: "אירוע",
        שם: "לחיצה",
        ערכים: { מזהה: self.id, סוג: self.type, ערך: self.name || "" }
      });
    });
  };
  Avatar.prototype.applyText = function () {
    this.name = this.text;
    if (this.el) this.el.setAttribute("aria-label", this.name || "פרופיל");
    if (this.initialsEl) this.initialsEl.textContent = avatarInitials(this.name);
  };
  Avatar.prototype.setName = function (n) {
    this.name = n == null ? "" : String(n);
    this.text = this.name;
    if (this.el) this.el.setAttribute("aria-label", this.name || "פרופיל");
    if (this.initialsEl) this.initialsEl.textContent = avatarInitials(this.name);
  };
  Avatar.prototype.setImage = function (src) {
    this.image = src == null ? "" : String(src);
    this.applyImage();
  };
  Avatar.prototype.setSize = function (s) {
    this.sizeName = s || "בינוני";
    if (!this.el) return;
    this.el.className =
      "yod-avatar yod-control " + (AVATAR_SIZE[this.sizeName] || AVATAR_SIZE["בינוני"]);
  };

  /* ——— רשימה ——— */
  function ListCtrl(opts) {
    Control.call(this, opts);
    this.type = "רשימה";
    this.items = normItems(opts.פריטים);
    this.value = "";
  }
  ListCtrl.prototype = Object.create(Control.prototype);
  ListCtrl.prototype.constructor = ListCtrl;
  ListCtrl.prototype.itemsTextJoined = function (oldestFirst) {
    var lines = [];
    var i;
    if (oldestFirst) {
      for (i = this.items.length - 1; i >= 0; i--) {
        if (this.items[i] && this.items[i].טקסט) lines.push(String(this.items[i].טקסט));
      }
    } else {
      for (i = 0; i < this.items.length; i++) {
        if (this.items[i] && this.items[i].טקסט) lines.push(String(this.items[i].טקסט));
      }
    }
    return lines.join("\n");
  };
  ListCtrl.prototype.openCopyMenu = function (ev, item) {
    var self = this;
    ev.preventDefault();
    ev.stopPropagation();
    var hasLine = !!(item && item.טקסט);
    var hasAny = this.items && this.items.length > 0;
    showYodContextMenu(ev.clientX, ev.clientY, [
      {
        label: "העתק",
        disabled: !hasLine,
        onClick: function () {
          var t = item && item.טקסט != null ? String(item.טקסט) : "";
          copyTextToClipboard(t);
          post({
            סוג: "אירוע",
            שם: "העתקה",
            ערכים: {
              מזהה: self.id,
              סוג: self.type,
              פעולה: "שורה",
              ערך: item && item.ערך != null ? String(item.ערך) : "",
              טקסט: t
            }
          });
        }
      },
      {
        label: "העתק הכל",
        disabled: !hasAny,
        onClick: function () {
          var t = self.itemsTextJoined(true);
          copyTextToClipboard(t);
          post({
            סוג: "אירוע",
            שם: "העתקה",
            ערכים: {
              מזהה: self.id,
              סוג: self.type,
              פעולה: "הכל",
              ערך: "",
              טקסט: t
            }
          });
        }
      }
    ]);
  };
  ListCtrl.prototype.createElement = function () {
    var ul = document.createElement("div");
    ul.className = "list-group";
    ul.setAttribute("data-yod-list", this.id || "");
    var self = this;
    ul.addEventListener("contextmenu", function (ev) {
      if (ev.target && ev.target.closest && ev.target.closest(".list-group-item")) return;
      self.openCopyMenu(ev, null);
    });
    this.render(ul);
    return ul;
  };
  ListCtrl.prototype.render = function (ul) {
    ul = ul || this.el;
    if (!ul) return;
    ul.innerHTML = "";
    var self = this;
    var sel = self.value != null ? String(self.value) : "";
    for (var i = 0; i < this.items.length; i++) {
      (function (it) {
        var a = document.createElement("button");
        a.type = "button";
        a.className = "list-group-item list-group-item-action";
        a.setAttribute("data-value", String(it.ערך));
        var kind = it.סוג || logKindFromText(it.טקסט);
        if (kind) {
          a.classList.add("yod-log-item");
          a.setAttribute("data-log-kind", kind);
        }
        if (it.התקדמות != null) {
          a.classList.add("yod-log-transfer");
          var row = document.createElement("div");
          row.className = "yod-log-transfer-row";
          var txt = document.createElement("span");
          txt.className = "yod-log-transfer-text";
          txt.textContent = i18nT(it.טקסט);
          var progWrap = document.createElement("div");
          progWrap.className = "progress yod-log-mini-progress";
          progWrap.setAttribute("role", "progressbar");
          progWrap.setAttribute("aria-valuenow", String(Math.round(it.התקדמות)));
          progWrap.setAttribute("aria-valuemin", "0");
          progWrap.setAttribute("aria-valuemax", "100");
          var bar = document.createElement("div");
          bar.className = "progress-bar";
          if (it.התקדמות >= 100) bar.classList.add("bg-success");
          bar.style.width = Math.max(0, Math.min(100, it.התקדמות)) + "%";
          progWrap.appendChild(bar);
          var pct = document.createElement("span");
          pct.className = "yod-log-mini-pct";
          pct.textContent = Math.round(it.התקדמות) + "%";
          row.appendChild(txt);
          row.appendChild(progWrap);
          row.appendChild(pct);
          a.appendChild(row);
        } else {
          a.textContent = i18nT(it.טקסט);
        }
        if (sel !== "" && String(it.ערך) === sel) {
          a.classList.add("active");
        }
        a.addEventListener("click", function () {
          var siblings = self.el.querySelectorAll(".list-group-item");
          for (var j = 0; j < siblings.length; j++) {
            siblings[j].classList.remove("active");
          }
          a.classList.add("active");
          self.value = String(it.ערך);
          postChange(self.id, self.type, self.value);
        });
        a.addEventListener("contextmenu", function (ev) {
          var siblings = self.el.querySelectorAll(".list-group-item");
          for (var j = 0; j < siblings.length; j++) {
            siblings[j].classList.remove("active");
          }
          a.classList.add("active");
          self.value = String(it.ערך);
          self.openCopyMenu(ev, it);
        });
        ul.appendChild(a);
      })(this.items[i]);
    }
  };
  ListCtrl.prototype.setItems = function (items) {
    this.items = normItems(items);
    this.render();
  };
  ListCtrl.prototype.setValue = function (v) {
    this.value = v == null || v === "" ? "" : String(v);
    this.render();
  };

  /* ——— אקורדיון ——— */
  function Accordion(opts) {
    Control.call(this, opts);
    this.type = "אקורדיון";
    this.rawItems = opts.פריטים || [];
  }
  Accordion.prototype = Object.create(Control.prototype);
  Accordion.prototype.constructor = Accordion;
  Accordion.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "accordion";
    this.render(wrap);
    return wrap;
  };
  Accordion.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    // מזהים באנגלית בלבד — מונע באגים ב־Selectors של Bootstrap עם עברית
    var safeBase = "acc_" + String(this.id).replace(/[^\w-]/g, "_");
    for (var i = 0; i < this.rawItems.length; i++) {
      var it = this.rawItems[i];
      var title = itemTitle(it);
      var body = itemBody(it) || title;
      var item = document.createElement("div");
      item.className = "accordion-item";
      var h = document.createElement("h2");
      h.className = "accordion-header";
      var btn = document.createElement("button");
      var open = i === 0;
      btn.type = "button";
      btn.className = "accordion-button" + (open ? "" : " collapsed");
      btn.setAttribute("aria-expanded", open ? "true" : "false");
      btn.textContent = title;
      h.appendChild(btn);
      var col = document.createElement("div");
      col.id = safeBase + "_c" + i;
      col.className = "accordion-collapse collapse" + (open ? " show" : "");
      var bd = document.createElement("div");
      bd.className = "accordion-body";
      bd.textContent = body;
      col.appendChild(bd);
      // פתיחה/סגירה ידנית לכל פאנל — בלי data-bs-toggle (לא קושר פאנלים)
      (function (button, panel) {
        button.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          var isOpen = panel.classList.contains("show");
          if (isOpen) {
            panel.classList.remove("show");
            button.classList.add("collapsed");
            button.setAttribute("aria-expanded", "false");
          } else {
            panel.classList.add("show");
            button.classList.remove("collapsed");
            button.setAttribute("aria-expanded", "true");
          }
        });
      })(btn, col);
      item.appendChild(h);
      item.appendChild(col);
      wrap.appendChild(item);
    }
  };
  Accordion.prototype.setItems = function (items) {
    this.rawItems = items || [];
    this.render();
  };

  /* ——— תצוגת עץ ——— */
  function treeToneClass(styleName) {
    var s = styleName == null ? "" : String(styleName);
    if (s === "הצלחה" || s === "ok" || s === "success") return "ok";
    if (s === "סכנה" || s === "danger" || s === "error") return "danger";
    if (s === "אזהרה" || s === "warn" || s === "warning") return "warn";
    if (s === "מידע" || s === "info") return "info";
    if (s === "משני" || s === "mute" || s === "secondary") return "mute";
    return "";
  }

  function normTreeNodes(items) {
    var out = [];
    if (!items) return out;
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (typeof it === "string" || typeof it === "number") {
        out.push({
          טקסט: String(it),
          ערך: String(it),
          תת_כותרת: "",
          איקון: "",
          סגנון: "",
          ילדים: []
        });
      } else if (it && typeof it === "object") {
        var label =
          it.טקסט != null
            ? String(it.טקסט)
            : it.כותרת != null
              ? String(it.כותרת)
              : String(it.ערך != null ? it.ערך : i);
        var val = it.ערך != null ? String(it.ערך) : label;
        var kids = it.ילדים || it.children || it.פריטים || [];
        var sub =
          it.תת_כותרת != null
            ? String(it.תת_כותרת)
            : it.משני != null
              ? String(it.משני)
              : it.subtitle != null
                ? String(it.subtitle)
                : "";
        var icon =
          it.איקון != null
            ? String(it.איקון)
            : it.icon != null
              ? String(it.icon)
              : "";
        var style =
          it.סגנון != null
            ? String(it.סגנון)
            : it.style != null
              ? String(it.style)
              : "";
        out.push({
          טקסט: label,
          ערך: val,
          תת_כותרת: sub,
          איקון: icon,
          סגנון: style,
          ילדים: normTreeNodes(kids)
        });
      }
    }
    return out;
  }

  function TreeView(opts) {
    Control.call(this, opts);
    this.type = "תצוגת_עץ";
    this.nodes = normTreeNodes(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this._open = Object.create(null);
  }
  TreeView.prototype = Object.create(Control.prototype);
  TreeView.prototype.constructor = TreeView;
  TreeView.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-tree";
    wrap.setAttribute("role", "tree");
    this.render(wrap);
    return wrap;
  };
  TreeView.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    this._renderNodes(wrap, this.nodes, 0);
  };
  TreeView.prototype._renderNodes = function (parent, nodes, depth) {
    var self = this;
    for (var i = 0; i < nodes.length; i++) {
      (function (node) {
        var hasKids = node.ילדים && node.ילדים.length > 0;
        var selected = self.value === node.ערך;
        var tone = treeToneClass(node.סגנון);
        var host = parent;
        if (depth === 0) {
          var card = document.createElement("div");
          card.className = "yod-tree-card";
          if (hasKids) card.classList.add("yod-tree-card-branch");
          if (tone) card.classList.add("yod-tree-tone-" + tone);
          if (selected) card.classList.add("is-selected");
          parent.appendChild(card);
          host = card;
        }
        var row = document.createElement("div");
        row.className = "yod-tree-row" + (depth === 0 ? " yod-tree-root" : " yod-tree-child");
        if (tone) row.classList.add("yod-tree-tone-" + tone);
        row.style.paddingInlineStart = depth * 0.85 + "rem";
        row.setAttribute("role", "treeitem");
        row.setAttribute("aria-selected", selected ? "true" : "false");

        if (hasKids) {
          var tog = document.createElement("button");
          tog.type = "button";
          tog.className = "yod-tree-toggle";
          tog.textContent = self._open[node.ערך] ? "▼" : "▶";
          tog.setAttribute("aria-label", "פתח/סגור");
          tog.addEventListener("click", function (ev) {
            ev.preventDefault();
            ev.stopPropagation();
            self._open[node.ערך] = !self._open[node.ערך];
            self.render();
          });
          row.appendChild(tog);
        }

        var lab = document.createElement("button");
        lab.type = "button";
        lab.className = "yod-tree-label" + (selected ? " active" : "");
        if (tone) lab.classList.add("yod-tree-st-" + tone);
        if (node.תת_כותרת) lab.classList.add("yod-tree-label-rich");

        var icoHtml = "";
        if (node.איקון) {
          icoHtml = menuIconHtml(node.איקון, node.טקסט);
          if (!icoHtml && ICO[node.איקון]) {
            icoHtml = svgIcon(node.איקון, "yod-tree-ico");
          }
        }
        if (!icoHtml && tone === "ok") icoHtml = svgIcon("check", "yod-tree-ico");
        if (!icoHtml && tone === "danger") icoHtml = svgIcon("x-circle", "yod-tree-ico");
        if (!icoHtml && tone === "warn") icoHtml = svgIcon("alert", "yod-tree-ico");
        if (!icoHtml && tone === "mute") icoHtml = svgIcon("pause", "yod-tree-ico");
        if (!icoHtml && tone === "info") icoHtml = svgIcon("globe", "yod-tree-ico");

        var titleText = escapeHtml(i18nT(node.טקסט || ""));
        var subText = node.תת_כותרת ? escapeHtml(String(node.תת_כותרת)) : "";
        lab.innerHTML =
          (icoHtml
            ? '<span class="yod-tree-ico-wrap" aria-hidden="true">' + icoHtml + "</span>"
            : "") +
          '<span class="yod-tree-label-body">' +
          '<span class="yod-tree-title">' +
          titleText +
          "</span>" +
          (subText
            ? '<span class="yod-tree-sub">' + subText + "</span>"
            : "") +
          "</span>";

        lab.addEventListener("click", function (ev) {
          ev.preventDefault();
          self.value = node.ערך;
          self.render();
          post({
            סוג: "אירוע",
            שם: "לחיצה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: node.ערך }
          });
          postChange(self.id, self.type, node.ערך);
        });

        lab.addEventListener("dblclick", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          self.value = node.ערך;
          post({
            סוג: "אירוע",
            שם: "לחיצה_כפולה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: node.ערך }
          });
        });

        row.appendChild(lab);
        host.appendChild(row);
        if (hasKids) {
          var kidsWrap = document.createElement("div");
          kidsWrap.className = "yod-tree-children";
          kidsWrap.style.display = self._open[node.ערך] ? "block" : "none";
          host.appendChild(kidsWrap);
          if (self._open[node.ערך]) {
            self._renderNodes(kidsWrap, node.ילדים, depth + 1);
          }
        }
      })(nodes[i]);
    }
  };
  TreeView.prototype.setItems = function (items) {
    this.nodes = normTreeNodes(items);
    this._openAll(this.nodes);
    this.render();
  };
  TreeView.prototype._openAll = function (nodes) {
    if (!nodes) return;
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i];
      if (n.ילדים && n.ילדים.length) {
        this._open[n.ערך] = true;
        this._openAll(n.ילדים);
      }
    }
  };
  TreeView.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.render();
  };

  /* ——— הצגת קבצים (עץ FileZilla) ——— */
  function fileIconKind(node) {
    if (node.סוג === "תיקיה") return "folder";
    var ext = (node.סיומת || "").toLowerCase().replace(/^\./, "");
    if (!ext && node.שם && node.שם.indexOf(".") >= 0) {
      ext = node.שם.split(".").pop().toLowerCase();
    }
    if (["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg", "ico"].indexOf(ext) >= 0) return "image";
    if (["zip", "rar", "7z", "tar", "gz", "bz2"].indexOf(ext) >= 0) return "archive";
    if (["txt", "md", "log", "csv", "json", "xml", "html", "css", "יוד", "yod"].indexOf(ext) >= 0) return "text";
    if (["js", "ts", "py", "go", "java", "c", "cpp", "h", "rs"].indexOf(ext) >= 0) return "code";
    if (["exe", "msi", "bat", "cmd", "dll"].indexOf(ext) >= 0) return "exec";
    if (["mp3", "wav", "ogg", "flac", "m4a"].indexOf(ext) >= 0) return "audio";
    if (["mp4", "avi", "mkv", "webm", "mov"].indexOf(ext) >= 0) return "video";
    return "file";
  }

  function formatFileSize(n) {
    var v = Number(n);
    if (!isFinite(v) || v < 0) return "";
    if (v < 1024) return String(Math.round(v)) + " B";
    if (v < 1048576) return (v / 1024).toFixed(1) + " KB";
    if (v < 1073741824) return (v / 1048576).toFixed(1) + " MB";
    return (v / 1073741824).toFixed(2) + " GB";
  }

  function formatFileTime(t) {
    if (t == null || t === "") return "";
    if (typeof t === "string") return t;
    var n = Number(t);
    if (!isFinite(n) || n <= 0) return "";
    if (n > 1e12) n = Math.floor(n / 1000);
    var d = new Date(n * 1000);
    if (isNaN(d.getTime())) return String(t);
    function pad(x) {
      return x < 10 ? "0" + x : String(x);
    }
    return (
      d.getFullYear() +
      "-" +
      pad(d.getMonth() + 1) +
      "-" +
      pad(d.getDate()) +
      " " +
      pad(d.getHours()) +
      ":" +
      pad(d.getMinutes())
    );
  }

  function normFileNodes(items) {
    var out = [];
    if (!items) return out;
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (typeof it === "string" || typeof it === "number") {
        out.push({
          שם: String(it),
          ערך: String(it),
          סוג: "קובץ",
          סיומת: "",
          גודל: 0,
          תאריך: "",
          הרשאות: "",
          בעלות: "",
          ילדים: [],
          יש_ילדים: false
        });
        continue;
      }
      if (!it || typeof it !== "object") continue;
      var name =
        it.שם != null
          ? String(it.שם)
          : it.name != null
            ? String(it.name)
            : it.טקסט != null
              ? String(it.טקסט)
              : String(it.ערך != null ? it.ערך : i);
      var val = it.ערך != null ? String(it.ערך) : it.value != null ? String(it.value) : name;
      var kind =
        it.סוג != null
          ? String(it.סוג)
          : it.type != null
            ? String(it.type)
            : it.תיקיה === true || it.isDir === true
              ? "תיקיה"
              : "קובץ";
      if (kind === "folder" || kind === "dir" || kind === "directory") kind = "תיקיה";
      if (kind === "file") kind = "קובץ";
      var kids = it.ילדים || it.children || it.פריטים || [];
      var hasKidsFlag =
        it.יש_ילדים === true ||
        it.hasChildren === true ||
        kind === "תיקיה" ||
        (kids && kids.length > 0);
      out.push({
        שם: name,
        ערך: val,
        סוג: kind,
        סיומת: it.סיומת != null ? String(it.סיומת) : it.ext != null ? String(it.ext) : "",
        גודל: it.גודל != null ? it.גודל : it.size != null ? it.size : 0,
        תאריך: it.תאריך != null ? it.תאריך : it.time != null ? it.time : it.mtime != null ? it.mtime : "",
        הרשאות: it.הרשאות != null ? String(it.הרשאות) : it.permissions != null ? String(it.permissions) : "",
        בעלות: it.בעלות != null ? String(it.בעלות) : it.owner != null ? String(it.owner) : "",
        ילדים: normFileNodes(kids),
        יש_ילדים: !!hasKidsFlag,
        _loaded: !!(kids && kids.length > 0) || kind !== "תיקיה"
      });
    }
    return out;
  }

  function FileBrowser(opts) {
    Control.call(this, opts);
    this.type = "הצגת_קבצים";
    this.nodes = normFileNodes(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this._open = Object.create(null);
  }
  FileBrowser.prototype = Object.create(Control.prototype);
  FileBrowser.prototype.constructor = FileBrowser;
  FileBrowser.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-files";
    wrap.setAttribute("role", "tree");
    this.render(wrap);
    return wrap;
  };
  FileBrowser.prototype._findNode = function (nodes, val) {
    if (!nodes) return null;
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].ערך === val) return nodes[i];
      var f = this._findNode(nodes[i].ילדים, val);
      if (f) return f;
    }
    return null;
  };
  FileBrowser.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    var head = document.createElement("div");
    head.className = "yod-files-head";
    head.innerHTML =
      '<span class="yod-files-col-icon"></span>' +
      '<span class="yod-files-col-name">שם</span>' +
      '<span class="yod-files-col-size">גודל</span>' +
      '<span class="yod-files-col-date">תאריך</span>' +
      '<span class="yod-files-col-perm">הרשאות</span>' +
      '<span class="yod-files-col-owner">בעלות</span>';
    wrap.appendChild(head);
    var body = document.createElement("div");
    body.className = "yod-files-body";
    wrap.appendChild(body);
    this._renderNodes(body, this.nodes, 0);
  };
  FileBrowser.prototype._renderNodes = function (parent, nodes, depth) {
    var self = this;
    if (!nodes) return;
    for (var i = 0; i < nodes.length; i++) {
      (function (node) {
        var expandable = node.סוג === "תיקיה" || node.יש_ילדים;
        var hasLoadedKids = node.ילדים && node.ילדים.length > 0;
        var row = document.createElement("div");
        row.className =
          "yod-files-row" + (self.value === node.ערך ? " active" : "");
        row.style.paddingInlineStart = 0.35 + depth * 0.85 + "rem";
        row.setAttribute("role", "treeitem");
        row.setAttribute("aria-selected", self.value === node.ערך ? "true" : "false");

        var tog = document.createElement("button");
        tog.type = "button";
        tog.className = "yod-files-toggle";
        if (!expandable) {
          tog.textContent = "·";
          tog.disabled = true;
        } else {
          tog.textContent = self._open[node.ערך] ? "▼" : "▶";
        }

        var icon = document.createElement("span");
        icon.className = "yod-files-icon yod-files-icon-" + fileIconKind(node);
        icon.setAttribute("aria-hidden", "true");

        var name = document.createElement("button");
        name.type = "button";
        name.className = "yod-files-name";
        name.textContent = node.שם;

        var size = document.createElement("span");
        size.className = "yod-files-col-size";
        size.textContent = node.סוג === "תיקיה" ? "" : formatFileSize(node.גודל);

        var date = document.createElement("span");
        date.className = "yod-files-col-date";
        date.textContent = formatFileTime(node.תאריך);

        var perm = document.createElement("span");
        perm.className = "yod-files-col-perm";
        perm.textContent = node.הרשאות || "";

        var owner = document.createElement("span");
        owner.className = "yod-files-col-owner";
        owner.textContent = node.בעלות || "";

        tog.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          if (!expandable) return;
          var willOpen = !self._open[node.ערך];
          self._open[node.ערך] = willOpen;
          if (willOpen && !hasLoadedKids) {
            post({
              סוג: "אירוע",
              שם: "פתיחה",
              ערכים: { מזהה: self.id, סוג: self.type, ערך: node.ערך }
            });
          }
          self.render();
        });
        function selectRow(ev) {
          if (ev) ev.preventDefault();
          self.value = node.ערך;
          self.render();
          post({
            סוג: "אירוע",
            שם: "לחיצה",
            ערכים: { מזהה: self.id, סוג: self.type, ערך: node.ערך }
          });
          postChange(self.id, self.type, node.ערך);
        }
        name.addEventListener("click", selectRow);
        row.addEventListener("dblclick", function (ev) {
          if (expandable) {
            ev.preventDefault();
            if (!self._open[node.ערך]) {
              self._open[node.ערך] = true;
              if (!hasLoadedKids) {
                post({
                  סוג: "אירוע",
                  שם: "פתיחה",
                  ערכים: { מזהה: self.id, סוג: self.type, ערך: node.ערך }
                });
              }
              self.render();
            }
          }
          selectRow(ev);
        });

        row.appendChild(tog);
        row.appendChild(icon);
        row.appendChild(name);
        row.appendChild(size);
        row.appendChild(date);
        row.appendChild(perm);
        row.appendChild(owner);
        parent.appendChild(row);

        if (expandable && self._open[node.ערך] && hasLoadedKids) {
          self._renderNodes(parent, node.ילדים, depth + 1);
        }
      })(nodes[i]);
    }
  };
  FileBrowser.prototype.setItems = function (items) {
    this.nodes = normFileNodes(items);
    this._open = Object.create(null);
    this.render();
  };
  FileBrowser.prototype.setChildren = function (parentVal, kids) {
    var node = this._findNode(this.nodes, String(parentVal));
    if (!node) return;
    node.ילדים = normFileNodes(kids || []);
    node.יש_ילדים = true;
    node._loaded = true;
    this._open[node.ערך] = true;
    this.render();
  };
  FileBrowser.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.render();
  };

  /* ——— קרוסלה ——— */
  function normSlides(items) {
    var out = [];
    if (!items) return out;
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (typeof it === "string" || typeof it === "number") {
        out.push({ כותרת: "", גוף: String(it) });
      } else if (it && typeof it === "object") {
        out.push({
          כותרת:
            it.כותרת != null
              ? String(it.כותרת)
              : it.טקסט != null
                ? String(it.טקסט)
                : "",
          גוף:
            it.גוף != null
              ? String(it.גוף)
              : it.ערך != null
                ? String(it.ערך)
                : it.טקסט != null
                  ? String(it.טקסט)
                  : ""
        });
      }
    }
    return out;
  }

  function Carousel(opts) {
    Control.call(this, opts);
    this.type = "קרוסלה";
    this.slides = normSlides(opts.פריטים);
    this.index =
      opts.ערך != null ? Math.max(0, Number(opts.ערך) || 0) : 0;
    this.trackEl = null;
    this.dotsEl = null;
    this.labelEl = null;
  }
  Carousel.prototype = Object.create(Control.prototype);
  Carousel.prototype.constructor = Carousel;
  Carousel.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-carousel";
    var viewport = document.createElement("div");
    viewport.className = "yod-carousel-viewport";
    var track = document.createElement("div");
    track.className = "yod-carousel-track";
    viewport.appendChild(track);
    var prev = document.createElement("button");
    prev.type = "button";
    prev.className = "btn btn-sm btn-outline-secondary yod-carousel-prev";
    prev.textContent = "‹";
    prev.setAttribute("aria-label", "הקודם");
    var next = document.createElement("button");
    next.type = "button";
    next.className = "btn btn-sm btn-outline-secondary yod-carousel-next";
    next.textContent = "›";
    next.setAttribute("aria-label", "הבא");
    var bar = document.createElement("div");
    bar.className = "yod-carousel-bar";
    var dots = document.createElement("div");
    dots.className = "yod-carousel-dots";
    var lab = document.createElement("span");
    lab.className = "yod-carousel-count text-secondary small";
    bar.appendChild(prev);
    bar.appendChild(dots);
    bar.appendChild(next);
    bar.appendChild(lab);
    wrap.appendChild(viewport);
    wrap.appendChild(bar);
    this.trackEl = track;
    this.dotsEl = dots;
    this.labelEl = lab;
    this.prevBtn = prev;
    this.nextBtn = next;
    this.render();
    return wrap;
  };
  Carousel.prototype.bindEvents = function () {
    var self = this;
    if (this.prevBtn) {
      this.prevBtn.addEventListener("click", function () {
        self.prev();
      });
    }
    if (this.nextBtn) {
      this.nextBtn.addEventListener("click", function () {
        self.next();
      });
    }
  };
  Carousel.prototype.render = function () {
    if (!this.trackEl) return;
    this.trackEl.innerHTML = "";
    if (this.dotsEl) this.dotsEl.innerHTML = "";
    if (!this.slides.length) {
      this.trackEl.innerHTML = '<div class="yod-carousel-slide text-secondary">אין שקופיות</div>';
      if (this.labelEl) this.labelEl.textContent = "0/0";
      return;
    }
    if (this.index >= this.slides.length) this.index = 0;
    if (this.index < 0) this.index = this.slides.length - 1;
    var self = this;
    for (var i = 0; i < this.slides.length; i++) {
      var s = this.slides[i];
      var slide = document.createElement("div");
      slide.className =
        "yod-carousel-slide" + (i === this.index ? " active" : "");
      slide.style.display = i === this.index ? "block" : "none";
      if (s.כותרת) {
        var h = document.createElement("div");
        h.className = "yod-carousel-title";
        h.textContent = s.כותרת;
        slide.appendChild(h);
      }
      var body = document.createElement("div");
      body.className = "yod-carousel-body";
      body.textContent = s.גוף || s.כותרת || "";
      slide.appendChild(body);
      this.trackEl.appendChild(slide);
      if (this.dotsEl) {
        (function (n) {
          var d = document.createElement("button");
          d.type = "button";
          d.className =
            "yod-carousel-dot" + (n === self.index ? " active" : "");
          d.setAttribute("aria-label", "שקופית " + (n + 1));
          d.addEventListener("click", function () {
            self.setValue(n);
          });
          self.dotsEl.appendChild(d);
        })(i);
      }
    }
    if (this.labelEl) {
      this.labelEl.textContent = this.index + 1 + "/" + this.slides.length;
    }
  };
  Carousel.prototype.setItems = function (items) {
    this.slides = normSlides(items);
    this.index = 0;
    this.render();
  };
  Carousel.prototype.setValue = function (v) {
    var n = Number(v) || 0;
    if (n < 0) n = 0;
    if (this.slides.length && n >= this.slides.length) n = this.slides.length - 1;
    this.index = n;
    this.render();
    postChange(this.id, this.type, this.index);
  };
  Carousel.prototype.next = function () {
    if (!this.slides.length) return;
    this.setValue((this.index + 1) % this.slides.length);
  };
  Carousel.prototype.prev = function () {
    if (!this.slides.length) return;
    this.setValue((this.index - 1 + this.slides.length) % this.slides.length);
  };

  /* ——— לשוניות ——— */
  function Tabs(opts) {
    Control.call(this, opts);
    this.type = "לשוניות";
    this.rawItems = opts.פריטים || [];
    this.value = opts.ערך != null ? String(opts.ערך) : "";
  }
  Tabs.prototype = Object.create(Control.prototype);
  Tabs.prototype.constructor = Tabs;
  Tabs.prototype.createElement = function () {
    var wrap = document.createElement("div");
    this.render(wrap);
    return wrap;
  };
  Tabs.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    var ul = document.createElement("ul");
    ul.className = "nav nav-tabs";
    var panes = document.createElement("div");
    panes.className = "tab-content border border-top-0 p-3";
    var self = this;
    var firstVal = "";
    for (var i = 0; i < this.rawItems.length; i++) {
      var it = this.rawItems[i];
      var title = itemTitle(it);
      var body = itemBody(it);
      var val = typeof it === "object" && it && it.ערך != null ? String(it.ערך) : title;
      if (!firstVal) firstVal = val;
      var active = (this.value && this.value === val) || (!this.value && i === 0);
      var li = document.createElement("li");
      li.className = "nav-item";
      var a = document.createElement("button");
      a.type = "button";
      a.className = "nav-link" + (active ? " active" : "");
      a.textContent = title;
      (function (v) {
        a.addEventListener("click", function () {
          self.setValue(v);
          postChange(self.id, self.type, self.value);
        });
      })(val);
      li.appendChild(a);
      ul.appendChild(li);
      var pane = document.createElement("div");
      pane.className = "tab-pane fade" + (active ? " show active" : "");
      pane.setAttribute("data-val", val);
      pane.textContent = body || title;
      panes.appendChild(pane);
    }
    if (!this.value) this.value = firstVal;
    wrap.appendChild(ul);
    wrap.appendChild(panes);
  };
  Tabs.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    if (!this.el) return;
    var links = this.el.querySelectorAll(".nav-link");
    var panes = this.el.querySelectorAll(".tab-pane");
    for (var i = 0; i < links.length; i++) {
      var on = panes[i] && panes[i].getAttribute("data-val") === this.value;
      links[i].classList.toggle("active", on);
      if (panes[i]) {
        panes[i].classList.toggle("show", on);
        panes[i].classList.toggle("active", on);
      }
    }
  };

  /* ——— פירורי לחם ——— */
  function Breadcrumbs(opts) {
    Control.call(this, opts);
    this.type = "פירורי_לחם";
    this.items = normItems(opts.פריטים);
  }
  Breadcrumbs.prototype = Object.create(Control.prototype);
  Breadcrumbs.prototype.constructor = Breadcrumbs;
  Breadcrumbs.prototype.createElement = function () {
    var nav = document.createElement("nav");
    var ol = document.createElement("ol");
    ol.className = "breadcrumb";
    nav.appendChild(ol);
    this.list = ol;
    this.render();
    return nav;
  };
  Breadcrumbs.prototype.render = function () {
    if (!this.list) return;
    this.list.innerHTML = "";
    var self = this;
    for (var i = 0; i < this.items.length; i++) {
      (function (it, last) {
        var li = document.createElement("li");
        li.className = "breadcrumb-item" + (last ? " active" : "");
        if (last) {
          li.textContent = i18nT(it.טקסט);
        } else {
          var a = document.createElement("a");
          a.href = "#";
          a.textContent = i18nT(it.טקסט);
          a.addEventListener("click", function (ev) {
            ev.preventDefault();
            post({
              סוג: "אירוע",
              שם: "לחיצה",
              ערכים: { מזהה: self.id, סוג: self.type, ערך: it.ערך }
            });
          });
          li.appendChild(a);
        }
        self.list.appendChild(li);
      })(this.items[i], i === this.items.length - 1);
    }
  };
  Breadcrumbs.prototype.setItems = function (items) {
    this.items = normItems(items);
    this.render();
  };

  /* ——— סרגל עליון (Navbar + תפריט מקונן) ——— */
  function TopNav(opts) {
    Control.call(this, opts);
    this.type = "סרגל_עליון";
    this.sourceTitle = opts.כותרת != null ? String(opts.כותרת) : this.sourceText || "";
    this.title = i18nT(this.sourceTitle);
    this.items = normMenuItems(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.brandIcon = opts.איקון != null ? String(opts.איקון) : "";
    this.brandEl = null;
    this.listEl = null;
    this.megaHost = null;
  }
  TopNav.prototype = Object.create(Control.prototype);
  TopNav.prototype.constructor = TopNav;
  TopNav.prototype.createElement = function () {
    var nav = document.createElement("nav");
    nav.className = "navbar navbar-expand yod-topnav px-3 py-2";
    var brand = document.createElement("span");
    brand.className = "navbar-brand mb-0 h1 fs-5";
    var mark;
    var brandSrc = this.brandIcon || "";
    var isBrandImg =
      brandSrc.indexOf("data:") === 0 ||
      brandSrc.indexOf("file:") === 0 ||
      brandSrc.indexOf("http:") === 0 ||
      brandSrc.indexOf("https:") === 0 ||
      /\.(png|jpe?g|gif|webp|svg|ico)(\?|#|$)/i.test(brandSrc);
    if (brandSrc && ICO[brandSrc]) {
      mark = document.createElement("span");
      mark.className = "yod-brand-mark yod-brand-ico";
      mark.innerHTML = svgIcon(brandSrc, "");
    } else if (brandSrc && isBrandImg) {
      mark = document.createElement("img");
      mark.className = "yod-brand-mark";
      mark.alt = this.title || "לוגו";
      mark.width = 28;
      mark.height = 28;
      mark.decoding = "async";
      mark.src = brandSrc;
    } else if (brandSrc) {
      mark = document.createElement("span");
      mark.className = "yod-brand-mark yod-brand-emoji";
      mark.textContent = brandSrc;
    } else {
      mark = document.createElement("img");
      mark.className = "yod-brand-mark";
      mark.alt = "יוד";
      mark.width = 28;
      mark.height = 28;
      mark.decoding = "async";
      mark.src = global.__יוד_לוגו || "לוגו-יוד.png";
    }
    var titleSpan = document.createElement("span");
    titleSpan.className = "yod-brand-title";
    titleSpan.textContent = this.title || "תפריט";
    brand.appendChild(mark);
    brand.appendChild(titleSpan);
    var list = document.createElement("div");
    list.className = "navbar-nav flex-row flex-wrap gap-1 yod-topnav-menus";
    var megaHost = document.createElement("div");
    megaHost.className = "yod-menu-mega-host";
    nav.appendChild(brand);
    nav.appendChild(list);
    nav.appendChild(megaHost);
    this.brandEl = brand;
    this.brandTitleEl = titleSpan;
    this.listEl = list;
    this.megaHost = megaHost;
    this.render();
    return nav;
  };
  TopNav.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    var r = root();
    if (r && this.el && this.el.parentNode === r) {
      r.insertBefore(this.el, r.firstChild);
    }
    ensureMainScroll();
    placeChromeLayout();
    return this;
  };
  TopNav.prototype._fireLeaf = function (val) {
    postMenuClick(this, val);
    this.render();
  };
  TopNav.prototype.render = function () {
    if (!this.listEl) return;
    closeAllYodMenus();
    this.listEl.innerHTML = "";
    if (this.megaHost) this.megaHost.innerHTML = "";
    var self = this;
    for (var i = 0; i < this.items.length; i++) {
      (function (it) {
        if (it.סוג === "מפריד" || it.סוג === "ריק" || it.סוג === "כותרת") {
          return;
        }
        var hasKids = it.ילדים && it.ילדים.length > 0;
        if (!hasKids) {
          var a = document.createElement("button");
          a.type = "button";
          var active = self.value && (self.value === it.ערך || self.value === it.טקסט);
          a.className = "nav-link btn btn-link px-2" + (active ? " active fw-semibold" : "");
          a.setAttribute("data-action", it.ערך || it.טקסט || "");
          if (it.מושבת) a.disabled = true;
          decorateMenuLabel(a, i18nT(it.טקסט || ""), it.איקון);
          if (!it.איקון && LABEL_ICO[it.טקסט]) {
            decorateButtonLabel(a, i18nT(it.טקסט || ""));
          }
          a.addEventListener("click", function (ev) {
            ev.preventDefault();
            self._fireLeaf(it.ערך);
          });
          self.listEl.appendChild(a);
          return;
        }
        var rootEl = document.createElement("div");
        rootEl.className = "yod-menu-root";
        var tog = document.createElement("button");
        tog.type = "button";
        tog.className = "nav-link btn btn-link px-2 yod-menu-toggle";
        tog.setAttribute("aria-expanded", "false");
        tog.setAttribute("aria-haspopup", "true");
        if (it.מושבת) tog.disabled = true;
        decorateMenuLabel(tog, i18nT(it.טקסט || ""), it.איקון);
        var caret = document.createElement("span");
        caret.className = "yod-menu-caret";
        caret.setAttribute("aria-hidden", "true");
        tog.appendChild(caret);
        var panel;
        var isMega = !!it.רוחב_מלא;
        if (isMega) {
          panel = buildMenuPanel(it.ילדים, self, 0);
          panel.className = "yod-menu yod-menu-mega";
          if (self.megaHost) self.megaHost.appendChild(panel);
          else rootEl.appendChild(panel);
        } else {
          panel = buildMenuPanel(it.ילדים, self, 0);
          rootEl.appendChild(panel);
        }
        tog.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          var wasOpen = rootEl.classList.contains("open");
          closeAllYodMenus();
          if (!wasOpen) {
            rootEl.classList.add("open");
            tog.setAttribute("aria-expanded", "true");
            if (isMega) panel.classList.add("show");
          }
        });
        rootEl.appendChild(tog);
        self.listEl.appendChild(rootEl);
      })(this.items[i]);
    }
  };
  TopNav.prototype.setTitle = function (raw) {
    this.sourceTitle = raw == null ? "" : String(raw);
    this.title = i18nT(this.sourceTitle);
    if (this.brandTitleEl) {
      this.brandTitleEl.textContent = this.title || i18nT("תפריט");
    } else if (this.brandEl) {
      this.brandEl.textContent = this.title || i18nT("תפריט");
    }
  };
  TopNav.prototype.setItems = function (items) {
    this.items = normMenuItems(items);
    this.render();
  };
  TopNav.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.render();
  };
  TopNav.prototype.applyI18n = function () {
    this.title = i18nT(this.sourceTitle || this.sourceText || "");
    if (this.brandTitleEl) {
      this.brandTitleEl.textContent = this.title || i18nT("תפריט");
    } else if (this.brandEl && !this.brandEl.querySelector("img")) {
      this.brandEl.textContent = this.title || i18nT("תפריט");
    }
    this.render();
  };

  /* ——— סרגל צד (Sidebar + קבוצות מקוננות) ——— */
  function SideNav(opts) {
    Control.call(this, opts);
    this.type = "סרגל_צד";
    this.sourceTitle = opts.כותרת != null ? String(opts.כותרת) : this.sourceText || "";
    this.title = i18nT(this.sourceTitle);
    this.items = normMenuItems(opts.פריטים);
    this.value = opts.ערך != null ? String(opts.ערך) : "";
    this.openGroups = {};
    this.titleEl = null;
    this.listEl = null;
  }
  SideNav.prototype = Object.create(Control.prototype);
  SideNav.prototype.constructor = SideNav;
  SideNav.prototype.createElement = function () {
    var wrap = document.createElement("nav");
    wrap.className = "yod-sidenav rounded p-2";
    wrap.setAttribute("aria-label", this.title || "תפריט צד");
    var tit = document.createElement("div");
    tit.className = "yod-sidenav-title px-2 py-1 mb-1";
    tit.textContent = this.title || "";
    if (!this.title) tit.style.display = "none";
    var list = document.createElement("div");
    list.className = "nav flex-column nav-pills gap-1 yod-sidenav-list";
    wrap.appendChild(tit);
    wrap.appendChild(list);
    this.titleEl = tit;
    this.listEl = list;
    this.render();
    return wrap;
  };
  SideNav.prototype._renderItems = function (container, items, depth) {
    depth = depth || 0;
    var self = this;
    for (var i = 0; i < items.length; i++) {
      (function (it) {
        if (it.סוג === "מפריד") {
          var sep = document.createElement("div");
          sep.className = "yod-menu-sep yod-sidenav-sep";
          container.appendChild(sep);
          return;
        }
        if (it.סוג === "ריק") {
          var sp = document.createElement("div");
          sp.className = "yod-menu-spacer yod-sidenav-spacer";
          container.appendChild(sp);
          return;
        }
        if (it.סוג === "כותרת") {
          var hd = document.createElement("div");
          hd.className = "yod-menu-header yod-sidenav-header";
          hd.textContent = i18nT(it.טקסט || "");
          container.appendChild(hd);
          return;
        }
        var hasKids = it.ילדים && it.ילדים.length > 0;
        if (hasKids) {
          var gid = (it.ערך || it.טקסט || "") + "@" + depth;
          var open = !!self.openGroups[gid];
          var group = document.createElement("div");
          group.className = "yod-sidenav-group" + (open ? " open" : "");
          var gbtn = document.createElement("button");
          gbtn.type = "button";
          gbtn.className = "nav-link text-start yod-sidenav-group-btn";
          if (it.מושבת) gbtn.disabled = true;
          decorateMenuLabel(gbtn, it.טקסט, it.איקון);
          var caret = document.createElement("span");
          caret.className = "yod-sidenav-caret";
          caret.setAttribute("aria-hidden", "true");
          gbtn.appendChild(caret);
          gbtn.addEventListener("click", function (ev) {
            ev.preventDefault();
            self.openGroups[gid] = !self.openGroups[gid];
            self.render();
          });
          var kids = document.createElement("div");
          kids.className = "yod-sidenav-children";
          if (open) {
            self._renderItems(kids, it.ילדים, depth + 1);
          }
          group.appendChild(gbtn);
          group.appendChild(kids);
          container.appendChild(group);
          return;
        }
        var a = document.createElement("button");
        a.type = "button";
        var active = self.value && (self.value === it.ערך || self.value === it.טקסט);
        a.className = "nav-link text-start" + (active ? " active" : "");
        if (it.מושבת) a.disabled = true;
        decorateMenuLabel(a, i18nT(it.טקסט || ""), it.איקון);
        a.addEventListener("click", function (ev) {
          ev.preventDefault();
          postMenuClick(self, it.ערך);
          self.render();
        });
        container.appendChild(a);
      })(items[i]);
    }
  };
  SideNav.prototype.render = function () {
    if (!this.listEl) return;
    this.listEl.innerHTML = "";
    this._renderItems(this.listEl, this.items, 0);
  };
  SideNav.prototype.setTitle = function (raw) {
    this.sourceTitle = raw == null ? "" : String(raw);
    this.title = i18nT(this.sourceTitle);
    if (this.titleEl) {
      this.titleEl.textContent = this.title;
      this.titleEl.style.display = this.title ? "" : "none";
    }
  };
  SideNav.prototype.setItems = function (items) {
    this.items = normMenuItems(items);
    this.render();
  };
  SideNav.prototype.setValue = function (v) {
    this.value = v == null ? "" : String(v);
    this.render();
  };

  /* ——— עמודים ——— */
  function Pagination(opts) {
    Control.call(this, opts);
    this.type = "עמודים";
    this.total = opts.סהכ != null ? Number(opts.סהכ) || 5 : 5;
    this.value = opts.ערך != null ? Number(opts.ערך) || 1 : 1;
  }
  Pagination.prototype = Object.create(Control.prototype);
  Pagination.prototype.constructor = Pagination;
  Pagination.prototype.createElement = function () {
    var nav = document.createElement("nav");
    var ul = document.createElement("ul");
    ul.className = "pagination";
    nav.appendChild(ul);
    this.list = ul;
    this.render();
    return nav;
  };
  Pagination.prototype.render = function () {
    if (!this.list) return;
    this.list.innerHTML = "";
    var self = this;
    for (var i = 1; i <= this.total; i++) {
      (function (n) {
        var li = document.createElement("li");
        li.className = "page-item" + (n === self.value ? " active" : "");
        var a = document.createElement("button");
        a.type = "button";
        a.className = "page-link";
        a.textContent = String(n);
        a.addEventListener("click", function () {
          self.setValue(n);
          postChange(self.id, self.type, self.value);
        });
        li.appendChild(a);
        self.list.appendChild(li);
      })(i);
    }
  };
  Pagination.prototype.setValue = function (v) {
    this.value = Number(v) || 1;
    this.render();
  };

  /* ——— מדריך שלבים ——— */
  function Stepper(opts) {
    Control.call(this, opts);
    this.type = "מדריך_שלבים";
    this.steps = [];
    var raw = opts.שלבים || [];
    for (var i = 0; i < raw.length; i++) this.steps.push(itemTitle(raw[i]));
    this.value = opts.ערך != null ? Number(opts.ערך) || 1 : 1;
  }
  Stepper.prototype = Object.create(Control.prototype);
  Stepper.prototype.constructor = Stepper;
  Stepper.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-stepper d-flex flex-wrap gap-2 align-items-center";
    this.render(wrap);
    return wrap;
  };
  Stepper.prototype.render = function (wrap) {
    wrap = wrap || this.el;
    if (!wrap) return;
    wrap.innerHTML = "";
    var self = this;
    for (var i = 0; i < this.steps.length; i++) {
      (function (n, label) {
        var b = document.createElement("button");
        b.type = "button";
        b.className =
          "btn btn-sm " + (n === self.value ? "btn-primary" : "btn-outline-secondary");
        b.textContent = n + ". " + label;
        b.addEventListener("click", function () {
          self.setValue(n);
          postChange(self.id, self.type, self.value);
        });
        wrap.appendChild(b);
        if (n < self.steps.length) {
          var sp = document.createElement("span");
          sp.className = "text-secondary";
          sp.textContent = "‹";
          wrap.appendChild(sp);
        }
      })(i + 1, this.steps[i]);
    }
  };
  Stepper.prototype.setValue = function (v) {
    this.value = Number(v) || 1;
    this.render();
  };

  /* ——— מכל / רשת / שלד ——— */
  function Box(opts) {
    Control.call(this, opts);
    this.type = "מכל";
    this.kind = opts.סוג_מכל || "רגיל";
  }
  Box.prototype = Object.create(Control.prototype);
  Box.prototype.constructor = Box;
  Box.prototype.createElement = function () {
    var d = document.createElement("div");
    if (this.kind === "שורה") {
      d.className = "yod-box yod-box-row";
    } else if (this.kind === "צד") {
      d.className = "yod-box yod-box-col yod-box-side border p-2";
    } else if (this.kind === "יומן") {
      d.className = "yod-box yod-box-col yod-box-log yod-log-footer";
      d.setAttribute("data-yod-box", "יומן");
    } else if (this.kind === "ראשי" || this.kind === "עמודה") {
      d.className = "yod-box yod-box-col border p-2";
    } else if (this.kind === "צר") {
      d.className = "yod-box container-sm border p-2";
    } else if (this.kind === "רחב") {
      d.className = "yod-box container-fluid border p-2";
    } else {
      d.className = "yod-box border p-2";
    }
    return d;
  };
  Box.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    if (this.kind === "יומן" && this.el) {
      var r = root();
      if (r) {
        ensureMainScroll();
        r.appendChild(this.el);
        placeChromeLayout();
      }
    }
    return this;
  };

  /* ——— כרזת פרסום (באנר נעוץ בתחתית) ——— */
  function AdBanner(opts) {
    Control.call(this, opts);
    this.type = "כרזת_פרסום";
    this.link = "";
  }
  AdBanner.prototype = Object.create(Control.prototype);
  AdBanner.prototype.constructor = AdBanner;
  AdBanner.prototype.createElement = function () {
    // ברירת מחדל: מקופל (d-none) עד שמגיעה פרסומת ראשונה — לא תופס מקום
    var d = document.createElement("div");
    d.className = "yod-adbar d-none";
    d.setAttribute("data-yod-box", "פרסומת");
    var a = document.createElement("a");
    a.className = "yod-adbar-link";
    a.href = "#";
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    var img = document.createElement("img");
    img.className = "yod-adbar-img";
    img.alt = "";
    a.appendChild(img);
    var tag = document.createElement("span");
    tag.className = "yod-adbar-tag";
    tag.textContent = i18nT("פרסומת");
    d.appendChild(a);
    d.appendChild(tag);
    this.linkEl = a;
    this.imgEl = img;
    this.tagEl = tag;
    return d;
  };
  AdBanner.prototype.mount = function (parent) {
    Control.prototype.mount.call(this, parent);
    if (this.el) {
      var r = root();
      if (r) {
        ensureMainScroll();
        r.appendChild(this.el);
        placeChromeLayout();
      }
    }
    return this;
  };
  AdBanner.prototype.bindEvents = function () {
    var self = this;
    this.el.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      if (!self.link) return;
      post({
        סוג: "אירוע",
        שם: "לחיצה",
        ערכים: { מזהה: self.id, סוג: "כרזת_פרסום", ערך: self.link }
      });
    });
  };
  AdBanner.prototype.setAd = function (v) {
    v = v || {};
    var img = v.תמונה != null ? String(v.תמונה) : "";
    this.link = v.קישור != null ? String(v.קישור) : "";
    var title = v.כותרת != null ? String(v.כותרת) : "";
    if (this.linkEl) {
      this.linkEl.href = this.link || "#";
      this.linkEl.title = title;
    }
    if (this.imgEl && img) {
      var el = this.imgEl;
      el.alt = title;
      // מעבר fade: מנמיכים שקיפות, ומחזירים כשהתמונה נטענת
      el.style.opacity = "0";
      el.onload = function () {
        el.style.opacity = "1";
      };
      el.onerror = function () {
        el.style.opacity = "1";
      };
      el.src = img;
      this.show();
    }
  };
  AdBanner.prototype.show = function () {
    if (!this.el) return;
    this.el.classList.remove("d-none");
    placeChromeLayout();
  };
  AdBanner.prototype.hide = function () {
    if (!this.el) return;
    this.el.classList.add("d-none");
  };
  AdBanner.prototype.applyI18n = function () {
    if (this.tagEl) this.tagEl.textContent = i18nT("פרסומת");
  };

  function Grid(opts) {
    Control.call(this, opts);
    this.type = "רשת";
    this.cols = opts.עמודות != null ? Number(opts.עמודות) || 2 : 2;
    this.cells = opts.תאים || [];
  }
  Grid.prototype = Object.create(Control.prototype);
  Grid.prototype.constructor = Grid;
  Grid.prototype.createElement = function () {
    var row = document.createElement("div");
    row.className = "row g-2";
    this.render(row);
    return row;
  };
  Grid.prototype.render = function (row) {
    row = row || this.el;
    if (!row) return;
    row.innerHTML = "";
    var span = Math.max(1, Math.floor(12 / this.cols));
    for (var i = 0; i < this.cells.length; i++) {
      var col = document.createElement("div");
      col.className = "col-" + span;
      var card = document.createElement("div");
      card.className = "border rounded p-2 h-100";
      card.textContent = itemTitle(this.cells[i]);
      col.appendChild(card);
      row.appendChild(col);
    }
  };
  Grid.prototype.setCells = function (cells) {
    this.cells = cells || [];
    this.render();
  };

  function Skeleton(opts) {
    Control.call(this, opts);
    this.type = "שלד_טעינה";
    this.rows = opts.שורות != null ? Number(opts.שורות) || 3 : 3;
  }
  Skeleton.prototype = Object.create(Control.prototype);
  Skeleton.prototype.constructor = Skeleton;
  Skeleton.prototype.createElement = function () {
    var wrap = document.createElement("div");
    wrap.className = "yod-skeleton";
    for (var i = 0; i < this.rows; i++) {
      var line = document.createElement("div");
      line.className = "placeholder-glow mb-2";
      line.innerHTML = '<span class="placeholder col-' + (i === this.rows - 1 ? "7" : "12") + '"></span>';
      wrap.appendChild(line);
    }
    return wrap;
  };

  var FACTORIES = {
    כפתור: function (opts) {
      return new Button(opts);
    },
    שדה: function (opts) {
      return new Field(opts);
    },
    אזור_טקסט: function (opts) {
      return new TextArea(opts);
    },
    תווית: function (opts) {
      return new Label(opts);
    },
    תיבת_סימון: function (opts) {
      return new CheckLike(opts, false);
    },
    מתג: function (opts) {
      return new CheckLike(opts, true);
    },
    לחצן_אפשרויות: function (opts) {
      return new RadioGroup(opts);
    },
    תיבת_בחירה: function (opts) {
      return new SelectBox(opts);
    },
    סרגל: function (opts) {
      return new Slider(opts);
    },
    בורר_צבע: function (opts) {
      return new ColorPicker(opts);
    },
    בורר_קבצים: function (opts) {
      return new FilePicker(opts);
    },
    דוגם_צבע: function (opts) {
      return new ColorSampler(opts);
    },
    בורר_תאריך: function (opts) {
      return new DatePicker(opts);
    },
    בורר_שעה: function (opts) {
      return new TimePicker(opts);
    },
    מפריד: function (opts) {
      return new Divider(opts);
    },
    מרווח: function (opts) {
      return new Spacer(opts);
    },
    תג: function (opts) {
      return new Badge(opts);
    },
    התרעה: function (opts) {
      return new AlertBox(opts);
    },
    פס_התקדמות: function (opts) {
      return new Progress(opts);
    },
    טעינה: function (opts) {
      return new Spinner(opts);
    },
    הסבר_צף: function (opts) {
      return new TooltipCtrl(opts);
    },
    הודעה_קופצת: function (opts) {
      return new ToastCtrl(opts);
    },
    חלון_מודאלי: function (opts) {
      return new ModalCtrl(opts);
    },
    חלון_צף: function (opts) {
      return new PopoverCtrl(opts);
    },
    דירוג: function (opts) {
      return new Rating(opts);
    },
    השלמה_אוטומטית: function (opts) {
      return new Autocomplete(opts);
    },
    כפתור_מפוצל: function (opts) {
      return new SplitButton(opts);
    },
    כפתור_איקון: function (opts) {
      return new IconButton(opts);
    },
    קבוצת_כפתורים: function (opts) {
      return new ButtonGroup(opts);
    },
    כרטיס: function (opts) {
      return new Card(opts);
    },
    טבלה: function (opts) {
      return new DataTable(opts);
    },
    תמונת_פרופיל: function (opts) {
      return new Avatar(opts);
    },
    רשימה: function (opts) {
      return new ListCtrl(opts);
    },
    אקורדיון: function (opts) {
      return new Accordion(opts);
    },
    תצוגת_עץ: function (opts) {
      return new TreeView(opts);
    },
    הצגת_קבצים: function (opts) {
      return new FileBrowser(opts);
    },
    קרוסלה: function (opts) {
      return new Carousel(opts);
    },
    לשוניות: function (opts) {
      return new Tabs(opts);
    },
    פירורי_לחם: function (opts) {
      return new Breadcrumbs(opts);
    },
    סרגל_עליון: function (opts) {
      return new TopNav(opts);
    },
    סרגל_צד: function (opts) {
      return new SideNav(opts);
    },
    עמודים: function (opts) {
      return new Pagination(opts);
    },
    מדריך_שלבים: function (opts) {
      return new Stepper(opts);
    },
    מכל: function (opts) {
      return new Box(opts);
    },
    רשת: function (opts) {
      return new Grid(opts);
    },
    שלד_טעינה: function (opts) {
      return new Skeleton(opts);
    },
    רשת_נתונים: function (opts) {
      return new DataGrid(opts);
    },
    כרזת_פרסום: function (opts) {
      return new AdBanner(opts);
    }
  };

  function createControl(opts) {
    var kind = opts.סוג || opts.type || "כפתור";
    var factory = FACTORIES[kind];
    if (!factory) {
      console.warn("סוג פקד לא מוכר:", kind);
      return null;
    }
    var id = opts.מזהה || opts.id;
    if (id && controls[id]) {
      try {
        controls[id].remove();
      } catch (e) {}
    }
    var c = factory(opts);
    c.mount();
    return c;
  }

  function applyCommand(data) {
    if (!data || data.סוג !== "פקודה") return;
    var name = data.שם;
    var v = data.ערכים || {};

    if (name === "צור") {
      createControl(v);
      return;
    }
    if (name === "קבע_טקסט") {
      var c1 = controls[v.מזהה];
      if (c1) {
        if (typeof c1.setValue === "function") {
          c1.setValue(v.טקסט);
        } else {
          c1.setText(v.טקסט);
        }
      }
      return;
    }
    if (name === "קבע_ערך") {
      var cv = controls[v.מזהה];
      if (cv && typeof cv.setValue === "function") {
        cv.setValue(v.ערך);
      } else if (cv) {
        cv.setText(v.ערך);
      }
      return;
    }
    if (name === "קבע_רמז") {
      var cp = controls[v.מזהה];
      if (cp && typeof cp.setPlaceholder === "function") {
        cp.setPlaceholder(v.רמז);
      }
      return;
    }
    if (name === "קבע_שורות") {
      var cr = controls[v.מזהה];
      if (cr && typeof cr.setRows === "function") {
        cr.setRows(v.שורות);
      }
      return;
    }
    if (name === "קבע_סגנון") {
      var cs = controls[v.מזהה];
      if (cs && typeof cs.setStyle === "function") {
        cs.setStyle(v.סגנון);
      }
      return;
    }
    if (name === "קבע_מסומן") {
      var ck = controls[v.מזהה];
      if (ck && typeof ck.setChecked === "function") {
        ck.setChecked(!!v.מסומן);
      }
      return;
    }
    if (name === "קבע_פריטים") {
      var ci = controls[v.מזהה];
      if (ci && typeof ci.setItems === "function") {
        ci.setItems(v.פריטים);
      }
      return;
    }
    if (name === "קבע_ילדים") {
      var ckids = controls[v.מזהה];
      if (ckids && typeof ckids.setChildren === "function") {
        ckids.setChildren(v.ערך, v.ילדים || v.פריטים || []);
      }
      return;
    }
    if (name === "קבע_גובה") {
      var ch = controls[v.מזהה];
      if (ch && typeof ch.setHeight === "function") {
        ch.setHeight(v.גובה);
      }
      return;
    }
    if (name === "קבע_עמודות") {
      var tc = controls[v.מזהה];
      if (tc && typeof tc.setColumns === "function") tc.setColumns(v.עמודות);
      return;
    }
    if (name === "קבע_שורות") {
      var tr = controls[v.מזהה];
      if (tr && typeof tr.setRows === "function") tr.setRows(v.שורות);
      return;
    }
    if (name === "הוסף_שורה") {
      var ar = controls[v.מזהה];
      if (ar && typeof ar.addRow === "function") ar.addRow(v.שורה);
      return;
    }
    if (name === "נקה_שורות") {
      var clr = controls[v.מזהה];
      if (clr && typeof clr.clear === "function") clr.clear();
      return;
    }
    if (name === "רשת_נושא") {
      var gt = controls[v.מזהה];
      if (gt && typeof gt.setGridTheme === "function") gt.setGridTheme(v.נושא);
      return;
    }
    if (name === "רשת_סנן") {
      var gf = controls[v.מזהה];
      if (gf && typeof gf.setFilter === "function") gf.setFilter(v.שדה, v.ערך);
      return;
    }
    if (name === "רשת_סנן_גלובלי") {
      var gfg = controls[v.מזהה];
      if (gfg && typeof gfg.setFilterGlobal === "function") gfg.setFilterGlobal(v.טקסט);
      return;
    }
    if (name === "רשת_נקה_סינון") {
      var gcf = controls[v.מזהה];
      if (gcf && typeof gcf.clearFilter === "function") gcf.clearFilter();
      return;
    }
    if (name === "רשת_קבץ") {
      var gg = controls[v.מזהה];
      if (gg && typeof gg.groupBy === "function") gg.groupBy(v.שדה);
      return;
    }
    if (name === "רשת_העתק") {
      var gcp = controls[v.מזהה];
      if (gcp && typeof gcp.copyGrid === "function") gcp.copyGrid();
      return;
    }
    if (name === "רשת_ייצא") {
      var gex = controls[v.מזהה];
      if (gex && typeof gex.exportGrid === "function") gex.exportGrid(v.פורמט, v.נתיב);
      return;
    }
    if (name === "רשת_עדכן_תא") {
      var guc = controls[v.מזהה];
      if (guc && typeof guc.updateCell === "function") guc.updateCell(v.מזהה_שורה, v.שדה, v.ערך);
      return;
    }
    if (name === "קבע_פרסומת") {
      var adc = controls[v.מזהה];
      if (adc && typeof adc.setAd === "function") adc.setAd(v);
      return;
    }
    if (name === "קבע_שם") {
      var an = controls[v.מזהה];
      if (an && typeof an.setName === "function") an.setName(v.שם);
      else if (an) an.setText(v.שם);
      return;
    }
    if (name === "קבע_תמונה") {
      var ai = controls[v.מזהה];
      if (ai && typeof ai.setImage === "function") ai.setImage(v.תמונה);
      return;
    }
    if (name === "קבע_גודל") {
      var asz = controls[v.מזהה];
      if (asz && typeof asz.setSize === "function") asz.setSize(v.גודל);
      return;
    }
    if (name === "קבע_תוכן") {
      var ct = controls[v.מזהה];
      if (ct && typeof ct.setContent === "function") {
        ct.setContent(v.תוכן);
      }
      return;
    }
    if (name === "קבע_יעד") {
      var cy = controls[v.מזהה];
      if (cy && typeof cy.setTarget === "function") {
        cy.setTarget(v.יעד);
      }
      return;
    }
    if (name === "קשר_הצגה") {
      var bid = v.מזהה;
      var tid = v.יעד == null ? "" : String(v.יעד);
      pendingShowTargets[bid] = tid;
      var bl = controls[bid];
      if (bl && typeof bl.setShowTarget === "function") {
        bl.setShowTarget(tid);
      }
      return;
    }
    if (name === "קבע_מקבל") {
      var fa = controls[v.מזהה];
      if (fa && typeof fa.setAccept === "function") fa.setAccept(v.מקבל);
      return;
    }
    if (name === "קבע_מרובה") {
      var fm = controls[v.מזהה];
      if (fm && typeof fm.setMultiple === "function") fm.setMultiple(!!v.מרובה);
      return;
    }
    if (name === "פתח") {
      var fo = controls[v.מזהה];
      if (fo && typeof fo.open === "function") fo.open();
      else if (fo && typeof fo.sample === "function") fo.sample();
      return;
    }
    if (name === "דגום") {
      var sm = controls[v.מזהה];
      if (sm && typeof sm.sample === "function") sm.sample();
      return;
    }
    if (name === "הבא") {
      var nx = controls[v.מזהה];
      if (nx && typeof nx.next === "function") nx.next();
      return;
    }
    if (name === "הקודם") {
      var pv = controls[v.מזהה];
      if (pv && typeof pv.prev === "function") pv.prev();
      return;
    }
    if (name === "הצג") {
      var sid = v.מזהה;
      var sh = controls[sid];
      if (sh && typeof sh.show === "function") {
        try {
          sh.show();
        } catch (e) {}
      } else {
        // ניסיון חוזר — לפעמים הפקודה מגיעה לפני mount ב־WebView2
        setTimeout(function () {
          var c = controls[sid];
          if (c && typeof c.show === "function") c.show();
        }, 50);
      }
      return;
    }
    if (name === "הסתר") {
      var hi = controls[v.מזהה];
      if (hi && typeof hi.hide === "function") hi.hide();
      return;
    }
    if (name === "קבע_כותרת") {
      var tit = controls[v.מזהה];
      if (tit && typeof tit.setTitle === "function") tit.setTitle(v.כותרת);
      return;
    }
    if (name === "קבע_גוף") {
      var bd = controls[v.מזהה];
      if (bd && typeof bd.setBody === "function") bd.setBody(v.גוף);
      return;
    }
    if (name === "קבע_כפתור_פעולה") {
      var act = controls[v.מזהה];
      if (act && typeof act.setActionLabel === "function") {
        act.setActionLabel(v.טקסט != null ? v.טקסט : v.כפתור_פעולה);
      }
      return;
    }
    if (name === "קבע_תאים") {
      var gc = controls[v.מזהה];
      if (gc && typeof gc.setCells === "function") gc.setCells(v.תאים);
      return;
    }
    if (name === "קבע_גלוי") {
      var c2 = controls[v.מזהה];
      if (c2) c2.setVisible(!!v.ערך);
      return;
    }
    if (name === "קבע_מושבת") {
      var c3 = controls[v.מזהה];
      if (c3) c3.setDisabled(!!v.ערך);
      return;
    }
    if (name === "מחק") {
      var c4 = controls[v.מזהה];
      if (c4) c4.remove();
      return;
    }
    if (name === "קבע_תרגום") {
      setI18nCatalog(v || {});
      return;
    }
    if (name === "הצג_טעינה") {
      if (typeof global.__יוד_הצג_טעינה === "function") {
        global.__יוד_הצג_טעינה(i18nT(v.טקסט || "טוען…"));
      }
      return;
    }
    if (name === "הסתר_טעינה") {
      if (typeof global.__יוד_הסתר_טעינה === "function") {
        global.__יוד_הסתר_טעינה();
      }
      return;
    }
    if (name === "נקה") {
      Object.keys(controls).forEach(function (id) {
        controls[id].remove();
      });
      var hint = document.getElementById("רמז_ריק");
      if (hint) hint.style.display = "";
      var r = root();
      if (r) r.classList.remove("has-controls");
      return;
    }
    if (name === "קבע_נושא") {
      var theme = v.נושא === "בהיר" || v.נושא === "light" ? "light" : "dark";
      document.documentElement.setAttribute("data-bs-theme", theme);
      return;
    }
    if (name === "קבע_כיוון") {
      var rtl = v.כיוון === "ימין_לשמאל" || v.כיוון === "rtl" || v.כיוון === "ימין";
      var langHint = v.שפה != null && String(v.שפה) !== "" ? String(v.שפה) : (rtl ? "he" : (i18n.lang || "en"));
      i18n.dir = rtl ? "rtl" : "ltr";
      if (langHint) i18n.lang = langHint;
      document.documentElement.setAttribute("lang", i18n.lang || (rtl ? "he" : "en"));
      applyDocumentDir(rtl ? "rtl" : "ltr");
      closeAllYodMenus();
      return;
    }
  }

  function enqueueOrRun(data) {
    if (!domReady) {
      pendingCommands.push(data);
      return;
    }
    applyCommand(data);
  }

  function flushQueue() {
    var q = pendingCommands.slice();
    pendingCommands = [];
    for (var i = 0; i < q.length; i++) {
      applyCommand(q[i]);
    }
  }

  function emitReady() {
    post({ סוג: "אירוע", שם: "מוכן", ערכים: {} });
  }

  function onReady() {
    if (domReady) return;
    domReady = true;
    flushQueue();
    // כשהחלון מוסתר עד splash — מחכים לחשיפה לפני בניית פקדי התוכנה
    if (global.__יוד_המתן_לחשיפה) {
      global.__יוד_שחרר_מוכן = emitReady;
    } else {
      emitReady();
    }
  }

  function boot() {
    global.__יוד_עיצוב = {
      post: post,
      controls: controls,
      applyCommand: applyCommand,
      הצג_טעינה: function (t) {
        if (typeof global.__יוד_הצג_טעינה === "function") global.__יוד_הצג_טעינה(t);
      },
      הסתר_טעינה: function () {
        if (typeof global.__יוד_הסתר_טעינה === "function") global.__יוד_הסתר_טעינה();
      }
    };
    global.__יוד_מנוע = global.__יוד_עיצוב;
    global.__יוד_קבל = function (data) {
      if (typeof data === "string") {
        try {
          data = JSON.parse(data);
        } catch (e) {
          return;
        }
      }
      enqueueOrRun(data);
    };
    // לא מאזינים ל־yod-message: browserSend קורא גם ל־__יוד_קבל וגם מפיץ את האירוע —
    // האזנה כפולה יוצרת כל פקד פעמיים ב־DOM.

    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", onReady);
    } else {
      onReady();
    }
  }

  boot();
})(typeof window !== "undefined" ? window : this);
