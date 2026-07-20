(() => {
  const KEY = "yod-weather-settings-v1";

  const CITIES = [
    { id: "telaviv", name: "תל אביב", lat: 32.0853, lon: 34.7818 },
    { id: "jerusalem", name: "ירושלים", lat: 31.7683, lon: 35.2137 },
    { id: "haifa", name: "חיפה", lat: 32.794, lon: 34.9896 },
    { id: "tiberias", name: "טבריה", lat: 32.7922, lon: 35.5312 },
    { id: "eilat", name: "אילת", lat: 29.5577, lon: 34.9519 },
    { id: "beer", name: "באר שבע", lat: 31.253, lon: 34.7915 },
    { id: "netanya", name: "נתניה", lat: 32.3215, lon: 34.8532 },
    { id: "ashdod", name: "אשדוד", lat: 31.8044, lon: 34.6553 },
    { id: "rishon", name: "ראשון לציון", lat: 31.973, lon: 34.7925 },
    { id: "petah", name: "פתח תקווה", lat: 32.084, lon: 34.8878 },
    { id: "herzliya", name: "הרצליה", lat: 32.1624, lon: 34.8447 },
    { id: "ashkelon", name: "אשקלון", lat: 31.6688, lon: 34.5743 },
  ];

  const defaults = {
    city: "telaviv",
    unit: "c",
    topmost: true,
    autoRefresh: true,
  };

  const app = document.getElementById("app");
  const weatherView = document.getElementById("weather-view");
  const settingsView = document.getElementById("settings-view");
  const panel = document.getElementById("panel");
  const menu = document.getElementById("menu");
  const setCity = document.getElementById("set-city");
  const setUnit = document.getElementById("set-unit");
  const setTopmost = document.getElementById("set-topmost");
  const setRefresh = document.getElementById("set-refresh");
  const btnTopmost = document.getElementById("btn-topmost");

  let cfg = load();
  let inSettings = false;
  let refreshTimer = null;
  let lastData = null;

  function load() {
    try {
      const raw = localStorage.getItem(KEY);
      if (!raw) return { ...defaults };
      const parsed = { ...defaults, ...JSON.parse(raw) };
      if (!CITIES.some((c) => c.id === parsed.city)) parsed.city = defaults.city;
      if (parsed.unit !== "c" && parsed.unit !== "f") parsed.unit = "c";
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

  function cityById(id) {
    return CITIES.find((c) => c.id === id) || CITIES[0];
  }

  function fillCities() {
    setCity.innerHTML = "";
    for (const c of CITIES) {
      const opt = document.createElement("option");
      opt.value = c.id;
      opt.textContent = c.name;
      setCity.appendChild(opt);
    }
    setCity.value = cfg.city;
  }

  function applyUi() {
    app.dataset.unit = cfg.unit;
    setUnit.value = cfg.unit;
    setTopmost.checked = !!cfg.topmost;
    setRefresh.checked = !!cfg.autoRefresh;
    btnTopmost.textContent = cfg.topmost ? "תמיד מעל ✓" : "תמיד מעל";
    fillCities();
    document.getElementById("city-name").textContent = cityById(cfg.city).name;
  }

  function toDisplayTemp(celsius) {
    if (cfg.unit === "f") return Math.round((celsius * 9) / 5 + 32);
    return Math.round(celsius);
  }

  function unitSym() {
    return cfg.unit === "f" ? "°F" : "°C";
  }

  function windDirHe(deg) {
    const dirs = ["צפון", "צפ׳־מז׳", "מזרח", "דד׳־מז׳", "דרום", "דד׳־מע׳", "מערב", "צפ׳־מע׳"];
    const i = Math.round((((deg % 360) + 360) % 360) / 45) % 8;
    return dirs[i];
  }

  function codeInfo(code) {
    // WMO Weather interpretation codes
    if (code === 0) return { icon: "☀️", text: "שמיים בהירים" };
    if (code === 1) return { icon: "🌤️", text: "בעיקר בהיר" };
    if (code === 2) return { icon: "⛅", text: "מעונן חלקית" };
    if (code === 3) return { icon: "☁️", text: "מעונן" };
    if (code === 45 || code === 48) return { icon: "🌫️", text: "ערפל" };
    if (code >= 51 && code <= 57) return { icon: "🌦️", text: "טפטוף" };
    if (code >= 61 && code <= 67) return { icon: "🌧️", text: "גשם" };
    if (code >= 71 && code <= 77) return { icon: "🌨️", text: "שלג" };
    if (code >= 80 && code <= 82) return { icon: "🌧️", text: "ממטרים" };
    if (code >= 85 && code <= 86) return { icon: "🌨️", text: "ממטרי שלג" };
    if (code >= 95) return { icon: "⛈️", text: "סופה / רעמים" };
    return { icon: "🌡️", text: "מזג אוויר" };
  }

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function formatClock(iso) {
    try {
      const d = new Date(iso);
      return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
    } catch {
      return "--:--";
    }
  }

  function render(data) {
    lastData = data;
    const cur = data.current;
    const daily = data.daily;
    const info = codeInfo(cur.weather_code);
    const sym = unitSym();

    document.getElementById("wx-icon").textContent = info.icon;
    document.getElementById("temp").textContent = `${toDisplayTemp(cur.temperature_2m)}${sym}`;
    document.getElementById("feel").textContent = `מרגיש כמו ${toDisplayTemp(cur.apparent_temperature)}${sym}`;
    document.getElementById("desc").textContent = info.text;
    document.getElementById("wind").textContent =
      `${Math.round(cur.wind_speed_10m)} קמ״ש · ${windDirHe(cur.wind_direction_10m)}`;
    document.getElementById("humidity").textContent = `${Math.round(cur.relative_humidity_2m)}%`;
    const rainMax = daily.precipitation_probability_max?.[0];
    document.getElementById("rain-chance").textContent =
      rainMax == null ? "—" : `${Math.round(rainMax)}%`;
    document.getElementById("minmax").textContent =
      `${toDisplayTemp(daily.temperature_2m_min[0])}° / ${toDisplayTemp(daily.temperature_2m_max[0])}°`;
    document.getElementById("sun").textContent =
      `זריחה ${formatClock(daily.sunrise[0])} · שקיעה ${formatClock(daily.sunset[0])}`;
    document.getElementById("updated").textContent =
      `עודכן ${formatClock(cur.time)} · ${cityById(cfg.city).name}`;

    const hoursEl = document.getElementById("hours");
    hoursEl.innerHTML = "";
    const now = new Date();
    const times = data.hourly.time || [];
    let shown = 0;
    for (let i = 0; i < times.length && shown < 12; i++) {
      const t = new Date(times[i]);
      if (t.getTime() + 30 * 60 * 1000 < now.getTime()) continue;
      const code = data.hourly.weather_code[i];
      const temp = data.hourly.temperature_2m[i];
      const rain = data.hourly.precipitation_probability[i];
      const hi = codeInfo(code);
      const div = document.createElement("div");
      div.className = "hour";
      div.innerHTML =
        `<div class="h-time">${pad(t.getHours())}:00</div>` +
        `<div class="h-icon">${hi.icon}</div>` +
        `<div class="h-temp">${toDisplayTemp(temp)}°</div>` +
        `<div class="h-rain">${rain == null ? "" : rain + "%"}</div>`;
      hoursEl.appendChild(div);
      shown++;
    }
    if (!shown) {
      hoursEl.innerHTML = `<div class="hour"><div class="h-temp">—</div></div>`;
    }
  }

  async function fetchWeather() {
    const city = cityById(cfg.city);
    document.getElementById("city-name").textContent = city.name;
    app.classList.add("loading");
    app.classList.remove("error");
    document.getElementById("updated").textContent = "טוען…";
    document.getElementById("desc").textContent = "מעדכן נתונים";

    const url =
      "https://api.open-meteo.com/v1/forecast?" +
      new URLSearchParams({
        latitude: String(city.lat),
        longitude: String(city.lon),
        timezone: "Asia/Jerusalem",
        forecast_days: "1",
        wind_speed_unit: "kmh",
        current:
          "temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m,wind_direction_10m",
        hourly: "temperature_2m,precipitation_probability,weather_code",
        daily:
          "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunrise,sunset",
      }).toString();

    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json();
      if (!data.current) throw new Error("תשובה לא תקינה");
      render(data);
      app.classList.remove("loading");
    } catch (err) {
      app.classList.remove("loading");
      app.classList.add("error");
      document.getElementById("desc").textContent = "לא ניתן לטעון מזג אוויר (בדקו רשת)";
      document.getElementById("updated").textContent = "שגיאה בטעינה";
    }
  }

  function scheduleRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }
    if (cfg.autoRefresh) {
      refreshTimer = setInterval(fetchWeather, 15 * 60 * 1000);
    }
  }

  function openMenu(x, y) {
    menu.hidden = false;
    const padPx = 6;
    const w = menu.offsetWidth || 150;
    const h = menu.offsetHeight || 140;
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
    applyUi();
    weatherView.hidden = true;
    settingsView.hidden = false;
  }

  function closeSettings() {
    inSettings = false;
    settingsView.hidden = true;
    weatherView.hidden = false;
    applyUi();
    fetchWeather();
    scheduleRefresh();
  }

  function setTopmostFlag(on) {
    cfg.topmost = !!on;
    post(cfg.topmost ? "תמיד_מעל:אמת" : "תמיד_מעל:שקר");
    save();
    applyUi();
  }

  document.querySelectorAll("[data-gear]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      openSettings();
    });
  });
  document.getElementById("btn-close-settings").addEventListener("click", closeSettings);
  document.getElementById("btn-back").addEventListener("click", closeSettings);
  document.getElementById("btn-exit").addEventListener("click", exitApp);
  document.getElementById("btn-refresh").addEventListener("click", (e) => {
    e.stopPropagation();
    fetchWeather();
  });

  panel.addEventListener("contextmenu", (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (inSettings) return;
    openMenu(e.clientX, e.clientY);
  });
  window.addEventListener("contextmenu", (e) => e.preventDefault());

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
    else if (act === "refresh") {
      closeMenu();
      fetchWeather();
    } else if (act === "topmost") {
      setTopmostFlag(!cfg.topmost);
      closeMenu();
    }
  });

  setCity.addEventListener("change", () => {
    cfg.city = setCity.value;
    save();
  });
  setUnit.addEventListener("change", () => {
    cfg.unit = setUnit.value;
    save();
    if (lastData) render(lastData);
  });
  setTopmost.addEventListener("change", () => setTopmostFlag(setTopmost.checked));
  setRefresh.addEventListener("change", () => {
    cfg.autoRefresh = setRefresh.checked;
    save();
    scheduleRefresh();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if (inSettings) closeSettings();
      else if (!menu.hidden) closeMenu();
    }
  });

  applyUi();
  post(cfg.topmost ? "תמיד_מעל:אמת" : "תמיד_מעל:שקר");
  fetchWeather();
  scheduleRefresh();
})();
