/** מפעיל את build של Electron בשם יוד.exe (עם איקון). */
const path = require("node:path");
const fs = require("node:fs");
const { spawn } = require("node:child_process");

const root = path.join(__dirname, "..");
const yodExe = path.join(root, "node_modules", "electron", "dist", "יוד.exe");
const electronExe = path.join(root, "node_modules", "electron", "dist", "electron.exe");
const exe = fs.existsSync(yodExe) ? yodExe : electronExe;

const child = spawn(exe, ["."], {
  cwd: root,
  stdio: "inherit",
  env: { ...process.env, ELECTRON_NO_ATTACH_CONSOLE: "1" },
  windowsHide: false,
});
child.on("exit", (code) => process.exit(code ?? 0));
