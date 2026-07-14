# תרומה ליוד (Contributing)

תודה שאתם רוצים לעזור! יוד היא שפת תכנות בעברית — כל תיקון, דוגמה או שיפור בתיעוד מתקבלים בברכה.

## התחלה מהירה

1. עשו Fork לריפו והעתיקו לעצמכם (clone)
2. התקינו [Go](https://go.dev/dl/) (מומלץ לפי `yod/go.mod`)
3. בנו והריצו:

```powershell
cd yod
go build -o ..\yod.exe ./cmd/yod
..\yod.exe גרסה
..\yod.exe הרץ examples\shalom.יוד
```

### עורך Electron (מומלץ)

דרוש גם Node.js:

```powershell
cd yod-ide
npm install
npm run build
cd ..
.\yod.exe עורך
```

האייקון של `yod.exe` מגיע מ־`cmd/yod/rsrc_windows_amd64.syso` (משובץ בריפו).
אם צריך לייצר מחדש אחרי שינוי לוגו:

```powershell
cd yod\cmd\yod
go generate
```

(דורש `rsrc`: `go install github.com/akavel/rsrc@latest`)

4. הריצו בדיקות:

```powershell
cd yod
go test ./...
```

ב־Windows הבדיקות כוללות גם רכיבי עורך/חלונות. בלינוקס/מק רוב הספריות נבדקות; חלקים תלויי־Windows מדלגים או משתמשים ב־stubs.

## איך עובדים על שינוי

1. פתחו [Issue](https://github.com/ycohen888/Yod-language-programming/issues) או בחרו Issue קיים (חפשו תווית `good first issue`)
2. צרו ענף: `feature/...` או `fix/...`
3. כתבו קוד + בדיקות כשיש מה לבדוק
4. עדכנו תיעוד אם שיניתם יכולת שהמשתמש רואה:
   - `מדריך שפת יוד/`
   - `info_program.html` (יומן)
   - `yod-ide/README.md` (עורך)
5. Commit עם הודעה ברורה (מה ולמה), ואז Push + Pull Request

## סגנון PR

- כותרת קצרה בעברית או באנגלית
- תארו בגוף ה־PR: מה השתנה, איך לבדוק
- שמרו על שינויים ממוקדים — עדיף כמה PR קטנים מאשר אחד ענק

## מבנה חשוב

| נתיב | תפקיד |
|------|--------|
| `yod/cmd/yod` | CLI |
| `yod/internal/lexer|parser|evaluator|vm` | ליבת השפה |
| `yod/internal/stdlib` | ספריות מובנות |
| `yod/internal/editor` | עורך Win32 + השקת Electron |
| `yod/internal/format` | `יוד סדר` |
| `yod-ide/` | עורך Electron (ברירת מחדל) |
| `פרוייקט דוגמה/` | פרויקטי הדגמה מלאים |
| `yod/examples` | דוגמאות קצרות `.יוד` |
| `מדריך שפת יוד/` | מדריך למשתמש (מקור; משובץ גם ב־`yod/internal/guide/guide.zip`) |
| `yod/internal/guide/` | שיבוץ המדריך בבינארי — אחרי עדכון HTML: `go generate ./internal/guide` |

## רישיון

בתרומה אתם מסכימים שהקוד ייכנס תחת [MIT License](LICENSE), עם ייחוס למקור הרשמי של יוד.

## שאלות?

פתחו Issue — נשמח לעזור.
