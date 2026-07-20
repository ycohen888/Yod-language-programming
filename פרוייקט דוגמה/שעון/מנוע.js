(() => {
  const KEY = "yod-clock-settings-v2";

  const THEMES = [
    { id: "neon", label: "ניאון" },
    { id: "lcd", label: "LCD ירוק" },
    { id: "minimal", label: "מינימלי" },
    { id: "sunset", label: "שקיעה" },
  ];

  const defaults = {
    theme: "neon",
    format: "24",
    seconds: true,
    date: true,
    topmost: true,
  };

  const app = document.getElementById("app");
  const clockView = document.getElementById("clock-view");
  const settingsView = document.getElementById("settings-view");
  const digTime = document.getElementById("dig-time");
  const digDate = document.getElementById("dig-date");
  const menu = document.getElementById("menu");
  const clockWrap = document.getElementById("clock-wrap");
  const setTheme = document.getElementById("set-theme");
  const setFormat = document.getElementById("set-format");
  const setSeconds = document.getElementById("set-seconds");
  const setDate = document.getElementById("set-date");
  const setTopmost = document.getElementById("set-topmost");
  const btnTopmost = document.getElementById("btn-topmost");

  let cfg = load();
  let inSettings = false;

  function load() {
    try {
      const raw = localStorage.getItem(KEY);
      if (!raw) return { ...defaults };
      const parsed = { ...defaults, ...JSON.parse(raw) };
      if (!THEMES.some((t) => t.id === parsed.theme)) parsed.theme = defaults.theme;
      return parsed;
    } catch {
      return { ...defaults };
    }
  }

  function save() {
    try {
      localStorage.setItem(KEY, JSON.stringify(cfg));
    } catch {}
  }

  function post(msg) {
    const s = String(msg);
    try {
      if (window.chrome && chrome.webview && chrome.webview.postMessage) {
        chrome.webview.postMessage(s);
      }
    } catch {}
    try {
      if (window.__יוד_גשר) {
        fetch(window.__יוד_גשר, {
          method: "POST",
          body: s,
          mode: "no-cors",
          keepalive: true,
        }).catch(() => {});
        const img = new Image();
        img.src = window.__יוד_גשר + "?" + encodeURIComponent(s) + "&_=" + Date.now();
      }
    } catch {}
  }

  function exitApp() {
    post("יציאה");
    setTimeout(() => post("יציאה"), 80);
    setTimeout(() => post("יציאה"), 250);
  }

  function fitWindow() {
    if (inSettings) {
      post("שקיפות:שקר");
      post("צורה:מעוגל");
      post("גודל:320,400");
      return;
    }
    post("שקיפות:שקר");
    post("גודל:300,170");
    setTimeout(() => post("צורה:מעוגל"), 40);
  }

  function fillThemes() {
    setTheme.innerHTML = "";
    for (const t of THEMES) {
      const opt = document.createElement("option");
      opt.value = t.id;
      opt.textContent = t.label;
      setTheme.appendChild(opt);
    }
    setTheme.value = cfg.theme;
  }

  function applyUi() {
    app.dataset.theme = cfg.theme;
    app.dataset.format = cfg.format;
    app.classList.toggle("no-seconds", !cfg.seconds);
    app.classList.toggle("no-date", !cfg.date);
    setFormat.value = cfg.format;
    setSeconds.checked = !!cfg.seconds;
    setDate.checked = !!cfg.date;
    setTopmost.checked = !!cfg.topmost;
    btnTopmost.textContent = cfg.topmost ? "תמיד מעל ✓" : "תמיד מעל";
    fillThemes();
  }

  function hebDate(d) {
    try {
      return new Intl.DateTimeFormat("he-IL", {
        weekday: "short",
        day: "numeric",
        month: "short",
      }).format(d);
    } catch {
      return d.toLocaleDateString("he-IL");
    }
  }

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function formatDigital(d) {
    let h = d.getHours();
    const m = pad(d.getMinutes());
    const s = pad(d.getSeconds());
    let suffix = "";
    if (cfg.format === "12") {
      suffix = h >= 12 ? " PM" : " AM";
      h = h % 12;
      if (h === 0) h = 12;
    }
    const hh = pad(h);
    return cfg.seconds ? `${hh}:${m}:${s}${suffix}` : `${hh}:${m}${suffix}`;
  }

  function tick() {
    const now = new Date();
    digTime.textContent = formatDigital(now);
    digDate.textContent = hebDate(now);
  }

  function openMenu(x, y) {
    menu.hidden = false;
    const padPx = 6;
    const w = menu.offsetWidth || 150;
    const h = menu.offsetHeight || 110;
    let left = Math.min(x, window.innerWidth - w - padPx);
    let top = Math.min(y, window.innerHeight - h - padPx);
    menu.style.left = Math.max(padPx, left) + "px";
    menu.style.top = Math.max(padPx, top) + "px";
  }

  function closeMenu() {
    menu.hidden = true;
  }

  function openSettings() {
    closeMenu();
    inSettings = true;
    app.classList.add("settings-open");
    applyUi();
    clockView.hidden = true;
    settingsView.hidden = false;
    fitWindow();
  }

  function closeSettings() {
    inSettings = false;
    app.classList.remove("settings-open");
    settingsView.hidden = true;
    clockView.hidden = false;
    applyUi();
    fitWindow();
  }

  function setTopmostFlag(on) {
    cfg.topmost = !!on;
    post(cfg.topmost ? "תמיד_מעל:אמת" : "תמיד_מעל:שקר");
    save();
    applyUi();
  }

  clockWrap.addEventListener("pointerdown", (e) => {
    if (e.button !== 0) return;
    if (e.target.closest("button")) return;
    if (inSettings) return;
    closeMenu();
    post("גרור");
  });

  document.querySelectorAll("[data-gear]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      openSettings();
    });
  });
  document.querySelectorAll("[data-exit]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      exitApp();
    });
  });
  document.getElementById("btn-close-settings").addEventListener("click", closeSettings);
  document.getElementById("btn-back").addEventListener("click", closeSettings);
  document.getElementById("btn-exit").addEventListener("click", exitApp);

  clockWrap.addEventListener("contextmenu", (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (inSettings) return;
    openMenu(e.clientX, e.clientY);
  });
  window.addEventListener("contextmenu", (e) => {
    e.preventDefault();
  });

  document.addEventListener(
    "pointerdown",
    (e) => {
      if (!menu.hidden && !menu.contains(e.target)) closeMenu();
    },
    true
  );

  menu.addEventListener("click", (e) => {
    const btn = e.target.closest("button[data-act]");
    if (!btn) return;
    e.preventDefault();
    e.stopPropagation();
    const act = btn.getAttribute("data-act");
    if (act === "exit") exitApp();
    else if (act === "settings") openSettings();
    else if (act === "topmost") {
      setTopmostFlag(!cfg.topmost);
      closeMenu();
    }
  });

  setTheme.addEventListener("change", () => {
    cfg.theme = setTheme.value;
    save();
    applyUi();
  });
  setFormat.addEventListener("change", () => {
    cfg.format = setFormat.value;
    save();
    applyUi();
    tick();
  });
  setSeconds.addEventListener("change", () => {
    cfg.seconds = setSeconds.checked;
    save();
    applyUi();
    tick();
  });
  setDate.addEventListener("change", () => {
    cfg.date = setDate.checked;
    save();
    applyUi();
  });
  setTopmost.addEventListener("change", () => {
    setTopmostFlag(setTopmost.checked);
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if (inSettings) closeSettings();
      else if (!menu.hidden) closeMenu();
      else exitApp();
    }
  });

  applyUi();
  tick();
  setInterval(tick, 50);
  fitWindow();
  post(cfg.topmost ? "תמיד_מעל:אמת" : "תמיד_מעל:שקר");
})();
