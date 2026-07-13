# יוד (Yod)

שפת תכנות בעברית — מפרש מאפס ב־Go.

## דרישות

- Go 1.22+ (מותקן אצלך ב־`C:\Program Files\Go`)

## בנייה

מתוך תיקיית `yod` (שם נמצא `go.mod`) — לא משורש `site-programing`:

```bat
cd F:\programming\site-programing\yod
go build -o yod.exe ./cmd/yod
```

לא: `go build -o yod.exe ./yod` משורש הפרויקט (אין שם `go.mod`).

## הרצה

```bat
yod.exe הרץ examples\shalom.יוד
yod.exe גרסה
```

## מה כבר עובד (MVP)

- משתנים (`משתנה` או השמה)
- `הדפס`, `קלט`
- `אם` / `אחרת` / `אחרת_אם`
- `כל_עוד`, `עצור`, `המשך`
- `פונקציה` / `החזר`
- מספרים, מחרוזות, `אמת` / `שקר` / `ריק`
- הערות `//`
