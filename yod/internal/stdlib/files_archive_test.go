package stdlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yod/internal/object"
)

func TestZipArchiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("שלום"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("עולם"), 0644); err != nil {
		t.Fatal(err)
	}

	mod := NewFilesModule()
	zipPath := filepath.Join(dir, "pack.zip")
	out := callFiles(t, mod, "ארוז",
		&object.String{Value: srcDir},
		&object.String{Value: zipPath},
	)
	if _, ok := out.(*object.Error); ok {
		t.Fatalf("ארוז נכשל: %v", out.Inspect())
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("קובץ ZIP לא נוצר: %v", err)
	}

	dest := filepath.Join(dir, "out")
	out = callFiles(t, mod, "חלץ",
		&object.String{Value: zipPath},
		&object.String{Value: dest},
	)
	if _, ok := out.(*object.Error); ok {
		t.Fatalf("חלץ נכשל: %v", out.Inspect())
	}

	got, err := os.ReadFile(filepath.Join(dest, "src", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "שלום" {
		t.Fatalf("תוכן a.txt: %q", got)
	}
	got, err = os.ReadFile(filepath.Join(dest, "src", "sub", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "עולם" {
		t.Fatalf("תוכן b.txt: %q", got)
	}
}

func TestArchiveContentsTable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("שלום יוד"), 0644); err != nil {
		t.Fatal(err)
	}
	mod := NewFilesModule()
	zipPath := filepath.Join(dir, "list.zip")
	out := callFiles(t, mod, "ארוז",
		&object.String{Value: file},
		&object.String{Value: zipPath},
	)
	if e, ok := out.(*object.Error); ok {
		t.Fatalf("ארוז: %s", e.Message)
	}
	entries, err := listZIP(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	table := formatArchiveTable(zipPath, entries)
	if !strings.Contains(table, "note.txt") {
		t.Fatalf("טבלה חסרה שם קובץ:\n%s", table)
	}
	if !strings.Contains(table, "גודל") || !strings.Contains(table, "דחוס") {
		t.Fatalf("טבלה חסרה כותרות:\n%s", table)
	}
	if !strings.Contains(table, "סה״כ") {
		t.Fatalf("טבלה חסרה סיכום:\n%s", table)
	}
	out = callFiles(t, mod, "תוכן_ארכיון", &object.String{Value: zipPath})
	if e, ok := out.(*object.Error); ok {
		t.Fatalf("תוכן_ארכיון: %s", e.Message)
	}
}

func TestZipArchiveSingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("יוד"), 0644); err != nil {
		t.Fatal(err)
	}
	mod := NewFilesModule()
	zipPath := filepath.Join(dir, "one.zip")
	out := callFiles(t, mod, "ארוז",
		&object.String{Value: file},
		&object.String{Value: zipPath},
	)
	if e, ok := out.(*object.Error); ok {
		t.Fatalf("ארוז: %s", e.Message)
	}
	dest := filepath.Join(dir, "extracted")
	out = callFiles(t, mod, "חלץ",
		&object.String{Value: zipPath},
		&object.String{Value: dest},
	)
	if e, ok := out.(*object.Error); ok {
		t.Fatalf("חלץ: %s", e.Message)
	}
	got, err := os.ReadFile(filepath.Join(dest, "note.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "יוד" {
		t.Fatalf("got %q", got)
	}
}

func callFiles(t *testing.T, mod *object.Module, name string, args ...object.Object) object.Object {
	t.Helper()
	b, ok := mod.Attrs[name].(*object.Builtin)
	if !ok {
		t.Fatalf("%s אינו Builtin", name)
	}
	return b.Fn(args...)
}
