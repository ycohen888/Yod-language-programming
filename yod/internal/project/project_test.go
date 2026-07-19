package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainPath(t *testing.T) {
	got := MainPath(`C:\proj`)
	want := filepath.Join(`C:\proj`, MainFileName)
	if got != want {
		t.Fatalf("MainPath: got %q want %q", got, want)
	}
}

func TestIsMain(t *testing.T) {
	if !IsMain(filepath.Join("a", MainFileName)) {
		t.Fatal("expected IsMain true")
	}
	if IsMain("אחר.יוד") {
		t.Fatal("expected IsMain false")
	}
}

func TestEnsureMainAndResolve(t *testing.T) {
	dir := t.TempDir()
	main, err := EnsureMain(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(main); err != nil {
		t.Fatal(err)
	}
	// פעם שנייה — לא דורס
	data1, _ := os.ReadFile(main)
	main2, err := EnsureMain(dir)
	if err != nil || main2 != main {
		t.Fatalf("second EnsureMain: %v %q", err, main2)
	}
	data2, _ := os.ReadFile(main)
	if string(data1) != string(data2) {
		t.Fatal("EnsureMain overwrote existing main")
	}

	got, err := ResolveEntry(dir)
	if err != nil || got != main {
		t.Fatalf("ResolveEntry dir: %v %q", err, got)
	}
	got, err = ResolveEntry(main)
	if err != nil || got != main {
		t.Fatalf("ResolveEntry file: %v %q", err, got)
	}
}

func TestResolveEntryMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveEntry(dir)
	if err == nil {
		t.Fatal("expected error for empty project dir")
	}
}

func TestFindProjectRoot(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, MainFileName), []byte("הדפס: 1\n"), 0644)
	sub := filepath.Join(dir, "תת")
	_ = os.Mkdir(sub, 0755)
	nested := filepath.Join(sub, "מודול.יוד")
	_ = os.WriteFile(nested, []byte("הדפס: 2\n"), 0644)

	if got := FindProjectRoot(nested); got != dir {
		t.Fatalf("from nested file: got %q want %q", got, dir)
	}
	if got := FindProjectRoot(sub); got != dir {
		t.Fatalf("from subdir: got %q want %q", got, dir)
	}
	if got := FindProjectRoot(dir); got != dir {
		t.Fatalf("from root: got %q want %q", got, dir)
	}
	lonely := t.TempDir()
	loneFile := filepath.Join(lonely, "בודד.יוד")
	_ = os.WriteFile(loneFile, []byte("הדפס: 3\n"), 0644)
	if got := FindProjectRoot(loneFile); got != "" {
		t.Fatalf("lonely file should have no project, got %q", got)
	}
}

func TestIsYodSource(t *testing.T) {
	if !IsYodSource("א.יוד") || !IsYodSource("a.yod") || !IsYodSource("A.YOD") {
		t.Fatal("expected yod sources")
	}
	if IsYodSource("a.txt") || IsYodSource("יוד") {
		t.Fatal("expected non-yod")
	}
}

func TestListYodFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, MainFileName), []byte("הדפס: 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "עזר.יוד"), []byte("פונקציה א()\n  החזר 1\nסוף\n"), 0644)
	sub := filepath.Join(dir, "תת")
	_ = os.Mkdir(sub, 0755)
	_ = os.WriteFile(filepath.Join(sub, "מודול.יוד"), []byte("הדפס: 2\n"), 0644)

	files, err := ListYodFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %v", files)
	}
}

func TestDefaultAppName(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "גיבוי")
	_ = os.Mkdir(proj, 0755)
	main := filepath.Join(proj, MainFileName)
	_ = os.WriteFile(main, []byte("הדפס: 1\n"), 0644)
	if got := DefaultAppName(main); got != "גיבוי" {
		t.Fatalf("project main: got %q want גיבוי", got)
	}
	exe := DefaultEXEPath(main)
	if filepath.Base(exe) != "גיבוי.exe" {
		t.Fatalf("DefaultEXEPath base: %q", filepath.Base(exe))
	}
	lonely := filepath.Join(dir, "סנייק.יוד")
	_ = os.WriteFile(lonely, []byte("הדפס: 1\n"), 0644)
	if got := DefaultAppName(lonely); got != "סנייק" {
		t.Fatalf("single file: got %q want סנייק", got)
	}
	if SanitizeAppName(`a:b*c`) != "a_b_c" {
		t.Fatalf("sanitize: %q", SanitizeAppName(`a:b*c`))
	}
}
