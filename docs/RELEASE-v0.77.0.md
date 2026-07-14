# יוד 0.77.0

## הורדה מוכנה (Windows)

הורידו את **`yod-0.77.0-windows-amd64.zip`**, חלצו את **כל** התיקייה, והריצו `yod.exe`.

בחבילה כלולים:
- `yod.exe` — CLI + מפעיל עורך
- `yod-ide/` — **העורך החדש** (Electron + CodeMirror), מוכן להרצה
- `פרוייקט דוגמה/` — פרויקטי הדגמה

> אל תורידו רק את ה־`.exe` הבודד ותצפו לעורך החדש — בלי תיקיית `yod-ide` ליד `yod.exe` נפתח העורך הישן (Win32).

## עורך חדש (Electron)
- עורך בסגנון VS Code: `yod-ide/` (React + CodeMirror 6, RTL מלא)
- `yod` / `yod עורך` מפעילים אותו כש־`yod-ide` זמין (בחבילה או אחרי `npm run build`)
- איקון יוד ב־EXE, בחלון ובפס המשימות; קיצור Start Menu «Yod»
- סייר, ניתוח קובץ, סמלי פרויקט, F12 / הפניות, F5–F7, gutter diagnostics

## שפה וכלים
- סגנון רשמי כמו PHP: `()` / `{}` (גם sof/`סוף` לתאימות)
- `יוד סדר` — מסדר קוד; מתקן מקרי `עבור … בתוך זה.` ו־`אחרת_אם` אחרי אם מקונן
- שיפורי פורמט/לינט וספריות

## פרויקטי דוגמה
- `פרוייקט דוגמה/` מעודכן לסגנון החדש: סנייק, מכרות, צייר, גרפים, רכיבים, מידע מחשב

## בנייה מהמקור (למפתחים)
```powershell
cd yod
go build -o ..\yod.exe ./cmd/yod
cd ..\yod-ide
npm install
npm run build
cd ..
.\yod.exe עורך
```

אריזת Release מקומית:
```powershell
powershell -File yod\tools\pack_windows_release.ps1
```
