//go:build windows

package editor

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yod/internal/version"
)

var guideWindow *walk.MainWindow

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
	abs = filepath.ToSlash(abs)
	if len(abs) >= 2 && abs[1] == ':' {
		// Windows: F:/foo → file:///F:/foo
		return "file:///" + abs
	}
	u := url.URL{Scheme: "file", Path: abs}
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

func showGuideWindow(owner walk.Form) {
	path := findGuideHTML()
	if path == "" {
		walk.MsgBox(owner, "מדריך",
			"לא נמצא קובץ המדריך.\nחפשו: מדריך שפת יוד/מדריך שפת יוד.html\nליד קובץ ההפעלה או בשורש הריפו.",
			walk.MsgBoxIconWarning)
		return
	}
	homeURL := fileURL(path)

	if guideWindow != nil {
		_ = guideWindow.SetFocus()
		guideWindow.Show()
		return
	}

	var mw *walk.MainWindow
	var wv *walk.WebView

	if err := (MainWindow{
		AssignTo:           &mw,
		Title:              "מדריך שפת יוד · " + version.String,
		MinSize:            Size{Width: 920, Height: 640},
		Size:               Size{Width: 1100, Height: 760},
		Layout:             VBox{MarginsZero: true, Spacing: 0},
		RightToLeftReading: true,
		Children: []Widget{
			Composite{
				Layout:     HBox{Margins: Margins{Left: 10, Right: 10, Top: 8, Bottom: 8}, Spacing: 8},
				Background: SolidColorBrush{Color: walk.RGB(15, 21, 32)},
				Children: []Widget{
					Label{
						Text:               "מדריך שפת יוד",
						TextColor:          walk.RGB(47, 212, 194),
						Font:               Font{Family: "Assistant", PointSize: 11, Bold: true},
						RightToLeftReading: true,
					},
					Label{
						Text:      "גרסה " + version.String,
						TextColor: walk.RGB(107, 124, 145),
						Font:      Font{Family: "Assistant", PointSize: 9},
					},
					HSpacer{},
					PushButton{
						Text: "דף הבית",
						OnClicked: func() {
							if wv != nil {
								_ = wv.SetURL(homeURL)
							}
						},
					},
					PushButton{
						Text: "סגור",
						OnClicked: func() {
							if mw != nil {
								mw.Close()
							}
						},
					},
				},
			},
			WebView{
				AssignTo:      &wv,
				URL:           homeURL,
				StretchFactor: 1,
				MinSize:       Size{Height: 400},
			},
		},
	}).Create(); err != nil {
		walk.MsgBox(owner, "מדריך", "לא ניתן לפתוח חלון מדריך:\n"+err.Error(), walk.MsgBoxIconError)
		return
	}

	guideWindow = mw
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		guideWindow = nil
	})
	mw.Show()
	_ = owner // שומר על הקשר לחלון הראשי
}
