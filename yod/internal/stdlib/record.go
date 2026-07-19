package stdlib

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"yod/internal/object"
)

func NewRecordModule() *object.Module {
	m := &object.Module{Name: "הקלטה", Attrs: map[string]object.Object{}}
	m.Attrs["זמין"] = &object.Builtin{Fn: recordAvailable}
	m.Attrs["קבע_ffmpeg"] = &object.Builtin{Fn: recordSetFFmpeg}
	m.Attrs["נתיב_ffmpeg"] = &object.Builtin{Fn: recordFFmpegPath}
	m.Attrs["בחר_אזור"] = &object.Builtin{Fn: recordPickRegion}
	m.Attrs["רשימת_מיקרופונים"] = &object.Builtin{Fn: recordListMics}
	m.Attrs["התחל_מסך"] = &object.Builtin{Fn: recordStartScreen}
	m.Attrs["התחל_קול"] = &object.Builtin{Fn: recordStartAudio}
	m.Attrs["עצור"] = &object.Builtin{Fn: recordStop}
	m.Attrs["מקליט"] = &object.Builtin{Fn: recordIsRecording}
	m.Attrs["קובץ_נוכחי"] = &object.Builtin{Fn: recordCurrentFile}
	m.Attrs["החל_טקסטים"] = &object.Builtin{Fn: recordApplyTexts}
	m.Attrs["גופן_ברירת_מחדל"] = &object.Builtin{Fn: recordDefaultFont}
	registerRecordAsync(m)
	return m
}

var (
	recordMu       sync.Mutex
	recordFFmpeg   string // נתיב ידני
	recordCmd      *exec.Cmd
	recordStdin    io.WriteCloser
	recordOutPath  string
	recordRunning  bool
	recordPID      int
	recordStderr   string // דגימת stderr אחרונה (לשגיאות התחלה)
)

func recordAvailable(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.זמין מצפה ל־0 ארגומנטים")
	}
	p := findFFmpeg()
	return &object.Boolean{Value: p != ""}
}

func recordSetFFmpeg(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הקלטה.קבע_ffmpeg מצפה לנתיב אחד")
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("הקלטה.קבע_ffmpeg מצפה למחרוזת")
	}
	s = strings.TrimSpace(s)
	recordMu.Lock()
	defer recordMu.Unlock()
	recordFFmpeg = s
	return object.Nil
}

func recordFFmpegPath(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.נתיב_ffmpeg מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: findFFmpeg()}
}

func recordIsRecording(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.מקליט מצפה ל־0 ארגומנטים")
	}
	recordMu.Lock()
	defer recordMu.Unlock()
	return &object.Boolean{Value: recordRunning}
}

func recordCurrentFile(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.קובץ_נוכחי מצפה ל־0 ארגומנטים")
	}
	recordMu.Lock()
	defer recordMu.Unlock()
	if !recordRunning {
		return object.Nil
	}
	return &object.String{Value: recordOutPath}
}

// findFFmpeg — נתיב ידני → ליד האפליקציה/יוד → PATH.
func findFFmpeg() string {
	recordMu.Lock()
	manual := recordFFmpeg
	recordMu.Unlock()
	if manual != "" {
		if st, err := os.Stat(manual); err == nil && !st.IsDir() {
			return manual
		}
	}
	names := []string{"ffmpeg.exe", "ffmpeg"}
	var dirs []string
	add := func(d string) {
		if d == "" {
			return
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		for _, e := range dirs {
			if e == abs {
				return
			}
		}
		dirs = append(dirs, abs)
	}
	add(AppBaseDir())
	if wd, err := os.Getwd(); err == nil {
		add(wd)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = resolved
		}
		add(filepath.Dir(exe))
	}
	for _, d := range dirs {
		for _, n := range names {
			p := filepath.Join(d, n)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	if p, err := exec.LookPath("ffmpeg.exe"); err == nil {
		return p
	}
	return ""
}

func resolveRecordPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	base := AppBaseDir()
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, path))
}

func ffmpegMissingErr() object.Object {
	return errObj("הקלטה: לא נמצא ffmpeg. התקן ffmpeg והוסף ל־PATH, או שים ffmpeg.exe ליד התוכנה, או קרא להקלטה.קבע_ffmpeg(נתיב)")
}
