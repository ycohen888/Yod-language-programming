//go:build windows

package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeModel_FlatVisible(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "a"))
	mustMkdir(t, filepath.Join(root, "a", "nested"))
	mustWrite(t, filepath.Join(root, "a", "nested", "deep.יוד"), "x")
	mustWrite(t, filepath.Join(root, "a", "one.יוד"), "x")
	mustWrite(t, filepath.Join(root, "b.יוד"), "x")
	mustMkdir(t, filepath.Join(root, "z"))

	m := NewFileTreeModel()
	m.SetRoot(root)

	// בהתחלה: רק ילדי השורש (a/, z/, b.יוד) — nested לא גלוי
	names := visibleNames(m)
	if !containsAll(names, []string{"a", "z", "b.יוד"}) {
		t.Fatalf("צפוי ילדי שורש, קיבל: %v", names)
	}
	if containsAll(names, []string{"nested"}) {
		t.Fatalf("nested לא אמור להיות גלוי לפני פתיחת a: %v", names)
	}

	aPath := filepath.Join(root, "a")
	m.ToggleExpanded(aPath)
	names = visibleNames(m)
	if !containsAll(names, []string{"a", "nested", "one.יוד", "z", "b.יוד"}) {
		t.Fatalf("אחרי פתיחת a חסרים צמתים: %v", names)
	}

	// depths
	for _, n := range m.Visible() {
		switch n.Name {
		case "a", "z", "b.יוד":
			if n.Depth != 0 {
				t.Fatalf("%s depth=%d want 0", n.Name, n.Depth)
			}
		case "nested", "one.יוד":
			if n.Depth != 1 {
				t.Fatalf("%s depth=%d want 1", n.Name, n.Depth)
			}
		}
	}

	m.ToggleExpanded(filepath.Join(aPath, "nested"))
	names = visibleNames(m)
	if !containsAll(names, []string{"deep.יוד"}) {
		t.Fatalf("אחרי פתיחת nested חסר deep: %v", names)
	}

	m.ToggleExpanded(aPath) // סגירה — גם הצאצאים נעלמים מהתצוגה
	names = visibleNames(m)
	if containsAll(names, []string{"nested", "one.יוד", "deep.יוד"}) {
		t.Fatalf("אחרי סגירת a עדיין רואים צאצאים: %v", names)
	}
}

func TestFileTreeModel_ExpandToPath(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "src", "lib", "x.יוד")
	mustMkdir(t, filepath.Dir(deep))
	mustWrite(t, deep, "x")

	m := NewFileTreeModel()
	m.SetRoot(root)
	m.ExpandToPath(deep)
	names := visibleNames(m)
	if !containsAll(names, []string{"src", "lib", "x.יוד"}) {
		t.Fatalf("ExpandToPath לא פתח אבות: %v", names)
	}
	if idx := m.IndexOfPath(deep); idx < 0 {
		t.Fatalf("IndexOfPath לא מצא את הקובץ")
	}
}

func visibleNames(m *FileTreeModel) []string {
	out := make([]string, 0, len(m.Visible()))
	for _, n := range m.Visible() {
		out = append(out, n.Name)
	}
	return out
}

func containsAll(have, need []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, n := range need {
		if !set[n] {
			return false
		}
	}
	return true
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
