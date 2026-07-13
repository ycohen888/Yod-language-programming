package materialicons

import (
	_ "embed"
	"sync"
	"syscall"
	"unsafe"
)

//go:embed MaterialIcons-Regular.ttf
var ttf []byte

const Family = "Material Icons"

// קודפוינטים מ־MaterialIcons-Regular.codepoints
const (
	PlayArrow    = '\uE037' // play_arrow
	Memory       = '\uE322' // memory
	Done         = '\uE876' // done
	Build        = '\uE869' // build
	NoteAdd      = '\uE89C' // note_add
	Archive      = '\uE149' // archive
	Folder       = '\uE2C7' // folder
	FolderOpen   = '\uE2C8' // folder_open
	Save         = '\uE161' // save
	Highlight    = '\uE25F' // highlight
	ErrorOutline = '\uE001' // error_outline
	Terminal     = '\uEB8E' // terminal
	FormatIndent = '\uE23E' // format_indent_increase
	InsertFile   = '\uE24D' // insert_drive_file — כללי
	Image        = '\uE3F4' // image
	Code         = '\uE86F' // code
	Description  = '\uE873' // description — טקסט
	Movie        = '\uE02C' // movie
	Audiotrack   = '\uE3A1' // audiotrack
	Settings     = '\uE8B8' // settings
	PicturePDF   = '\uE415' // picture_as_pdf
	TableChart   = '\uE265' // table_chart
)

var (
	once   sync.Once
	loaded bool
)

var (
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	procAddFontMemResource = gdi32.NewProc("AddFontMemResourceEx")
)

// Ensure טוען את הפונט לזיכרון התהליך (פעם אחת)
func Ensure() bool {
	once.Do(func() {
		if len(ttf) == 0 {
			return
		}
		var num uint32
		r, _, _ := procAddFontMemResource.Call(
			uintptr(unsafe.Pointer(&ttf[0])),
			uintptr(len(ttf)),
			0,
			uintptr(unsafe.Pointer(&num)),
		)
		loaded = r != 0 && num > 0
	})
	return loaded
}

// Glyph מחזיר מחרוזת עם תו האיקון
func Glyph(r rune) string {
	return string(r)
}
