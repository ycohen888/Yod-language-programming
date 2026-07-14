//go:build windows

package console

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	user32                         = windows.NewLazySystemDLL("user32.dll")
	procSetConsoleOutputCP         = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP               = kernel32.NewProc("SetConsoleCP")
	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procSetCurrentConsoleFontEx    = kernel32.NewProc("SetCurrentConsoleFontEx")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleWindow           = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procShowWindow            = user32.NewProc("ShowWindow")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
)

const (
	stdOutputHandle           = uint32(0xFFFFFFF5) // -11
	enableVirtualTerminalProc = 0x0004
	enableProcessedOutput     = 0x0001
)

type coord struct {
	X, Y int16
}

type consoleFontInfoEx struct {
	cbSize     uint32
	nFont      uint32
	dwFontSize coord
	FontFamily uint32
	FontWeight uint32
	FaceName   [32]uint16
}

// Init מכין את קונסול Windows לעברית (UTF-8 + גופן מתאים).
func Init() {
	_, _, _ = procSetConsoleOutputCP.Call(uintptr(65001))
	_, _, _ = procSetConsoleCP.Call(uintptr(65001))

	hOut, _, _ := procGetStdHandle.Call(uintptr(stdOutputHandle))
	if hOut == 0 || hOut == uintptr(syscall.InvalidHandle) {
		return
	}

	enableVT(hOut)
	setHebrewFont(hOut)

	// מוודאים ש־stdout/stderr לא במצב broken pipe נדיר
	_ = os.Stdout
	_ = os.Stderr
}

// HideIfOwned מסתיר את חלון ה־CMD רק אם התהליך לבד בקונסול
// (לחיצה כפולה מסייר / קיצור דרך) — לא כשמריצים מתוך טרמינל קיים.
func HideIfOwned() {
	var buf [8]uint32
	r1, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	// r1 = מספר תהליכים המחוברים לקונסול
	if r1 == 0 || r1 > 1 {
		return
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	_, _, _ = procShowWindow.Call(hwnd, 0) // SW_HIDE
	_, _, _ = procFreeConsole.Call()
}

func enableVT(hOut uintptr) {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(hOut, uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return
	}
	mode |= enableVirtualTerminalProc | enableProcessedOutput
	_, _, _ = procSetConsoleMode.Call(hOut, uintptr(mode))
}

func setHebrewFont(hOut uintptr) {
	// Courier New / Lucida Console תומכים בעברית ברוב התקנות Windows
	fonts := []string{"Courier New", "Lucida Console", "Consolas", "Cascadia Mono"}
	for _, name := range fonts {
		if trySetFont(hOut, name) {
			return
		}
	}
}

func trySetFont(hOut uintptr, face string) bool {
	var info consoleFontInfoEx
	info.cbSize = uint32(unsafe.Sizeof(info))
	info.FontWeight = 400
	info.dwFontSize = coord{X: 0, Y: 16}
	info.FontFamily = 54 // FF_MODERN | TMPF_TRUETYPE-ish; Windows מקבל גם 0
	copyUTF16(info.FaceName[:], face)

	r, _, _ := procSetCurrentConsoleFontEx.Call(hOut, 0, uintptr(unsafe.Pointer(&info)))
	return r != 0
}

func copyUTF16(dst []uint16, s string) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return
	}
	n := len(u)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst, u[:n])
}
