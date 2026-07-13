//go:build windows

package editor

import (
	"testing"

	materialicons "yod/MaterialIcons"
)

func TestClassifyFileKind(t *testing.T) {
	cases := []struct {
		name string
		dir  bool
		want int
	}{
		{"src", true, fileKindDir},
		{"התחל.יוד", false, fileKindYod},
		{"a.yod", false, fileKindYod},
		{"app.ico", false, fileKindImage},
		{"shot.png", false, fileKindImage},
		{"readme.md", false, fileKindText},
		{"main.go", false, fileKindCode},
		{"pack.zip", false, fileKindArchive},
		{"song.mp3", false, fileKindAudio},
		{"clip.mp4", false, fileKindVideo},
		{"doc.pdf", false, fileKindPDF},
		{"data.bin", false, fileKindUnknown},
		{"mystery", false, fileKindUnknown},
	}
	for _, c := range cases {
		if got := classifyFileKind(c.name, c.dir); got != c.want {
			t.Fatalf("%s: kind=%d want %d", c.name, got, c.want)
		}
	}
}

func TestLooksBinary(t *testing.T) {
	if looksBinary([]byte("שלום יוד\n")) {
		t.Fatal("עברית לא אמורה להיחשב בינארי")
	}
	if !looksBinary([]byte{0x00, 0x01, 0x02, 'a'}) {
		t.Fatal("null-byte חייב בינארי")
	}
	// כותרת ICO טיפוסית
	ico := []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}
	if !looksBinary(ico) {
		t.Fatal("ico header אמור להיות בינארי")
	}
}

func TestFileKindIconUnknown(t *testing.T) {
	r, _ := fileKindIcon(fileKindUnknown, false)
	if r != materialicons.InsertFile {
		t.Fatalf("איקון כללי: got %U want InsertFile", r)
	}
}
