package stdlib

import (
	"testing"

	"yod/internal/object"
)

func TestSystemInfoBuiltins(t *testing.T) {
	mod := NewSystemModule()
	for _, name := range []string{
		"שם_מחשב", "מערכת_הפעלה", "גרסת_הפעלה", "ארכיטקטורה",
		"ליבות", "מעבד", "זיכרון", "איפי", "משתמש", "בית", "מידע",
		"תהליכים", "סיים_תהליך", "שימוש_מעבד", "זמן_פעיל", "כוננים",
	} {
		if _, ok := mod.Attrs[name]; !ok {
			t.Fatalf("חסר מאפיין: %s", name)
		}
	}

	host := call0(t, mod, "שם_מחשב")
	if _, ok := host.(*object.String); !ok {
		t.Fatalf("שם_מחשב צריך מחרוזת, קיבל %T", host)
	}

	osName := call0(t, mod, "מערכת_הפעלה")
	s, ok := osName.(*object.String)
	if !ok || s.Value == "" {
		t.Fatalf("מערכת_הפעלה ריקה או לא מחרוזת: %#v", osName)
	}

	cores := call0(t, mod, "ליבות")
	n, ok := cores.(*object.Number)
	if !ok || n.Value < 1 {
		t.Fatalf("ליבות לא תקינות: %#v", cores)
	}

	mem := call0(t, mod, "זיכרון")
	h, ok := mem.(*object.Hash)
	if !ok {
		t.Fatalf("זיכרון צריך מילון, קיבל %T: %v", mem, mem)
	}
	total, ok := h.Pairs["סהכ_מגה"].(*object.Number)
	if !ok || total.Value < 1 {
		t.Fatalf("סהכ_מגה לא תקין: %#v", h.Pairs["סהכ_מגה"])
	}

	cpu := call0(t, mod, "מעבד")
	ch, ok := cpu.(*object.Hash)
	if !ok {
		t.Fatalf("מעבד צריך מילון, קיבל %T", cpu)
	}
	if _, ok := ch.Pairs["שם"].(*object.String); !ok {
		t.Fatalf("מעבד.שם חסר")
	}

	ips := call0(t, mod, "איפי")
	if _, ok := ips.(*object.Array); !ok {
		t.Fatalf("איפי צריך רשימה, קיבל %T", ips)
	}

	info := call0(t, mod, "מידע")
	ih, ok := info.(*object.Hash)
	if !ok {
		t.Fatalf("מידע צריך מילון, קיבל %T", info)
	}
	if _, ok := ih.Pairs["שם_מחשב"]; !ok {
		t.Fatalf("מידע חסר שם_מחשב")
	}
}

func call0(t *testing.T, mod *object.Module, name string) object.Object {
	t.Helper()
	b, ok := mod.Attrs[name].(*object.Builtin)
	if !ok {
		t.Fatalf("%s אינו Builtin", name)
	}
	return b.Fn()
}
