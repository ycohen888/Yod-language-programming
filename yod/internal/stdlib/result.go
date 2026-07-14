package stdlib

import "yod/internal/object"

func NewResultModule() *object.Module {
	m := &object.Module{Name: "תוצאה", Attrs: map[string]object.Object{}}
	m.Attrs["מ"] = &object.Builtin{Fn: resultOk}
	m.Attrs["שגיאה"] = &object.Builtin{Fn: resultErr}
	return m
}

func resultOk(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("תוצאה.מ מצפה לערך אחד")
	}
	return object.ResultOk(args[0])
}

func resultErr(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("תוצאה.שגיאה מצפה להודעה ולקוד אופציונלי")
	}
	msg := args[0].Inspect()
	if s, ok := args[0].(*object.String); ok {
		msg = s.Value
	}
	code := 1.0
	if len(args) == 2 {
		if n, ok := args[1].(*object.Number); ok {
			code = n.Value
		}
	}
	return object.ResultErr(msg, code)
}
