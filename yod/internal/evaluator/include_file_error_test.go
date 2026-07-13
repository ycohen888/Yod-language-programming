package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/lexer"
	"yod/internal/object"
	"yod/internal/parser"
)

func TestIncludeErrorNamesFile(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "התחל.יוד")
	bad := filepath.Join(dir, "רע.יוד")
	if err := os.WriteFile(bad, []byte("הדפס: ש\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(main, []byte("כלול \"רע.יוד\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(main)
	if err != nil {
		t.Fatal(err)
	}
	PushSourceFile(main)
	defer PopSourceFile()

	pr := parser.New(lexer.New(string(data)))
	program := pr.ParseProgram()
	if errs := pr.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	env := NewGlobalEnv(dir)
	result := Eval(program, env)
	if result == nil || result.Type() != object.ErrorObj {
		t.Fatalf("expected error, got %v", result)
	}
	msg := result.Inspect()
	if !strings.Contains(msg, "רע.יוד") {
		t.Fatalf("expected file name in error, got %q", msg)
	}
	if !strings.Contains(msg, "לא מוגדר") {
		t.Fatalf("expected undefined var, got %q", msg)
	}
}
