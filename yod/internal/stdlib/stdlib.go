package stdlib

import (
	"fmt"
	"sort"

	"yod/internal/object"
)

func BuiltinNames() []string {
	return []string{
		"בסיס", "קבצים", "JSON", "זמן", "מתמטיקה", "מערכת", "רשת", "SQL", "חלונות", "ציור", "הצפנה", "לוח",
	}
}

// BuiltinAttrNames — שמות מאפיינים/פונקציות של ספרייה מובנית (להשלמה בעורך).
func BuiltinAttrNames(name string) []string {
	mod, err := LoadBuiltin(name)
	if err != nil {
		return nil
	}
	m, ok := mod.(*object.Module)
	if !ok || m == nil {
		return nil
	}
	out := make([]string, 0, len(m.Attrs))
	for k := range m.Attrs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LoadBuiltin טוען מודול מובנה לפי שם ומחזיר אותו (או שגיאה).
func LoadBuiltin(name string) (object.Object, error) {
	switch name {
	case "בסיס":
		return NewBasisModule(), nil
	case "קבצים":
		return NewFilesModule(), nil
	case "SQL", "sql":
		return NewSQLModule(), nil
	case "רשת":
		return NewNetModule(), nil
	case "חלונות":
		return NewWindowsModule(), nil
	case "ציור":
		return NewDrawingModule(), nil
	case "הצפנה":
		return NewCryptoModule(), nil
	case "לוח":
		return NewClipboardModule(), nil
	case "JSON", "json":
		return NewJSONModule(), nil
	case "זמן":
		return NewTimeModule(), nil
	case "מתמטיקה":
		return NewMathModule(), nil
	case "מערכת":
		return NewSystemModule(), nil
	default:
		return nil, fmt.Errorf("ספרייה מובנית לא נמצאה: %q", name)
	}
}

func errObj(msg string) *object.Error {
	return &object.Error{Message: msg}
}

func expectArgs(name string, n int, args []object.Object) *object.Error {
	if len(args) != n {
		return errObj(fmt.Sprintf("%s מצפה ל־%d ארגומנטים, קיבל %d", name, n, len(args)))
	}
	return nil
}

func asString(obj object.Object) (string, bool) {
	s, ok := obj.(*object.String)
	if !ok {
		return "", false
	}
	return s.Value, true
}
