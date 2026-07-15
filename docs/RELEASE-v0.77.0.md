# יוד 0.77.0

## הורדה מוכנה (Windows)

הורידו את **`yod-0.77.0-windows-amd64.zip`**, חלצו את **כל** התיקייה, והריצו `yod.exe`
(או לחצו על **`Yod IDE.exe`**).

בחבילה כלולים:
- `yod.exe` — CLI + מפעיל עורך
- **`Yod IDE.exe`** + `resources/` — עורך Electron **ארוז ומוכן** (אין צורך ב־Node)
- `מדריך שפת יוד/` — מדריך HTML (נפתח מתוך העורך: עזרה ← מדריך)
- `פרוייקט דוגמה/` — פרויקטי הדגמה

> חובה לחלץ את **כל** התיקייה. אין צורך ב־`npm` / Node כדי להשתמש בעורך.

## עורך (Electron, ארוז)
- אפליקציה עצמאית: `Yod IDE.exe` (electron-builder)
- `yod` / `yod עורך` מזהים את האפליקציה הארוזה ליד `yod.exe`
- React + CodeMirror 6, RTL מלא, ממשק בסגנון VS Code
- סייר, ניתוח קובץ, סמלי פרויקט, F12 / הפניות, F5–F7

## שפה וכלים
- סגנון רשמי כמו PHP: `()` / `{}` (גם sof/`סוף` לתאימות)
- `יוד סדר` — מסדר קוד
- שיפורי פורמט/לינט וספריות

## פרויקטי דוגמה
- `פרוייקט דוגמה/` מעודכן: סנייק, מכרות, צייר, גרפים, רכיבים, מידע מחשב

## בנייה מהמקור (למפתחים)
```powershell
cd yod
go build -o ..\yod.exe ./cmd/yod
cd ..\yod-ide
npm install
npm run dist
cd ..
.\yod.exe עורך
```

אריזת Release מקומית:
```powershell
powershell -File yod\tools\pack_windows_release.ps1
```
