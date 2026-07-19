package gpu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPerspectiveLookAt(t *testing.T) {
	p := Perspective(60, 16.0/9.0, 0.1, 100)
	if p[0] == 0 || p[5] == 0 {
		t.Fatalf("perspective degenerate: %#v", p)
	}
	v := LookAt(Vec3{0, 1, 4}, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	if v[15] != 1 {
		t.Fatalf("lookAt w row broken")
	}
}

func TestCubeMesh(t *testing.T) {
	m := NewCubeMesh()
	if len(m.Positions) < 36 {
		t.Fatalf("cube too small: %d", len(m.Positions))
	}
	if len(m.Indices) == 0 {
		t.Fatal("cube missing indices")
	}
}

func TestLoadGLTFPyramid(t *testing.T) {
	path := filepath.Join("..", "..", "..", "פרוייקט דוגמה", "תלת", "מודלים", "פירמידה.gltf")
	if _, err := os.Stat(path); err != nil {
		t.Skip("pyramid gltf missing:", err)
	}
	m, err := LoadGLTF(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Positions) < 9 {
		t.Fatalf("positions: %d", len(m.Positions))
	}
	if len(m.Indices) < 3 {
		t.Fatalf("indices: %d", len(m.Indices))
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "מניפסט.json")
	man := &AssetManifest{Assets: []AssetEntry{{Name: "a", Kind: "תמונה", Path: "a.png"}}}
	if err := SaveManifest(path, man); err != nil {
		t.Fatal(err)
	}
	got, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.FindAsset("a") == nil {
		t.Fatal("missing asset")
	}
}
