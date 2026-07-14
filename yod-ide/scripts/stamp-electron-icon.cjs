/**
 * יוצר node_modules/electron/dist/יוד.exe עם איקון ומטא־דאטה של יוד.
 * פס המשימות ב־Windows מציג את איקון ה־EXE — לכן חייבים EXE נפרד, לא רק BrowserWindow.icon.
 */
const path = require("node:path");
const fs = require("node:fs");

const root = path.join(__dirname, "..");
const electronDist = path.join(root, "node_modules", "electron", "dist");
const electronExe = path.join(electronDist, "electron.exe");
const yodExe = path.join(electronDist, "יוד.exe");
const iconIco = path.join(root, "build", "icon.ico");
const stampMeta = path.join(root, "build", ".icon-stamp");

function die(msg) {
  console.error(msg);
  process.exit(1);
}

if (!fs.existsSync(electronExe)) {
  die("חסר electron.exe — הריצו npm install ב־yod-ide");
}
if (!fs.existsSync(iconIco)) {
  die("חסר build/icon.ico");
}

fs.mkdirSync(path.join(root, "build"), { recursive: true });

const stampKey = [
  fs.statSync(iconIco).mtimeMs,
  fs.statSync(iconIco).size,
  fs.statSync(electronExe).size,
  fs.statSync(electronExe).mtimeMs,
].join("|");

if (fs.existsSync(yodExe) && fs.existsSync(stampMeta) && fs.readFileSync(stampMeta, "utf8") === stampKey) {
  console.log("יוד.exe כבר מעודכן עם האיקון");
  process.exit(0);
}

fs.copyFileSync(electronExe, yodExe);

async function main() {
  let rcedit;
  try {
    rcedit = require("rcedit");
  } catch {
    die("חסר rcedit — הריצו: npm i -D rcedit");
  }
  if (typeof rcedit === "object" && rcedit.rcedit) {
    rcedit = rcedit.rcedit;
  }

  await rcedit(yodExe, {
    icon: iconIco,
    "version-string": {
      CompanyName: "יוד",
      FileDescription: "עורך יוד",
      ProductName: "יוד",
      InternalName: "yod-ide",
      OriginalFilename: "יוד.exe",
      LegalCopyright: "יוד",
    },
    "file-version": "0.77.0",
    "product-version": "0.77.0",
  });

  // עותק גם ב־build לפתיחה ידנית / קיצורי דרך
  fs.copyFileSync(yodExe, path.join(root, "build", "יוד.exe"));
  fs.writeFileSync(stampMeta, stampKey, "utf8");
  console.log("נוצר:", yodExe);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
