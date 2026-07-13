package vm

import (
	"bytes"
	"strings"
	"testing"

	"yod/internal/compiler"
	"yod/internal/console"
	"yod/internal/lexer"
	"yod/internal/parser"
)

func runVMSource(t *testing.T, src string) string {
	t.Helper()
	var buf bytes.Buffer
	console.SetStdout(&buf)
	defer console.SetStdout(nil)

	p := parser.New(lexer.New(src))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	c := compiler.New(".")
	if err := c.Compile(prog); err != nil {
		t.Fatalf("compile: %v", err)
	}
	machine := New(c.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("vm: %v", err)
	}
	return buf.String()
}

func TestVMArithmetic(t *testing.T) {
	out := runVMSource(t, `הדפס(6 * 7);`)
	if !strings.Contains(out, "42") {
		t.Fatalf("output %q", out)
	}
}

func TestVMFunctionClosure(t *testing.T) {
	out := runVMSource(t, `
פונקציה צור_מוסיף(כמה) {
  פונקציה הוסף(x) { החזר x + כמה; }
  החזר הוסף;
}
משתנה פ = צור_מוסיף(5);
הדפס(פ(7));
`)
	if !strings.Contains(out, "12") {
		t.Fatalf("output %q", out)
	}
}

func TestVMArray(t *testing.T) {
	out := runVMSource(t, `
משתנה ר = [1, 2, 3];
הדפס(ר[1]);
הדפס(אורך(ר));
`)
	if !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Fatalf("output %q", out)
	}
}

func TestVMClass(t *testing.T) {
	out := runVMSource(t, `
מחלקה שחקן {
  משתנה נקודות = 0;
  פונקציה בנאי(שם) { זה.שם = שם; }
  פונקציה הצג() { הדפס(זה.שם); }
}
משתנה פ = חדש שחקן("נועם");
פ.הצג();
`)
	if !strings.Contains(out, "נועם") {
		t.Fatalf("output %q", out)
	}
}
