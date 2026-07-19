package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/vfs"
)

func TestBundleRoundTrip(t *testing.T) {
	vfs.Clear()
	defer vfs.Clear()

	dir := t.TempDir()
	files := map[string][]byte{
		"התחל.יוד":           []byte("כלול \"עזר.יוד\"\nהדפס: 1\n"),
		"עזר.יוד":             []byte("הדפס: 2\n"),
		"עיצוב/קליפה.html": []byte("<html/>"),
	}
	out := filepath.Join(dir, "app.exe")
	engine := []byte("MZFAKEENGINE")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(engine); err != nil {
		t.Fatal(err)
	}
	if err := WriteBundleOverlay(f, files, "התחל.יוד"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	src, virtual, ok, err := LoadEmbedded(out)
	if err != nil || !ok {
		t.Fatalf("LoadEmbedded: ok=%v err=%v", ok, err)
	}
	if !strings.Contains(src, "כלול") {
		t.Fatalf("entry source: %q", src)
	}
	if !strings.Contains(virtual, "התחל.יוד") {
		t.Fatalf("virtual=%q", virtual)
	}
	fs := vfs.Active()
	if fs == nil {
		t.Fatal("VFS not mounted")
	}
	data, ok := fs.Get("עיצוב/קליפה.html")
	if !ok || string(data) != "<html/>" {
		t.Fatalf("asset missing: %v %q", ok, data)
	}
	stripped := stripOverlay(mustRead(t, out))
	if string(stripped) != string(engine) {
		t.Fatalf("stripOverlay got %q", stripped)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCollectBundleNested(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "התחל.יוד"), []byte("כלול \"א.יוד\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "א.יוד"), []byte("כלול \"ב.יוד\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ב.יוד"), []byte("הדפס: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := CollectBundle(filepath.Join(root, "התחל.יוד"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"התחל.יוד", "א.יוד", "ב.יוד"} {
		if _, ok := b.Files[k]; !ok {
			t.Fatalf("missing %s in bundle: %#v", k, b.Files)
		}
	}
}
