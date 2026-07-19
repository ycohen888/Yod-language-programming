const { app, BrowserWindow, ipcMain, dialog, Menu, shell, nativeTheme } = require("electron");
const path = require("node:path");
const fs = require("node:fs/promises");
const fssync = require("node:fs");
const { spawn } = require("node:child_process");
const { fileURLToPath } = require("node:url");

const isPackaged = app.isPackaged;
const appRoot = isPackaged ? path.join(__dirname, "..") : path.join(__dirname, "..");
const distIndex = path.join(appRoot, "dist", "index.html");
const useDevServer =
  !isPackaged && (process.env.YOD_IDE_DEV === "1" || !fssync.existsSync(distIndex));

const MAX_OPEN_BYTES = 8 * 1024 * 1024;

/** @type {BrowserWindow | null} */
let mainWindow = null;

// תפריטים ודיאלוגים במצב כהה
nativeTheme.themeSource = "dark";

function resolveYodExe() {
  const env = process.env.YOD_EXE;
  if (env && fssync.existsSync(env)) return env;
  const candidates = [
    path.join(path.dirname(process.execPath), "yod.exe"),
    path.join(process.resourcesPath || "", "..", "yod.exe"),
    // בפיתוח: מעדיפים את מנוע Go שבתיקיית yod/ כי הוא נבנה שם בעבודה שוטפת.
    path.join(__dirname, "..", "..", "yod", "yod.exe"),
    path.join(__dirname, "..", "..", "yod.exe"),
    path.join(__dirname, "..", "yod.exe"),
  ];
  for (const c of candidates) {
    if (c && fssync.existsSync(c)) return c;
  }
  return "yod";
}

function sendMenu(action) {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send("yod:menu", action);
  }
}

function resolveAppIcon() {
  const candidates = [
    path.join(process.resourcesPath || "", "icon.ico"),
    path.join(__dirname, "..", "build", "icon.ico"),
    path.join(appRoot, "build", "icon.ico"),
    path.join(__dirname, "..", "public", "icon.ico"),
    path.join(appRoot, "public", "icon.ico"),
    path.join(__dirname, "..", "..", "yod", "assets", "yod.ico"),
  ];
  for (const c of candidates) {
    if (c && fssync.existsSync(c)) return c;
  }
  return undefined;
}

function createWindow(openPath) {
  const icon = resolveAppIcon();
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 900,
    minHeight: 560,
    backgroundColor: "#1e1e1e",
    title: "יוד",
    show: false,
    autoHideMenuBar: true,
    ...(icon ? { icon } : {}),
    webPreferences: {
      preload: path.join(__dirname, "preload.cjs"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  // תפריט מותאם ב־HTML (כהה, בלי רדיוס) — בלי תפריט נייטיב בהיר
  Menu.setApplicationMenu(null);

  mainWindow.once("ready-to-show", () => {
    mainWindow?.show();
  });

  if (useDevServer) {
    mainWindow.loadURL("http://127.0.0.1:5173");
  } else {
    mainWindow.loadFile(distIndex);
  }

  mainWindow.webContents.on("did-finish-load", () => {
    if (openPath) {
      mainWindow?.webContents.send("yod:open-path", openPath);
    }
  });

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

app.whenReady().then(() => {
  if (process.platform === "win32") {
    app.setAppUserModelId("il.yod.ide");
    // קיצור Start Menu רק בפיתוח (דורש Node)
    if (!isPackaged) {
      try {
        const shortcutInstaller = path.join(__dirname, "..", "scripts", "install-shortcut.cjs");
        if (fssync.existsSync(shortcutInstaller)) {
          spawn("node", [shortcutInstaller], {
            cwd: path.join(__dirname, ".."),
            detached: true,
            stdio: "ignore",
            windowsHide: true,
          }).unref();
        }
      } catch {
        /* ignore */
      }
    }
  }
  try {
    app.setName("יוד");
  } catch {
    /* ignore */
  }
  const argPath = process.argv.find(
    (a, i) =>
      i > 0 &&
      !a.startsWith("-") &&
      a !== "." &&
      !a.includes("electron") &&
      a !== "עורך" &&
      !a.endsWith("Yod IDE.exe") &&
      !a.endsWith("YodIDE.exe")
  );
  createWindow(argPath && fssync.existsSync(argPath) ? argPath : "");
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

// ——— IPC ———

ipcMain.handle("fs:readDir", async (_e, dirPath) => {
  const entries = await fs.readdir(dirPath, { withFileTypes: true });
  const out = [];
  for (const ent of entries) {
    const name = ent.name;
    if (name === "." || name === "..") continue;
    if (name.startsWith(".") && name !== ".יוד_חבילות") continue;
    out.push({
      name,
      path: path.join(dirPath, name),
      isDir: ent.isDirectory(),
    });
  }
  out.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.localeCompare(b.name, "he");
  });
  return out;
});

ipcMain.handle("fs:readFile", async (_e, filePath) => {
  const st = await fs.stat(filePath);
  if (st.size > MAX_OPEN_BYTES) {
    throw new Error(
      `הקובץ גדול מדי לעורך (${Math.round(st.size / 1024 / 1024)}MB). מקסימום ${MAX_OPEN_BYTES / 1024 / 1024}MB.`
    );
  }
  const buf = await fs.readFile(filePath);
  const sample = buf.subarray(0, Math.min(8000, buf.length));
  let nul = 0;
  for (const b of sample) if (b === 0) nul++;
  if (nul > 0) throw new Error("קובץ בינארי — לא ניתן לפתוח בעורך הטקסט");
  return { text: buf.toString("utf8"), size: st.size, mtimeMs: st.mtimeMs };
});

ipcMain.handle("fs:writeFile", async (_e, filePath, text) => {
  await fs.writeFile(filePath, text, "utf8");
  return true;
});

ipcMain.handle("fs:mkdir", async (_e, dirPath) => {
  await fs.mkdir(dirPath, { recursive: true });
  return true;
});

ipcMain.handle("fs:rename", async (_e, from, to) => {
  await fs.rename(from, to);
  return true;
});

ipcMain.handle("fs:remove", async (_e, targetPath) => {
  const st = await fs.stat(targetPath);
  if (st.isDirectory()) await fs.rm(targetPath, { recursive: true, force: true });
  else await fs.unlink(targetPath);
  return true;
});

ipcMain.handle("fs:stat", async (_e, p) => {
  const st = await fs.stat(p);
  return { isDir: st.isDirectory(), size: st.size, mtimeMs: st.mtimeMs };
});

ipcMain.handle("fs:exists", async (_e, p) => {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
});

ipcMain.handle("dialog:openFile", async () => {
  const res = await dialog.showOpenDialog(mainWindow, {
    properties: ["openFile"],
    title: "פתיחת קובץ",
    filters: [
      { name: "קבצי יוד / טקסט", extensions: ["*"] },
      { name: "הכל", extensions: ["*"] },
    ],
  });
  if (res.canceled || !res.filePaths[0]) return null;
  return res.filePaths[0];
});

ipcMain.handle("dialog:openFolder", async () => {
  const res = await dialog.showOpenDialog(mainWindow, {
    properties: ["openDirectory"],
    title: "בחירת תיקיית פרויקט",
  });
  if (res.canceled || !res.filePaths[0]) return null;
  return res.filePaths[0];
});

ipcMain.handle("dialog:saveFile", async (_e, defaultPath) => {
  const res = await dialog.showSaveDialog(mainWindow, {
    title: "שמירת קובץ",
    defaultPath: defaultPath || "קובץ.יוד",
    filters: [
      { name: "יוד", extensions: ["יוד"] },
      { name: "הכל", extensions: ["*"] },
    ],
  });
  if (res.canceled || !res.filePath) return null;
  return res.filePath;
});

ipcMain.handle("dialog:prompt", async (_e, opts) => {
  // Electron אין prompt מובנה — משתמשים ב־messageBox + קלט דרך פשוט:
  // מחזירים null אם בוטל; לשינוי שם נשתמש ב־showMessageBox עם detail בלבד
  // ונפתח דיאלוג מותאם קל דרך input ב־renderer.
  // כאן: confirm בלבד אם kind=confirm
  if (opts && opts.kind === "confirm") {
    const res = await dialog.showMessageBox(mainWindow, {
      type: "question",
      buttons: ["אישור", "ביטול"],
      defaultId: 0,
      cancelId: 1,
      title: opts.title || "אישור",
      message: opts.message || "",
    });
    return res.response === 0;
  }
  if (opts && opts.kind === "info") {
    await dialog.showMessageBox(mainWindow, {
      type: "info",
      buttons: ["סגור"],
      title: opts.title || "יוד",
      message: opts.message || "",
      detail: opts.detail || undefined,
    });
    return true;
  }
  return null;
});

ipcMain.handle("shell:showItem", async (_e, p) => {
  shell.showItemInFolder(p);
});

ipcMain.handle("shell:openPath", async (_e, p) => {
  await shell.openPath(p);
});

ipcMain.handle("shell:openExternal", async (_e, url) => {
  if (typeof url !== "string" || !/^(https?:|mailto:)/i.test(url)) {
    return false;
  }
  await shell.openExternal(url);
  return true;
});

ipcMain.handle("clipboard:writeText", async (_e, text) => {
  const { clipboard } = require("electron");
  clipboard.writeText(String(text ?? ""));
  return true;
});

const GUIDE_DIR = "מדריך שפת יוד";
const GUIDE_INDEX = "מדריך שפת יוד.html";

/** @type {BrowserWindow | null} */
let guideWindow = null;

function resolveGuideHtml() {
  const candidates = [
    path.join(path.dirname(process.execPath), GUIDE_DIR, GUIDE_INDEX),
    path.join(process.resourcesPath || "", GUIDE_DIR, GUIDE_INDEX),
    path.join(__dirname, "..", "..", GUIDE_DIR, GUIDE_INDEX),
    path.join(__dirname, "..", GUIDE_DIR, GUIDE_INDEX),
    path.join(app.getAppPath(), "..", GUIDE_DIR, GUIDE_INDEX),
  ];
  for (const c of candidates) {
    if (c && fssync.existsSync(c)) return c;
  }
  return "";
}

function openGuideWindow() {
  const html = resolveGuideHtml();
  if (!html) {
    return { ok: false, error: "לא נמצא תיקיית המדריך ליד העורך." };
  }
  if (guideWindow && !guideWindow.isDestroyed()) {
    guideWindow.focus();
    return { ok: true, path: html };
  }

  const guideRoot = path.dirname(html);
  const icon = resolveAppIcon();
  guideWindow = new BrowserWindow({
    width: 1120,
    height: 820,
    minWidth: 720,
    minHeight: 480,
    backgroundColor: "#0f1419",
    title: "מדריך שפת יוד",
    autoHideMenuBar: true,
    show: false,
    ...(icon ? { icon } : {}),
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  guideWindow.once("ready-to-show", () => {
    guideWindow?.show();
  });

  guideWindow.on("closed", () => {
    guideWindow = null;
  });

  // ניווט רק בתוך תיקיית המדריך (קובץ מקומי)
  guideWindow.webContents.on("will-navigate", (event, url) => {
    try {
      if (!url.startsWith("file:")) {
        event.preventDefault();
        return;
      }
      const dest = path.normalize(fileURLToPath(url));
      const rootNorm = path.normalize(guideRoot);
      const destLow = dest.toLowerCase();
      const rootLow = rootNorm.toLowerCase();
      const prefix = rootLow.endsWith(path.sep) ? rootLow : rootLow + path.sep;
      if (destLow !== rootLow && !destLow.startsWith(prefix)) {
        event.preventDefault();
      }
    } catch {
      event.preventDefault();
    }
  });

  guideWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith("http:") || url.startsWith("https:")) {
      shell.openExternal(url);
    }
    return { action: "deny" };
  });

  void guideWindow.loadFile(html);
  return { ok: true, path: html };
}

ipcMain.handle("guide:open", async () => openGuideWindow());

ipcMain.handle("app:getPaths", async () => {
  const ideRoot = path.join(__dirname, "..");
  const repoRoot = path.join(ideRoot, "..");
  return {
    yodExe: resolveYodExe(),
    userData: app.getPath("userData"),
    version: app.getVersion(),
    ideRoot,
    guideHtml: resolveGuideHtml() || path.join(repoRoot, GUIDE_DIR, GUIDE_INDEX),
  };
});

ipcMain.handle("app:quit", async () => {
  app.quit();
  return true;
});

ipcMain.handle("yod:run", async (_e, args, cwd) => {
  const exe = resolveYodExe();
  return await runProcess(exe, args, cwd || path.dirname(exe));
});

function runProcess(exe, args, cwd) {
  return new Promise((resolve) => {
    const child = spawn(exe, args, {
      cwd,
      windowsHide: true,
      env: { ...process.env, PYTHONIOENCODING: "utf-8" },
    });
    let stdout = "";
    let stderr = "";
    child.stdout?.on("data", (d) => {
      stdout += d.toString("utf8");
    });
    child.stderr?.on("data", (d) => {
      stderr += d.toString("utf8");
    });
    child.on("error", (err) => {
      resolve({ code: -1, stdout, stderr: err.message, exe, args });
    });
    child.on("close", (code) => {
      resolve({ code: code ?? -1, stdout, stderr, exe, args });
    });
  });
}
