//go:build windows

package editor

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/jchv/go-webview2"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"

	"yod/internal/version"
)

var (
	guideMu   sync.Mutex
	guideOpen bool
	guideHWND uintptr
)

func findGuideHTML() string {
	rel := filepath.Join("מדריך שפת יוד", "מדריך שפת יוד.html")
	var cands []string
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, rel),
			filepath.Join(dir, "..", rel),
			filepath.Join(dir, "..", "..", rel),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands,
			filepath.Join(wd, rel),
			filepath.Join(wd, "..", rel),
			filepath.Join(wd, "..", "..", rel),
		)
	}
	seen := map[string]bool{}
	for _, c := range cands {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
			return abs
		}
	}
	return ""
}

func fileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)
	slash := filepath.ToSlash(abs)
	if len(slash) >= 2 && slash[1] == ':' {
		// Windows: F:/foo → file:///F:/foo (עם אחוזי־קידוד לעברית/רווחים)
		u := &url.URL{Scheme: "file", Path: "/" + slash}
		return u.String()
	}
	u := &url.URL{Scheme: "file", Path: slash}
	return u.String()
}

func showAboutDialog(owner walk.Form) {
	var dlg *walk.Dialog
	text := "יוד (Yod) היא שפת תכנות מודרנית בעברית.\n\n" +
		"מילות מפתח, ספריות, שגיאות ומדריך — בעברית, מימין לשמאל.\n" +
		"כותבים קבצי .יוד, מריצים במפרש או במכונה, ועורכים בעורך המובנה.\n\n" +
		"פרויקט = תיקייה עם קובץ ראשי התחל.יוד; קבצים נוספים נכללים עם כלול.\n" +
		"אפשר גם תוכנית בקובץ יחיד.\n\n" +
		"גרסה: " + version.String + "\n" +
		"סיומת קבצים: .יוד"

	_, _ = Dialog{
		AssignTo:           &dlg,
		Title:              "אודות יוד",
		MinSize:            Size{Width: 460, Height: 340},
		Layout:             VBox{Margins: Margins{Left: 18, Right: 18, Top: 16, Bottom: 14}, Spacing: 12},
		RightToLeftReading: true,
		Children: []Widget{
			Label{
				Text:               "יוד",
				Font:               Font{Family: "Assistant", PointSize: 22, Bold: true},
				TextColor:          walk.RGB(47, 212, 194),
				RightToLeftReading: true,
			},
			Label{
				Text:               "שפת תכנות בעברית · גרסה " + version.String,
				TextColor:          walk.RGB(154, 171, 192),
				Font:               Font{Family: "Assistant", PointSize: 10},
				RightToLeftReading: true,
			},
			TextEdit{
				Text:               text,
				ReadOnly:           true,
				VScroll:            true,
				RightToLeftReading: true,
				MinSize:            Size{Height: 160},
				Font:               Font{Family: "Assistant", PointSize: 10},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "סגור", OnClicked: func() { dlg.Accept() }},
				},
			},
		},
	}.Run(owner)
}

func focusGuideWindow() bool {
	guideMu.Lock()
	hwnd := guideHWND
	open := guideOpen
	guideMu.Unlock()
	if !open || hwnd == 0 {
		return false
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	showWindow := user32.NewProc("ShowWindow")
	setForeground := user32.NewProc("SetForegroundWindow")
	const swRestore = 9
	_, _, _ = showWindow.Call(hwnd, swRestore)
	_, _, _ = setForeground.Call(hwnd)
	return true
}

func guideDataPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "Yod", "guide-webview")
}

func openGuideExternal(fileURLStr, path string) bool {
	edgeCands := []string{
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
	}
	for _, edge := range edgeCands {
		if _, err := os.Stat(edge); err != nil {
			continue
		}
		cmd := exec.Command(edge, "--app="+fileURLStr)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return true
		}
	}
	// דפדפן ברירת מחדל
	cmd := exec.Command("cmd", "/c", "start", "", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start() == nil
}

func showGuideWindow(owner walk.Form) {
	path := findGuideHTML()
	if path == "" {
		walk.MsgBox(owner, "מדריך",
			"לא נמצא קובץ המדריך.\nחפשו: מדריך שפת יוד/מדריך שפת יוד.html\nליד קובץ ההפעלה או בשורש הריפו.",
			walk.MsgBoxIconWarning)
		return
	}
	homeURL := fileURL(path)

	if focusGuideWindow() {
		return
	}

	guideMu.Lock()
	if guideOpen {
		guideMu.Unlock()
		_ = focusGuideWindow()
		return
	}
	guideOpen = true
	guideMu.Unlock()

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer func() {
			guideMu.Lock()
			guideOpen = false
			guideHWND = 0
			guideMu.Unlock()
		}()

		_ = os.MkdirAll(guideDataPath(), 0o755)

		w := webview2.NewWithOptions(webview2.WebViewOptions{
			Debug:     false,
			AutoFocus: true,
			DataPath:  guideDataPath(),
			WindowOptions: webview2.WindowOptions{
				Title:  "מדריך שפת יוד · " + version.String,
				Width:  1100,
				Height: 760,
				Center: true,
			},
		})
		if w == nil {
			if !openGuideExternal(homeURL, path) {
				if owner != nil {
					owner.Synchronize(func() {
						walk.MsgBox(owner, "מדריך",
							"לא ניתן לפתוח את המדריך.\nהתקינו את WebView2 Runtime או Edge, או פתחו ידנית את קובץ ה־HTML.",
							walk.MsgBoxIconWarning)
					})
				}
			}
			return
		}

		guideMu.Lock()
		guideHWND = uintptr(w.Window())
		guideMu.Unlock()

		defer w.Destroy()
		w.SetSize(1100, 760, webview2.HintNone)
		w.Navigate(homeURL)
		w.Run()
	}()
}
