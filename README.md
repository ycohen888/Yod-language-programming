# יוד (Yod) — שפת תכנות בעברית

**יוד** היא שפת תכנות מודרנית בעברית: תחביר מימין־לשמאל, ספריות מובנות, עורך בסגנון VS Code, ומכונה וירטואלית.

גרסה נוכחית: **0.77.0**

<p align="center">
  <img src="docs/screenshots/editor-code.png" alt="עורך יוד — קוד בעברית" width="720" />
</p>

<p align="center">
  <img src="docs/screenshots/editor-autocomplete.png" alt="השלמה אוטומטית בעורך" width="360" />
  &nbsp;
  <img src="docs/screenshots/server-gui-browser.png" alt="שרת יוד — GUI ודפדפן" width="360" />
</p>

<p align="center">
  <img src="docs/screenshots/editor-error.png" alt="עורך יוד — הצגת שגיאת קומפילציה" width="720" />
</p>

---

## מה יש בריפו

| נתיב | תוכן |
|------|------|
| [`yod/`](yod/) | קוד המקור של השפה, ה־CLI, הספריות והעורך הישן (Win32) |
| [`yod-ide/`](yod-ide/) | **עורך יוד החדש** — Electron + React + CodeMirror 6 |
| [`פרוייקט דוגמה/`](פרוייקט%20דוגמה/) | פרויקטי הדגמה (סנייק, מכרות, צייר, גרפים, רכיבים…) |
| [`מדריך שפת יוד/`](מדריך%20שפת%20יוד/) | מדריך HTML בעברית (תחביר + ספריות) |
| [`docs/screenshots/`](docs/screenshots/) | צילומי מסך של העורך והשרת |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | איך לבנות, לבדוק ולפתוח PR |
| [`info_program.html`](info_program.html) | יומן פיתוח וסטטוס פיצ׳רים |
| [`.cursorrules`](.cursorrules) | כללי עבודה לפיתוח השפה |

---

## התקנה ובנייה

### אפשרות א׳ — הורדה מוכנה (Windows)

ב־[Releases](https://github.com/ycohen888/Yod-language-programming/releases) יש `yod.exe` מוכן להורדה.
הגרסה האחרונה: **[v0.77.0](https://github.com/ycohen888/Yod-language-programming/releases/tag/v0.77.0)**.

להפעלת **העורך החדש** אחרי clone/הורדת הריפו:

```powershell
cd yod-ide
npm install
npm run build
cd ..
.\yod.exe עורך
```

### אפשרות ב׳ — בנייה מקוד המקור

דרוש [Go](https://go.dev/dl/) 1.22+ ו־[Node.js](https://nodejs.org/) (לעורך החדש).

```powershell
cd yod
go build -o ..\yod.exe ./cmd/yod

cd ..\yod-ide
npm install
npm run build
```

הרצה:

```powershell
.\yod.exe              # פותח את העורך (Electron אם נבנה, אחרת Win32)
.\yod.exe גרסה
.\yod.exe הרץ "פרוייקט דוגמה\סנייק"
.\yod.exe עזרה
```

עורך ישן (Win32):

```powershell
$env:YOD_LEGACY_EDITOR=1
.\yod.exe עורך
```

---

## התחלה מהירה

קובץ `שלום.יוד`:

```יוד
הדפס("שלום עולם")

משתנה שם = קלט("מה שמך? ")
הדפס("היי, " + שם)
```

```powershell
.\yod.exe הרץ שלום.יוד
```

עוד דוגמאות ב־[`yod/examples/`](yod/examples/) וב־[`פרוייקט דוגמה/`](פרוייקט%20דוגמה/).

### פרויקט (כמה קבצים)

הקובץ הראשי הוא תמיד `התחל.יוד`. שאר הקבצים נכללים עם `כלול` / `יבא`.
מדריך: [קובץ יחיד או פרויקט](מדריך%20שפת%20יוד/עמודים/קובץ-או-פרויקט.html).

```powershell
.\yod.exe הרץ "פרוייקט דוגמה\גרפים"
# או: פתח תיקייה בעורך → F5 מריץ את התחל.יוד
```

---

## יכולות מרכזיות

### שפה
- משתנים, תנאים (`אם` / `אחרת`), לולאות (`כל_עוד`, `עבור`)
- פונקציות, מחלקות, ירושה (`מרחיב`), `פרטי` / `ציבורי`
- `נסה` / `תפוס` / `זרוק`, `בחר` / `מקרה`
- מתודות עשירות על מחרוזת, מספר, רשימה ומילון
- הרצה במפרש או במכונה וירטואלית (`הרץ --מכונה`)
- סגנון רשמי כמו PHP: `()` לתנאים ופרמטרים, `{}` לבלוקים (גם `סוף` עדיין נתמך)

### ספריות מובנות (`כלול "..."`)
| ספרייה | תפקיד |
|--------|--------|
| `בסיס` | הדפס, קלט, המרות, אקראי… |
| `קבצים` | קריאה/כתיבה, העתקה, ארכיונים |
| `JSON` | פרסור וסריאליזציה |
| `זמן` | תאריך, חותמת, פרסור, הפרשים |
| `מתמטיקה` | פי, סינוס, חזקה… |
| `מערכת` | סביבה, זיכרון, תהליכים, כוננים |
| `רשת` | HTTP בעברית + שרת |
| `SQL` | SQLite, MySQL, PostgreSQL |
| `חלונות` | GUI ל־Windows (כולל טבלה וגרף) |
| `גרפים` | עמודות, קו, עוגה מקצועיים |
| `ציור` | לוח, צורות, שמירת תמונות |
| `הצפנה` | Base64, MD5, SHA-256, HMAC |
| `לוח` | clipboard (Windows) |
| `מספרים` | פסיקים, כסף, בתים, אחוזים לתצוגה |
| `עכבר` | מיקום ולחיצות עכבר |

### עורך יוד (חדש — `yod-ide/`)
- Electron + React + CodeMirror 6, ממשק כהה בסגנון VS Code
- RTL מלא לעברית, צביעת תחביר PHP Dark+
- סייר קבצים, ניתוח קובץ, סמלי פרויקט
- F12 / Ctrl+לחיצה להגדרה, Shift+F12 להפניות, Ctrl+P / Ctrl+Shift+F
- F5 הרצה · F6 מכונה · F7 בדיקה (סימוני gutter)
- `יוד סדר` / Shift+Alt+F לסידור קוד
- איקון יוד ב־EXE, בחלון ובפס המשימות

פרטים: [`yod-ide/README.md`](yod-ide/README.md).

### אריזה
```powershell
.\yod.exe ארוז תוכנית.יוד
```

---

## מבנה הקוד

```
yod/                 # ליבת השפה + CLI
yod-ide/             # עורך Electron (ברירת מחדל ב־Windows)
פרוייקט דוגמה/       # סנייק, מכרות, צייר, גרפים, רכיבים, מידע מחשב
מדריך שפת יוד/       # מדריך HTML
```

---

## מדריך

- [`מדריך שפת יוד/מדריך שפת יוד.html`](מדריך%20שפת%20יוד/מדריך%20שפת%20יוד.html)

---

## רישיון

הפרויקט מופץ תחת [MIT License](LICENSE).
**המקור הרשמי:** [ycohen888/Yod-language-programming](https://github.com/ycohen888/Yod-language-programming)

---

## תרומה / פיתוח

ראו [`CONTRIBUTING.md`](CONTRIBUTING.md).
[Issues](https://github.com/ycohen888/Yod-language-programming/issues)

---

נבנה באהבה לעברית 🇮🇱
