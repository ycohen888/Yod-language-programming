package stdlib

import (
	"os"
	"path/filepath"
	"testing"

	"yod/internal/object"
)

func TestSteppedCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	payload := make([]byte, 1000)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	if err := os.WriteFile(src, payload, 0644); err != nil {
		t.Fatal(err)
	}

	start := filesStartCopy(&object.String{Value: src}, &object.String{Value: dst})
	h, ok := start.(*object.Hash)
	if !ok {
		t.Fatalf("start: %v", start)
	}
	id := h.Pairs["מזהה"]

	done := false
	for i := 0; i < 100 && !done; i++ {
		st := filesStepCopy(id, &object.Number{Value: 200})
		sh, ok := st.(*object.Hash)
		if !ok {
			t.Fatalf("step: %v", st)
		}
		if errS, _ := sh.Pairs["שגיאה"].(*object.String); errS != nil && errS.Value != "" {
			t.Fatalf("copy error: %s", errS.Value)
		}
		if b, ok := sh.Pairs["הסתיים"].(*object.Boolean); ok && b.Value {
			done = true
		}
	}
	if !done {
		t.Fatal("copy did not finish")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(payload) {
		t.Fatalf("len got=%d want=%d", len(got), len(payload))
	}
	for i := range payload {
		if got[i] != payload[i] {
			t.Fatalf("byte %d mismatch", i)
		}
	}
}
