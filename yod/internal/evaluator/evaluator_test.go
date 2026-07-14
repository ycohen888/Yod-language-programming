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

func TestUnaryMinusEval(t *testing.T) {
	_, out := evalSource(t, `
הדפס: -1
הדפס: -(-3)
משתנה x = 5
הדפס: -x
פונקציה כפול א
  החזר א * 2
סוף
הדפס: כפול(-4)
`)
	if !strings.Contains(out, "-1") {
		t.Fatalf("want -1, got %q", out)
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("want 3 from -(-3), got %q", out)
	}
	if !strings.Contains(out, "-5") {
		t.Fatalf("want -5, got %q", out)
	}
	if !strings.Contains(out, "-8") {
		t.Fatalf("want -8 from call, got %q", out)
	}
}

func TestForRangeEval(t *testing.T) {
	_, out := evalSource(t, `
משתנה ס = 0
עבור i מ 1 עד 3
  ס = ס + i
סוף
הדפס: ס
עבור j מ 0 עד 4 בצע 2
  הדפס: j
סוף
`)
	if !strings.Contains(out, "6") {
		t.Fatalf("want sum 6, got %q", out)
	}
	if !strings.Contains(out, "0") || !strings.Contains(out, "2") || !strings.Contains(out, "4") {
		t.Fatalf("want even steps, got %q", out)
	}
}

func TestBlockCommentEval(t *testing.T) {
	_, out := evalSource(t, `
/* לא רץ
הדפס: "לא"
*/
הדפס: "כן"
`)
	if strings.Contains(out, "לא") {
		t.Fatalf("block comment should hide print, got %q", out)
	}
	if !strings.Contains(out, "כן") {
		t.Fatalf("want כן, got %q", out)
	}
}

func TestTemplateLiteral(t *testing.T) {
	_, out := evalSource(t, `
משתנה שם = "נועם"
הדפס: `+"`שלום ${שם}`"+`
הדפס: `+"`סכום ${1 + 2}`"+`
`)
	if !strings.Contains(out, "שלום נועם") {
		t.Fatalf("want שלום נועם, got %q", out)
	}
	if !strings.Contains(out, "סכום 3") {
		t.Fatalf("want סכום 3, got %q", out)
	}
}

func TestEnum(t *testing.T) {
	_, out := evalSource(t, `
סדרה איכות { חלש, טוב, מעולה }
משתנה ק = איכות.טוב
אם ק == איכות.טוב
  הדפס: "כן"
סוף
הדפס: ק
`)
	if !strings.Contains(out, "כן") {
		t.Fatalf("want כן, got %q", out)
	}
	if !strings.Contains(out, "איכות.טוב") {
		t.Fatalf("want איכות.טוב, got %q", out)
	}
}

func TestHashInsertionOrder(t *testing.T) {
	_, out := evalSource(t, `
משתנה מ = { "ג": 3, "א": 1, "ב": 2 }
עבור מפתח בתוך מ
  הדפס: מפתח
סוף
`)
	idxG := strings.Index(out, "ג")
	idxA := strings.Index(out, "א")
	idxB := strings.Index(out, "ב")
	if idxG < 0 || idxA < 0 || idxB < 0 || !(idxG < idxA && idxA < idxB) {
		t.Fatalf("want insertion order ג,א,ב got %q", out)
	}
}

func TestTypeAnnotations(t *testing.T) {
	CheckTypes = true
	defer func() { CheckTypes = false }()
	_, out := evalSource(t, `
פונקציה סכום א: מספר, ב: מספר -> מספר
  החזר א + ב
סוף
הדפס: סכום(2, 3)
`)
	if !strings.Contains(out, "5") {
		t.Fatalf("want 5, got %q", out)
	}
	res, _ := evalSource(t, `
פונקציה רק_מספר א: מספר -> מספר
  החזר א
סוף
הדפס: רק_מספר("לא")
`)
	if res == nil || res.Type() != object.ErrorObj {
		t.Fatalf("expected type error, got %#v", res)
	}
}

func TestResultModule(t *testing.T) {
	_, out := evalSource(t, `
משתנה טוב = תוצאה.מ(42)
הדפס: טוב.הצלחה
הדפס: טוב.ערך
משתנה רע = תוצאה.שגיאה("כשל", 7)
הדפס: רע.הצלחה
הדפס: רע.קוד
`)
	if !strings.Contains(out, "אמת") || !strings.Contains(out, "42") {
		t.Fatalf("ok result: %q", out)
	}
	if !strings.Contains(out, "שקר") || !strings.Contains(out, "7") {
		t.Fatalf("err result: %q", out)
	}
}

func TestAsyncTask(t *testing.T) {
	_, out := evalSource(t, `
משתנה מ = משימה: פונקציה ()
  החזר 11
סוף
הדפס: המתן(מ)
משתנה א = משימה: פונקציה ()
  החזר 1
סוף
משתנה ב = משימה: פונקציה ()
  החזר 2
סוף
משתנה ר = במקביל([א, ב])
הדפס: ר[0]
הדפס: ר[1]
`)
	if !strings.Contains(out, "11") {
		t.Fatalf("await: %q", out)
	}
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") {
		t.Fatalf("parallel: %q", out)
	}
}
