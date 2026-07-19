package vfs

import (
	"path/filepath"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		`עיצוב\קליפה.html`:   "עיצוב/קליפה.html",
		`./עיצוב/../מדריך/a`: "מדריך/a",
		`\\עיצוב\\x`:         "עיצוב/x",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", in, got, want)
		}
	}
}

func TestReadExistsIsDir(t *testing.T) {
	root := t.TempDir()
	fs := New(root)
	fs.Add("התחל.יוד", []byte("הדפס: 1\n"))
	fs.Add("עיצוב/קליפה.html", []byte("<html/>"))
	fs.Add("מדריך/מדריך.html", []byte("<h1/>"))

	abs := filepath.Join(root, "עיצוב", "קליפה.html")
	data, ok := fs.Read(abs)
	if !ok || string(data) != "<html/>" {
		t.Fatalf("Read abs: ok=%v data=%q", ok, data)
	}
	if !fs.Exists(filepath.Join(root, "עיצוב")) {
		t.Fatal("עיצוב should exist")
	}
	if !fs.IsDir(filepath.Join(root, "עיצוב")) {
		t.Fatal("עיצוב should be dir")
	}
	if !fs.Exists(filepath.Join(root, "התחל.יוד")) {
		t.Fatal("entry should exist")
	}
	names, ok := fs.List(filepath.Join(root, "עיצוב"))
	if !ok || len(names) != 1 || names[0] != "קליפה.html" {
		t.Fatalf("List=%v ok=%v", names, ok)
	}
}

func TestMountReadPrefer(t *testing.T) {
	Clear()
	defer Clear()
	root := t.TempDir()
	fs := New(root)
	fs.Add("a.txt", []byte("from-vfs"))
	Mount(fs)
	data, err := ReadPrefer(filepath.Join(root, "a.txt"))
	if err != nil || string(data) != "from-vfs" {
		t.Fatalf("got %q err=%v", data, err)
	}
}
