//go:build windows

package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	materialicons "yod/MaterialIcons"
	"yod/internal/project"

	"github.com/lxn/walk"
)

// סוגי קובץ לתצוגה/פתיחה בסייר
const (
	fileKindDir = iota
	fileKindYod
	fileKindImage
	fileKindText
	fileKindCode
	fileKindArchive
	fileKindAudio
	fileKindVideo
	fileKindPDF
	fileKindData
	fileKindUnknown
)

const maxEditorOpenBytes = 2 * 1024 * 1024

var (
	colIconFolder  = walk.RGB(220, 180, 90)
	colIconYod     = walk.RGB(78, 201, 176)
	colIconImage   = walk.RGB(120, 170, 230)
	colIconText    = walk.RGB(180, 185, 190)
	colIconCode    = walk.RGB(150, 200, 160)
	colIconArchive = walk.RGB(200, 160, 120)
	colIconMedia   = walk.RGB(190, 140, 210)
	colIconPDF     = walk.RGB(220, 120, 110)
	colIconData    = walk.RGB(140, 190, 200)
	colIconUnknown = walk.RGB(140, 145, 150)
)

func fileExt(name string) string {
	return strings.ToLower(filepath.Ext(name))
}

func classifyFileKind(name string, isDir bool) int {
	if isDir {
		return fileKindDir
	}
	if project.IsYodSource(name) {
		return fileKindYod
	}
	ext := fileExt(name)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".webp", ".svg", ".tif", ".tiff":
		return fileKindImage
	case ".txt", ".md", ".markdown", ".log", ".csv", ".tsv", ".rtf":
		return fileKindText
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rs", ".c", ".h", ".cpp", ".hpp",
		".java", ".cs", ".php", ".rb", ".swift", ".kt", ".html", ".htm", ".css", ".scss",
		".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".bat", ".cmd", ".ps1",
		".sh", ".sql", ".json":
		return fileKindCode
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2":
		return fileKindArchive
	case ".mp3", ".wav", ".ogg", ".flac", ".m4a", ".wma":
		return fileKindAudio
	case ".mp4", ".avi", ".mkv", ".mov", ".wmv", ".webm":
		return fileKindVideo
	case ".pdf":
		return fileKindPDF
	case ".xlsx", ".xls", ".ods", ".db", ".sqlite":
		return fileKindData
	default:
		return fileKindUnknown
	}
}

func fileKindIcon(kind int, expanded bool) (rune, walk.Color) {
	switch kind {
	case fileKindDir:
		if expanded {
			return materialicons.FolderOpen, colIconFolder
		}
		return materialicons.Folder, colIconFolder
	case fileKindYod:
		return materialicons.Code, colIconYod
	case fileKindImage:
		return materialicons.Image, colIconImage
	case fileKindText:
		return materialicons.Description, colIconText
	case fileKindCode:
		return materialicons.Terminal, colIconCode
	case fileKindArchive:
		return materialicons.Archive, colIconArchive
	case fileKindAudio:
		return materialicons.Audiotrack, colIconMedia
	case fileKindVideo:
		return materialicons.Movie, colIconMedia
	case fileKindPDF:
		return materialicons.PicturePDF, colIconPDF
	case fileKindData:
		return materialicons.TableChart, colIconData
	default:
		return materialicons.InsertFile, colIconUnknown
	}
}

// looksBinary — null-byte או יותר מדי בתים לא־טקסטואליים.
func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	nonText := 0
	for _, b := range sample {
		if b == 0 {
			return true
		}
		if b < 0x09 || (b > 0x0d && b < 0x20 && b != 0x1b) {
			nonText++
		}
	}
	return nonText*100/len(sample) > 15
}

func isImagePath(path string) bool {
	return classifyFileKind(filepath.Base(path), false) == fileKindImage
}

func isEditorFriendlyKind(kind int) bool {
	return kind == fileKindYod || kind == fileKindText || kind == fileKindCode
}

// openExternally פותח קובץ עם תוכנית ברירת המחדל של Windows.
func openExternally(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", abs).Start()
}

// openFromExplorer — פתיחה בטוחה מהסייר: עורך / תצוגת תמונה / הודעה.
func openFromExplorer(owner walk.Form, path string, openInEditor func(string) error) error {
	path = filepath.Clean(path)
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return nil
	}

	base := filepath.Base(path)
	ext := fileExt(base)
	kind := classifyFileKind(base, false)

	if kind == fileKindImage {
		if err := openExternally(path); err != nil {
			walk.MsgBox(owner, "תמונה",
				"לא ניתן לפתוח את התמונה לתצוגה:\n"+err.Error(),
				walk.MsgBoxIconWarning)
		}
		return nil
	}

	if isEditorFriendlyKind(kind) {
		if fi.Size() > maxEditorOpenBytes {
			walk.MsgBox(owner, "פתיחת קובץ",
				"הקובץ גדול מדי לעורך יוד ("+base+").\nאפשר לפתוח אותו מסייר Windows.",
				walk.MsgBoxIconInformation)
			return nil
		}
		sample, err := peekFile(path, 8192)
		if err != nil {
			return err
		}
		if looksBinary(sample) {
			walk.MsgBox(owner, "פתיחת קובץ",
				"קובץ זה לא נפתח בעורך יוד (נראה כקובץ בינארי):\n"+base,
				walk.MsgBoxIconInformation)
			return nil
		}
		return openInEditor(path)
	}

	kindLabel := "קובץ"
	if ext != "" {
		kindLabel = "קובץ " + ext
	}
	walk.MsgBox(owner, "פתיחת קובץ",
		kindLabel+" זה לא נפתח בעורך יוד.\n"+
			"שם: "+base+"\n\n"+
			"ניתן לפתוח אותו מסייר Windows (תפריט ימני ← הצג בסייר Windows).",
		walk.MsgBoxIconInformation)
	return nil
}

func peekFile(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	got, err := f.Read(buf)
	if err != nil && got == 0 {
		return nil, err
	}
	return buf[:got], nil
}
