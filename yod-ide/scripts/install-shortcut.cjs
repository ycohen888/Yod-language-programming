/** קיצור Start Menu — מפעיל ישירות את Electron (בלי yod.exe/CMD). */
const path = require("node:path");
const fs = require("node:fs");
const { spawnSync } = require("node:child_process");

const ideRoot = path.join(__dirname, "..");
const repoRoot = path.join(ideRoot, "..");
const iconIco = path.join(ideRoot, "build", "icon.ico");
const ps1 = path.join(__dirname, "install-start-menu-shortcut.ps1");

function resolveElectronYod() {
  const candidates = [
    path.join(ideRoot, "node_modules", "electron", "dist", "יוד.exe"),
    path.join(ideRoot, "build", "יוד.exe"),
  ];
  for (const c of candidates) {
    if (fs.existsSync(c)) return c;
  }
  return "";
}

const exe = resolveElectronYod();
if (!exe) {
  console.warn("אין יוד.exe של Electron — דילוג על קיצור Start Menu");
  process.exit(0);
}

const r = spawnSync(
  "powershell.exe",
  [
    "-NoProfile",
    "-ExecutionPolicy",
    "Bypass",
    "-WindowStyle",
    "Hidden",
    "-File",
    ps1,
    "-YodExe",
    exe,
    "-WorkDir",
    ideRoot,
    "-IconIco",
    fs.existsSync(iconIco) ? iconIco : path.join(repoRoot, "yod.ico"),
    "-AppId",
    "il.yod.ide",
    "-Arguments",
    ".",
  ],
  { encoding: "utf8", windowsHide: true }
);
if (r.stdout) process.stdout.write(r.stdout);
if (r.stderr) process.stderr.write(r.stderr);
process.exit(0);
