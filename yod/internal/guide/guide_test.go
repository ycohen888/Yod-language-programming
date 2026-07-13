package guide

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractEmbedded(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	dir, err := extractToAppData()
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(dir, indexName)
	if fi, err := os.Stat(index); err != nil || fi.IsDir() {
		t.Fatalf("index missing: %v", err)
	}
	css := filepath.Join(dir, "css", "guide.css")
	if _, err := os.Stat(css); err != nil {
		t.Fatalf("css missing after extract: %v", err)
	}
	dir2, err := extractToAppData()
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != dir {
		t.Fatalf("cache path changed: %q vs %q", dir, dir2)
	}
}

func TestFindOnDiskPreferRepo(t *testing.T) {
	p := FindOnDisk()
	if p == "" {
		t.Skip("מדריך לא על הדיסק (בנייה מבודדת)")
	}
	if filepath.Base(p) != indexName {
		t.Fatalf("unexpected index name: %s", p)
	}
}
