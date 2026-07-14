//go:build windows

package stdlib

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"yod/internal/object"
)

const (
	sndAsync    = 0x0001
	sndFilename = 0x00020000
	sndNoDefault = 0x0002
)

var (
	winmm            = windows.NewLazySystemDLL("winmm.dll")
	procPlaySoundW   = winmm.NewProc("PlaySoundW")
	procMciSendStringW = winmm.NewProc("mciSendStringW")
)

func mciSend(cmd string) error {
	r, _, _ := procMciSendStringW.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(cmd))),
		0, 0, 0,
	)
	if r != 0 {
		return fmt.Errorf("MCI שגיאה %d: %s", r, cmd)
	}
	return nil
}

func soundPlayEffect(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שמע.נגן_אפקט מצפה לנתיב אחד")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("שמע.נגן_אפקט מצפה למחרוזת")
	}
	full := resolveMediaPath(path)
	p, err := windows.UTF16PtrFromString(full)
	if err != nil {
		return errObj("נתיב לא תקין")
	}
	r, _, _ := procPlaySoundW.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(sndAsync|sndFilename|sndNoDefault))
	if r == 0 {
		return errObj("לא הצלחתי לנגן אפקט: " + full)
	}
	return &object.Null{}
}

func soundPlay(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שמע.נגן מצפה לנתיב אחד")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("שמע.נגן מצפה למחרוזת")
	}
	full := resolveMediaPath(path)
	soundMu.Lock()
	defer soundMu.Unlock()
	_ = mciSend("close " + soundAlias)
	// MCI דורש נתיב במרכאות אם יש רווחים
	escaped := strings.ReplaceAll(full, `"`, ``)
	cmd := fmt.Sprintf(`open "%s" type mpegvideo alias %s`, escaped, soundAlias)
	ext := strings.ToLower(filepath.Ext(full))
	if ext == ".wav" {
		cmd = fmt.Sprintf(`open "%s" type waveaudio alias %s`, escaped, soundAlias)
	}
	if err := mciSend(cmd); err != nil {
		// ניסיון כללי
		cmd = fmt.Sprintf(`open "%s" alias %s`, escaped, soundAlias)
		if err2 := mciSend(cmd); err2 != nil {
			return errObj("לא הצלחתי לפתוח קובץ שמע: " + full)
		}
	}
	soundOpenPath = full
	play := "play " + soundAlias
	if soundLooping {
		play += " repeat"
	}
	if err := mciSend(play); err != nil {
		return errObj("לא הצלחתי לנגן: " + err.Error())
	}
	_ = mciSend(fmt.Sprintf("setaudio %s volume to %d", soundAlias, soundVol*10)) // 0–1000
	if soundOnEndFn != nil && !soundLooping {
		go watchSoundEnd()
	}
	return &object.Null{}
}

func watchSoundEnd() {
	for {
		time.Sleep(200 * time.Millisecond)
		soundMu.Lock()
		mode := mciStatus("mode")
		fn := soundOnEndFn
		soundMu.Unlock()
		if mode == "stopped" || mode == "" || mode == "not ready" {
			if fn != nil {
				fireSoundEnd(fn)
			}
			return
		}
		if mode == "playing" || mode == "paused" {
			continue
		}
	}
}

func mciStatus(what string) string {
	buf := make([]uint16, 128)
	cmd := fmt.Sprintf("status %s %s", soundAlias, what)
	r, _, _ := procMciSendStringW.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(cmd))),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if r != 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(syscall.UTF16ToString(buf)))
}

func fireSoundEnd(fn object.Object) {
	run := func() { invokeYod(fn, nil) }
	if uiSyncFn != nil {
		uiSyncFn(run)
		return
	}
	run()
}

func soundPause(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שמע.השהה מצפה ל־0 ארגומנטים")
	}
	soundMu.Lock()
	defer soundMu.Unlock()
	_ = mciSend("pause " + soundAlias)
	return &object.Null{}
}

func soundResume(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שמע.המשך מצפה ל־0 ארגומנטים")
	}
	soundMu.Lock()
	defer soundMu.Unlock()
	_ = mciSend("resume " + soundAlias)
	return &object.Null{}
}

func soundStop(args ...object.Object) object.Object {
	soundMu.Lock()
	defer soundMu.Unlock()
	_ = mciSend("stop " + soundAlias)
	_ = mciSend("close " + soundAlias)
	soundOpenPath = ""
	return &object.Null{}
}

func soundLoop(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שמע.לולאה מצפה לבוליאני")
	}
	b, ok := args[0].(*object.Boolean)
	if !ok {
		return errObj("שמע.לולאה מצפה לאמת/שקר")
	}
	soundMu.Lock()
	soundLooping = b.Value
	soundMu.Unlock()
	return &object.Null{}
}

func soundVolume(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שמע.עוצמה מצפה למספר 0–100")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("שמע.עוצמה מצפה למספר")
	}
	v := int(n.Value)
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	soundMu.Lock()
	soundVol = v
	_ = mciSend(fmt.Sprintf("setaudio %s volume to %d", soundAlias, v*10))
	soundMu.Unlock()
	return &object.Null{}
}

func soundIsPlaying(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שמע.מנגן מצפה ל־0 ארגומנטים")
	}
	soundMu.Lock()
	defer soundMu.Unlock()
	return &object.Boolean{Value: mciStatus("mode") == "playing"}
}

func soundOnEnd(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שמע.בעת_סיום מצפה לפונקציה")
	}
	switch args[0].(type) {
	case *object.Function, *object.Closure, *object.CompiledFunction:
		soundMu.Lock()
		soundOnEndFn = args[0]
		soundMu.Unlock()
		return &object.Null{}
	default:
		return errObj("שמע.בעת_סיום מצפה לפונקציה")
	}
}
