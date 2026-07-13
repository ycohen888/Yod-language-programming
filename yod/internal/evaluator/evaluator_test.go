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

func evalSource(t *testing.T, src string) (object.Object, string) {
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
	result := Eval(prog, env)
	return result, buf.String()
}

func TestMethodWithoutParens(t *testing.T) {
	_, out := evalSource(t, `
משתנה פפ = "אאא"
הדפס: פפ.אורך
הדפס: פפ.אורך + 1
`)
	if !strings.Contains(out, "3") {
		t.Fatalf("expected length 3, got %q", out)
	}
	if strings.Contains(out, "מתודה מובנית") {
		t.Fatalf("should auto-call method, got %q", out)
	}
	if !strings.Contains(out, "4") {
		t.Fatalf("expected 4 from אורך+1, got %q", out)
	}
}

func TestVarWithoutInitialValue(t *testing.T) {
	_, out := evalSource(t, `
משתנה תשובה
הדפס: תשובה
תשובה = "מוכן"
הדפס: תשובה
`)
	if !strings.Contains(out, "ריק") || !strings.Contains(out, "מוכן") {
		t.Fatalf("output %q", out)
	}
}

func TestEvalFunction(t *testing.T) {
	_, out := evalSource(t, `
פונקציה כפול(x) { החזר x * 2; }
הדפס(כפול(21));
`)
	if !strings.Contains(out, "42") {
		t.Fatalf("output %q", out)
	}
}

func TestEvalClosure(t *testing.T) {
	_, out := evalSource(t, `
פונקציה צור(התחלה) {
  משתנה נ = התחלה;
  פונקציה הבא() { נ = נ + 1; החזר נ; }
  החזר הבא;
}
משתנה מ = צור(10);
הדפס(מ());
הדפס(מ());
`)
	if !strings.Contains(out, "11") || !strings.Contains(out, "12") {
		t.Fatalf("output %q", out)
	}
}

func TestEvalClass(t *testing.T) {
	_, out := evalSource(t, `
מחלקה קופסה {
  פונקציה בנאי(v) { זה.ערך = v; }
  פונקציה הצג() { הדפס(זה.ערך); }
}
משתנה ק = חדש קופסה(7);
ק.הצג();
`)
	if !strings.Contains(out, "7") {
		t.Fatalf("output %q", out)
	}
}

func TestEvalError(t *testing.T) {
	res, _ := evalSource(t, `הדפס(1 / 0);`)
	if res == nil || res.Type() != object.ErrorObj {
		t.Fatalf("expected error, got %#v", res)
	}
}
