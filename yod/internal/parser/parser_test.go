package parser

import (
	"testing"

	"yod/internal/ast"
	"yod/internal/lexer"
)

func parse(t *testing.T, input string) *ast.Program {
	t.Helper()
	p := New(lexer.New(input))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	return prog
}

func TestSubtractionNotJuxtaCall(t *testing.T) {
	prog := parse(t, "(א - ב).מוחלט()")
	es, ok := prog.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("got %T", prog.Statements[0])
	}
	call, ok := es.Expr.(*ast.CallExpression)
	if !ok {
		t.Fatalf("want Call, got %T", es.Expr)
	}
	mem, ok := call.Function.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("want Member, got %T", call.Function)
	}
	if _, ok := mem.Object.(*ast.InfixExpression); !ok {
		t.Fatalf("want Infix under member (חיסור), got %T — juxta bug?", mem.Object)
	}
}


func TestVarWithoutValue(t *testing.T) {
	prog := parse(t, "משתנה א\nהדפס: א")
	if len(prog.Statements) != 2 {
		t.Fatalf("statements: %d", len(prog.Statements))
	}
	vs, ok := prog.Statements[0].(*ast.VarStatement)
	if !ok {
		t.Fatalf("got %T", prog.Statements[0])
	}
	if vs.Value != nil {
		t.Fatalf("expected nil value, got %#v", vs.Value)
	}
}

func TestIfFunctionClass(t *testing.T) {
	src := `
פונקציה כפול(x) { החזר x * 2; }
מחלקה נקודה {
  פונקציה בנאי(א) { זה.א = א; }
}
אם (אמת) { הדפס(1); } אחרת { הדפס(0); }
`
	prog := parse(t, src)
	if len(prog.Statements) < 3 {
		t.Fatalf("expected >=3 statements, got %d", len(prog.Statements))
	}
}

func TestSofBlocks(t *testing.T) {
	src := `
פונקציה כפול x
    החזר x * 2;
סוף

מחלקה נקודה
    פונקציה בנאי א
        זה.א = א;
    סוף
סוף

משתנה א = 1;
אם אמת
    הדפס: 1;
אחרת
    הדפס: 0;
סוף

כל_עוד א <= 3
    א = א + 1;
סוף
`
	prog := parse(t, src)
	if len(prog.Statements) < 4 {
		t.Fatalf("expected >=4 statements, got %d", len(prog.Statements))
	}
}

func TestNoParensStyle(t *testing.T) {
	src := `
פונקציה כפל x, y
    החזר x * y;
סוף
הדפס: "שלום";
הדפס "עולם";
הדפס: כפל: 2, 7;
עבור שם בתוך ["א", "ב"]
    הדפס: שם;
סוף
אם שם == "א"
    הדפס: 1;
סוף
`
	prog := parse(t, src)
	if len(prog.Statements) < 4 {
		t.Fatalf("got %d", len(prog.Statements))
	}
}

func TestRTLMirroredBraces(t *testing.T) {
	src := `
משתנה א = 1;
אם (א > 10) } הדפס("א גדול מ-10"); {
כל_עוד (א <= 3) } א = א + 1; {
פונקציה כפל(x, y) } החזר x * y; {
`
	prog := parse(t, src)
	if len(prog.Statements) < 4 {
		t.Fatalf("expected >=4 statements, got %d", len(prog.Statements))
	}
}

func TestOptionalConditionParens(t *testing.T) {
	prog := parse(t, `אם שם == "דוד" { הדפס(1); }`)
	if len(prog.Statements) != 1 {
		t.Fatalf("got %d", len(prog.Statements))
	}
}

func TestRequireSemicolon(t *testing.T) {
	// ; כבר לא חובה — פקודה בשורה מספיקה
	prog := parse(t, "הדפס(1)")
	if len(prog.Statements) != 1 {
		t.Fatalf("got %d statements", len(prog.Statements))
	}
}

func TestNewlineEndsStatement(t *testing.T) {
	prog := parse(t, "הדפס(1)\nהדפס(2)")
	if len(prog.Statements) != 2 {
		t.Fatalf("got %d statements", len(prog.Statements))
	}
}

func TestMultiStatementOneLine(t *testing.T) {
	prog := parse(t, `הדפס(1); הדפס(2);`)
	if len(prog.Statements) != 2 {
		t.Fatalf("got %d statements", len(prog.Statements))
	}
}

func TestSameLineNeedsSemicolon(t *testing.T) {
	p := New(lexer.New(`הדפס(1) הדפס(2)`))
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for two statements on one line without ;")
	}
}

func TestSyntaxError(t *testing.T) {
	p := New(lexer.New(`משתנה = `))
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors")
	}
}
