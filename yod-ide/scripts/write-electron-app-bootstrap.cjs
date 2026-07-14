/** משאב Electron: אפשר להפעיל יוד.exe גם בלי ארגומנטים (הצמדה לפס משימות). */
const path = require("node:path");
const fs = require("node:fs");

const root = path.join(__dirname, ".."); // yod-ide
const appDir = path.join(root, "node_modules", "electron", "dist", "resources", "app");

fs.mkdirSync(appDir, { recursive: true });

const boot = `// נוצר אוטומטית — לא לערוך ידנית
const path = require("path");
const fs = require("fs");
const ideRoot = path.resolve(__dirname, "..", "..", "..", "..", "..");
const main = path.join(ideRoot, "electron", "main.cjs");
if (!fs.existsSync(main)) {
  console.error("לא נמצא עורך יוד:", main);
  process.exit(1);
}
process.chdir(ideRoot);
require(main);
`;

fs.writeFileSync(path.join(appDir, "boot.cjs"), boot, "utf8");
fs.writeFileSync(
  path.join(appDir, "package.json"),
  JSON.stringify(
    {
      name: "yod-ide",
      productName: "יוד",
      version: "0.77.0",
      main: "boot.cjs",
    },
    null,
    2
  ),
  "utf8"
);

console.log("resources/app מוכן להפעלה בלי ארגומנטים");
