package stdlib

import (
	"strings"
	"testing"

	"yod/internal/object"
)

func TestNumbersFormat(t *testing.T) {
	mod := NewNumbersModule()

	got := callNum(t, mod, "מפריד", &object.Number{Value: 5554324})
	if got != "5,554,324" {
		t.Fatalf("מפריד: got %q", got)
	}

	got = callNum(t, mod, "שלם", &object.Number{Value: 1234.7})
	if got != "1,235" {
		t.Fatalf("שלם: got %q", got)
	}

	got = callNum(t, mod, "עשרוני", &object.Number{Value: 1234.5}, &object.Number{Value: 2})
	if got != "1,234.50" {
		t.Fatalf("עשרוני: got %q", got)
	}

	got = callNum(t, mod, "כסף", &object.Number{Value: 19.9})
	if !strings.HasPrefix(got, "₪") || !strings.Contains(got, "19.90") {
		t.Fatalf("כסף: got %q", got)
	}

	got = callNum(t, mod, "בתים", &object.Number{Value: 5 * 1024 * 1024})
	if !strings.Contains(got, "מגה") {
		t.Fatalf("בתים: got %q", got)
	}

	got = callNum(t, mod, "אחוז", &object.Number{Value: 87})
	if got != "87%" {
		t.Fatalf("אחוז: got %q", got)
	}
}

func callNum(t *testing.T, mod *object.Module, name string, args ...object.Object) string {
	t.Helper()
	b, ok := mod.Attrs[name].(*object.Builtin)
	if !ok {
		t.Fatalf("%s missing", name)
	}
	res := b.Fn(args...)
	s, ok := res.(*object.String)
	if !ok {
		t.Fatalf("%s returned %T %v", name, res, res)
	}
	return s.Value
}
