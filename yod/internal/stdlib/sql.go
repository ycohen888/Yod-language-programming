package stdlib

import (
	"database/sql"
	"fmt"
	"strings"

	"yod/internal/object"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func NewSQLModule() *object.Module {
	m := &object.Module{Name: "SQL", Attrs: map[string]object.Object{}}
	m.Attrs["התחבר"] = &object.Builtin{Fn: sqlConnect}
	return m
}

func sqlConnect(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("SQL.התחבר מצפה ל־(סוג, מחרוזת_חיבור) או (נתיב) ל־sqlite")
	}

	driverKey := "sqlite"
	var dsn string
	if len(args) == 1 {
		p, ok := asString(args[0])
		if !ok {
			return errObj("SQL.התחבר: נתיב חייב להיות מחרוזת")
		}
		dsn = p
	} else {
		d, ok := asString(args[0])
		if !ok {
			return errObj("SQL.התחבר: סוג מסד חייב להיות מחרוזת")
		}
		driverKey = d
		p, ok := asString(args[1])
		if !ok {
			return errObj("SQL.התחבר: מחרוזת חיבור חייבת להיות מחרוזת")
		}
		dsn = p
	}

	goDriver, errMsg := resolveSQLDriver(driverKey)
	if errMsg != "" {
		return errObj(errMsg)
	}

	db, err := sql.Open(goDriver, dsn)
	if err != nil {
		return errObj("חיבור נכשל: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return errObj("חיבור נכשל: " + err.Error())
	}
	return newSQLConn(db, goDriver, driverKey)
}

func resolveSQLDriver(name string) (goDriver string, errMsg string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sqlite", "sqlite3":
		return "sqlite", ""
	case "mysql", "מישקל":
		return "mysql", ""
	case "postgres", "postgresql", "pg", "פוסטגרס":
		return "postgres", ""
	default:
		return "", "סוג מסד לא נתמך: " + name + " (נתמכים: sqlite, mysql, postgres)"
	}
}

func newSQLConn(db *sql.DB, goDriver, kind string) *object.Module {
	m := &object.Module{Name: "חיבור_SQL", Attrs: map[string]object.Object{}}
	m.Attrs["סוג"] = &object.String{Value: kind}
	m.Attrs["הפעל"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sqlExec(db, goDriver, args...)
	}}
	m.Attrs["שלוף"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sqlQuery(db, goDriver, args...)
	}}
	m.Attrs["סגור"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if err := db.Close(); err != nil {
			return errObj("סגירה נכשלה: " + err.Error())
		}
		return &object.Null{}
	}}
	return m
}

func sqlArgs(args []object.Object) (query string, params []any, err *object.Error) {
	if len(args) < 1 {
		return "", nil, errObj("חסרה שאילתת SQL")
	}
	q, ok := asString(args[0])
	if !ok {
		return "", nil, errObj("שאילתה חייבת להיות מחרוזת")
	}
	params = make([]any, 0, len(args)-1)
	for _, a := range args[1:] {
		params = append(params, toGoValue(a))
	}
	return q, params, nil
}

func toGoValue(o object.Object) any {
	switch v := o.(type) {
	case *object.Number:
		if v.Value == float64(int64(v.Value)) {
			return int64(v.Value)
		}
		return v.Value
	case *object.String:
		return v.Value
	case *object.Boolean:
		return v.Value
	case *object.Null:
		return nil
	default:
		return v.Inspect()
	}
}

func sqlExec(db *sql.DB, goDriver string, args ...object.Object) object.Object {
	q, params, err := sqlArgs(args)
	if err != nil {
		return err
	}
	q = rewritePlaceholders(q, goDriver)
	res, e := db.Exec(q, params...)
	if e != nil {
		return errObj("הפעל נכשל: " + e.Error())
	}
	n, _ := res.RowsAffected()
	return &object.Number{Value: float64(n)}
}

func sqlQuery(db *sql.DB, goDriver string, args ...object.Object) object.Object {
	q, params, err := sqlArgs(args)
	if err != nil {
		return err
	}
	q = rewritePlaceholders(q, goDriver)
	rows, e := db.Query(q, params...)
	if e != nil {
		return errObj("שלוף נכשל: " + e.Error())
	}
	defer rows.Close()

	cols, e := rows.Columns()
	if e != nil {
		return errObj("שלוף נכשל: " + e.Error())
	}

	out := &object.Array{Elements: []object.Object{}}
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if e := rows.Scan(ptrs...); e != nil {
			return errObj("סריקת שורה נכשלה: " + e.Error())
		}
		row := &object.Hash{Pairs: map[string]object.Object{}}
		for i, col := range cols {
			row.Pairs[col] = fromDBValue(raw[i])
		}
		out.Elements = append(out.Elements, row)
	}
	if e := rows.Err(); e != nil {
		return errObj("שלוף נכשל: " + e.Error())
	}
	return out
}

// rewritePlaceholders — ב־PostgreSQL ממיר ? ל־$1,$2 כדי לשמור על אותה תחביר בכל המסדים
func rewritePlaceholders(query, goDriver string) string {
	if goDriver != "postgres" {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprintf("%d", n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func fromDBValue(v any) object.Object {
	if v == nil {
		return &object.Null{}
	}
	switch x := v.(type) {
	case int64:
		return &object.Number{Value: float64(x)}
	case float64:
		return &object.Number{Value: x}
	case string:
		return &object.String{Value: x}
	case []byte:
		return &object.String{Value: string(x)}
	case bool:
		return &object.Boolean{Value: x}
	default:
		return &object.String{Value: fmt.Sprint(x)}
	}
}
