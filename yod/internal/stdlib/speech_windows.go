//go:build windows

package stdlib

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"yod/internal/object"
)

var (
	speechMu   sync.Mutex
	speechCmd  *exec.Cmd
	speechRateVal = 0 // SAPI Rate: ‎-10…10
)

func speechSpeak(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("דיבור.הקרא מצפה למחרוזת אחת")
	}
	text, ok := asString(args[0])
	if !ok {
		return errObj("דיבור.הקרא מצפה למחרוזת")
	}
	speechMu.Lock()
	defer speechMu.Unlock()
	speechKillLocked()

	rate := speechRateVal
	if rate < -10 {
		rate = -10
	}
	if rate > 10 {
		rate = 10
	}
	// System.Speech — קול עברי אם מותקן במערכת, אחרת ברירת מחדל.
	ps := fmt.Sprintf(
		`Add-Type -AssemblyName System.Speech; $s = New-Object System.Speech.Synthesis.SpeechSynthesizer; $s.Rate = %d; $s.Speak(%s)`,
		rate,
		psSingleQuoted(text),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return errObj("דיבור.הקרא נכשל: " + err.Error())
	}
	speechCmd = cmd
	go func(c *exec.Cmd) {
		_ = c.Wait()
		speechMu.Lock()
		if speechCmd == c {
			speechCmd = nil
		}
		speechMu.Unlock()
	}(cmd)
	return object.Nil
}

func speechStop(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("דיבור.עצור מצפה ל־0 ארגומנטים")
	}
	speechMu.Lock()
	defer speechMu.Unlock()
	speechKillLocked()
	return object.Nil
}

func speechRate(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("דיבור.קצב מצפה למספר (‎-10 עד 10)")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("דיבור.קצב מצפה למספר")
	}
	v := int(n.Value)
	if v < -10 {
		v = -10
	}
	if v > 10 {
		v = 10
	}
	speechMu.Lock()
	speechRateVal = v
	speechMu.Unlock()
	return object.Nil
}

func speechKillLocked() {
	if speechCmd == nil || speechCmd.Process == nil {
		speechCmd = nil
		return
	}
	_ = speechCmd.Process.Kill()
	speechCmd = nil
}

func psSingleQuoted(s string) string {
	// PowerShell single-quoted literal: ' → ''
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
