# יוד IDE — עורך הדור הבא

עורך בסגנון **Visual Studio Code**: Electron + React + **CodeMirror 6** (RTL מלא לעברית).

## הפצה (אפליקציה עצמאית)

```powershell
cd yod-ide
npm install
npm run dist
```

נוצרת תיקייה `release/win-unpacked/` עם **`Yod IDE.exe`** — בלי צורך ב־Node אצל המשתמש הסופי.
`yod עורך` מזהה את האפליקציה הארוזה ליד `yod.exe` (או ב־`yod-ide/release/win-unpacked`).

תפריט **עזרה ← מדריך** פותח את המדריך בחלון פנימי של העורך (לא בדפדפן).
בחבילת Release מועתקת גם תיקיית `מדריך שפת יוד/`.

| פקודה | תוצאה |
|--------|--------|
| `npm run dist` | תיקייה ארוזה (`dir`) |
| `npm run dist:portable` | קובץ portable יחיד |

אריזת הפצה מלאה (yod.exe + IDE + מדריך + דוגמאות):

```powershell
powershell -File yod\tools\pack_windows_release.ps1
```

## דרישות (למפתחים)

- [Node.js](https://nodejs.org/) 18+
- `yod.exe` בשורש הריפו (או ב־`PATH`) — להרצה / בדיקה / אריזה מהעורך

## פיתוח מקומי (בלי אריזה)

```powershell
cd yod-ide
npm install
npm run build
```

`npm run build` גם:
- בונה את ממשק Vite ל־`dist/`
- חותם איקון יוד על `יוד.exe` (עותק של Electron בפיתוח)
- יוצר קיצור Start Menu בשם **Yod**

אחרי build פיתוח, `yod עורך` יכול להפעיל גם את Electron מתוך `node_modules` (fallback).

## פיתוח (Hot reload)

```powershell
npm run dev
```

פותח Electron מול Vite ב־`http://127.0.0.1:5173`.

## קיצורים

| קיצור | פעולה |
|--------|--------|
| F1 | פלטת פקודות |
| Ctrl+P | פתיחה מהירה (קבצים / סמלים) |
| Ctrl+F | חיפוש בקובץ (פאנל VS Code–like) |
| Ctrl+H | החלפה בקובץ |
| Ctrl+Shift+F | חיפוש בפרויקט |
| F12 / Ctrl+לחיצה | מעבר להגדרה |
| Shift+F12 | מצא הפניות |
| Ctrl+O | פתיחת קובץ |
| Ctrl+Shift+O | פתיחת תיקייה |
| Ctrl+S | שמירה |
| Ctrl+N | קובץ חדש בסייר |
| F5 | הרצה (`התחל.יוד` בפרויקט) |
| F6 | מכונה |
| F7 | בדיקה (+ סימוני gutter) |
| Ctrl+Shift+P | ארוז ל־EXE |
| הרצה → פתח מיקום EXE | פותח `dist_exe/` בסייר (כבוי בלי התיקייה) |
| Shift+Alt+F | סדר קוד (`yod סדר` או הזחה מקומית) |
| Ctrl+H | חיפוש והחלפה |
| Ctrl+} | מעבר לזוג תואם (`{`/`}` או פתיחה/`סוף`) |

## סיידבר

- **סייר קבצים** — עץ תיקיות
- **ניתוח קובץ** — מחלקות / פונקציות / משתנים בקובץ הפעיל
- **סימבולי פרויקט** — כל הסמלים בפרויקט

## איקון ופס משימות

האיקון הרשמי: `build/icon.ico` (מקור: `../yod/assets/yod-icon-source.png`).

אחרי `npm run dist`, האיקון מוטמע ב־`Yod IDE.exe`. בפיתוח: הצמידו את **Yod** מתפריט התחל (לא את `electron.exe` הגולמי).

## ביצועים

- טעינת קבצים אסינכרונית (עד 8MB)
- מעבר בין טאבים מיידי (מצב CodeMirror נפרד לכל קובץ)
- אינדקס סימבולים להשלמה ולניווט
- הרצה/בדיקה דרך `yod.exe`, פלט בפאנל
- אחרי F7 — סימוני שגיאה ב־gutter ובהדגשת שורה
