package evaluator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/console"
	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
)

func TestImportExportModule(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "חשבון.יוד")
	if err := os.WriteFile(modPath, []byte(`
מודול חשבון
יצא פונקציה סכום א, ב
  החזר א + ב
סוף
יצא משתנה גרסה = 2
פונקציה פנימי
  החזר 99
סוף
`), 0644); err != nil {
		t.Fatal(err)
	}
	_, out := evalInDir(t, dir, `
יבא חשבון מתוך "חשבון.יוד"
הדפס: חשבון.סכום(3, 4)
הדפס: חשבון.גרסה
`)
	if !strings.Contains(out, "7") {
		t.Fatalf("want 7, got %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("want version 2, got %q", out)
	}
	if strings.Contains(out, "99") {
		t.Fatalf("internal function should not leak, got %q", out)
	}
}

func TestImportNamed(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "עזר.יוד")
	if err := os.WriteFile(modPath, []byte(`
יצא פונקציה כפול א
  החזר א * 2
סוף
`), 0644); err != nil {
		t.Fatal(err)
	}
	_, out := evalInDir(t, dir, `
יבא { כפול } מתוך "עזר.יוד"
הדפס: כפול(5)
`)
	if !strings.Contains(out, "10") {
		t.Fatalf("want 10, got %q", out)
	}
}

func evalInDir(t *testing.T, baseDir, src string) (object.Object, string) {
	t.Helper()
	var buf bytes.Buffer
	console.SetStdout(&buf)
	defer console.SetStdout(nil)

	p := parser.New(lexer.New(src))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	env := NewGlobalEnv(baseDir)
	result := Eval(prog, env)
	return result, buf.String()
}
