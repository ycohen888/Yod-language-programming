package evaluator

import (
	"bytes"
	"strings"
	"testing"

	"yod/internal/console"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
)

// evalSourceLocked — כמו evalSource אבל מחזיק את מנעול הריצה סביב Eval, כפי שעושה
// main.go בפועל. כך מדמים תוכנית אמיתית ובודקים שאין data race מול משימות רקע.
func evalSourceLocked(t *testing.T, src string) (object.Object, string) {
	t.Helper()
	var buf bytes.Buffer
	console.SetStdout(&buf)
	defer console.SetStdout(nil)

	p := parser.New(lexer.New(src))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	env := NewGlobalEnv(".")
	object.LockYod()
	result := Eval(prog, env)
	object.UnlockYod()
	return result, buf.String()
}

func TestTaskAwait(t *testing.T) {
	_, out := evalSourceLocked(t, `
משתנה מ = משימה(פונקציה() { החזר 21 + 21 })
הדפס: המתן(מ)
`)
	if !strings.Contains(out, "42") {
		t.Fatalf("expected 42, got %q", out)
	}
}

func TestParallelTasks(t *testing.T) {
	_, out := evalSourceLocked(t, `
משתנה משימות = []
עבור i בתוך טווח(0, 50) {
    משימות.הוסף(משימה(פונקציה() { החזר 1 }))
}
משתנה תוצאות = במקביל(משימות)
משתנה סכום = 0
עבור ת בתוך תוצאות {
    סכום = סכום + ת
}
הדפס: סכום
`)
	if !strings.Contains(out, "50") {
		t.Fatalf("expected 50, got %q", out)
	}
}

// TestTaskConcurrentGlobals — משימות שקוראות/כותבות מצב משותף דרך המנוע; מיועד להרצה
// עם -race כדי לוודא שהמנעול מונע data race.
func TestTaskConcurrentGlobals(t *testing.T) {
	_, out := evalSourceLocked(t, `
פונקציה עבודה(n) {
    משתנה ס = 0
    עבור i בתוך טווח(0, n) {
        ס = ס + i
    }
    החזר ס
}
משתנה משימות = []
עבור k בתוך טווח(0, 20) {
    משימות.הוסף(משימה(פונקציה() { החזר עבודה(100) }))
}
משתנה תוצאות = במקביל(משימות)
משתנה סכום = 0
עבור ת בתוך תוצאות {
    סכום = סכום + ת
}
הדפס: סכום
`)
	// כל משימה מחזירה 0+1+...+99 = 4950, ו-20 משימות => 99000
	if !strings.Contains(out, "99000") {
		t.Fatalf("expected 99000, got %q", out)
	}
}
