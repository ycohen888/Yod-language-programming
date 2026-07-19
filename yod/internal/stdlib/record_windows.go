//go:build windows

package stdlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"yod/internal/object"
)

func recordListMics(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.רשימת_מיקרופונים מצפה ל־0 ארגומנטים")
	}
	ff := findFFmpeg()
	if ff == "" {
		return ffmpegMissingErr()
	}
	cmd := exec.Command(ff, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()
	names := parseDShowAudioDevices(stderr.String())
	arr := &object.Array{Elements: make([]object.Object, 0, len(names))}
	for _, n := range names {
		arr.Elements = append(arr.Elements, &object.String{Value: n})
	}
	return arr
}

func recordStartScreen(args ...object.Object) object.Object {
	if len(args) < 5 {
		return errObj("הקלטה.התחל_מסך מצפה ל־נתיב, x, y, רוחב, גובה [, אפשרויות]")
	}
	outPath, ok := asString(args[0])
	if !ok || strings.TrimSpace(outPath) == "" {
		return errObj("הקלטה.התחל_מסך: נתיב חייב להיות מחרוזת")
	}
	x, ok1 := asNumber(args[1])
	y, ok2 := asNumber(args[2])
	w, ok3 := asNumber(args[3])
	h, ok4 := asNumber(args[4])
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return errObj("הקלטה.התחל_מסך: x,y,רוחב,גובה חייבים להיות מספרים")
	}
	opts := map[string]object.Object{}
	if len(args) >= 6 {
		if args[5] != nil && args[5].Type() != object.NullObj {
			hsh, ok := args[5].(*object.Hash)
			if !ok {
				return errObj("הקלטה.התחל_מסך: אפשרויות חייבות להיות מילון")
			}
			opts = hsh.Pairs
		}
	}
	if len(args) > 6 {
		return errObj("הקלטה.התחל_מסך מצפה לכל היותר 6 ארגומנטים")
	}

	ff := findFFmpeg()
	if ff == "" {
		return ffmpegMissingErr()
	}

	xi, yi := int(x), int(y)
	wi, hi := int(w), int(h)
	if wi < 2 || hi < 2 {
		return errObj("הקלטה.התחל_מסך: רוחב וגובה חייבים להיות לפחות 2")
	}
	wi -= wi % 2
	hi -= hi % 2
	if wi < 2 || hi < 2 {
		return errObj("הקלטה.התחל_מסך: אזור קטן מדי אחרי יישור לזוגי")
	}

	fps := 30
	useMic := false
	micDev := ""
	if v, ok := opts["fps"]; ok {
		if n, okn := asNumber(v); okn && n > 0 {
			fps = int(n)
		}
	}
	if v, ok := opts["מיקרופון"]; ok {
		if b, okb := v.(*object.Boolean); okb {
			useMic = b.Value
		}
	}
	if v, ok := opts["התקן"]; ok {
		if s, oks := asString(v); oks {
			micDev = strings.TrimSpace(s)
		}
	}

	// גודל פלט קבוע (אופציונלי): הסרטון תמיד ברזולוציה הזו, אבל הקטע המוקלט
	// נשמר בגודלו האמיתי (בלי הגדלה) וממורכז עם ריפוד — לא נמתח ל"ענק".
	outW, outH := 0, 0
	if v, ok := opts["רוחב_פלט"]; ok {
		if n, okn := asNumber(v); okn && n > 0 {
			outW = int(n)
		}
	}
	if v, ok := opts["גובה_פלט"]; ok {
		if n, okn := asNumber(v); okn && n > 0 {
			outH = int(n)
		}
	}
	outW -= outW % 2
	outH -= outH % 2

	outPath = resolveRecordPath(outPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return errObj("הקלטה.התחל_מסך: לא ניתן ליצור תיקייה: " + err.Error())
	}

	ffArgs := []string{
		"-y",
		"-f", "gdigrab",
		"-framerate", strconv.Itoa(fps),
		"-offset_x", strconv.Itoa(xi),
		"-offset_y", strconv.Itoa(yi),
		"-video_size", fmt.Sprintf("%dx%d", wi, hi),
		"-i", "desktop",
	}
	if useMic {
		if micDev == "" {
			mics := recordListMics()
			if err, isErr := mics.(*object.Error); isErr {
				return err
			}
			arr, _ := mics.(*object.Array)
			if arr == nil || len(arr.Elements) == 0 {
				return errObj("הקלטה.התחל_מסך: לא נמצא מיקרופון. בדוק התקנים או כבה מיקרופון באפשרויות")
			}
			micDev, _ = asString(arr.Elements[0])
		}
		ffArgs = append(ffArgs, "-f", "dshow", "-i", "audio="+micDev)
	}
	// פילטר גודל פלט קבוע: מקטין רק אם הקטע גדול מהמסגרת (min), אף פעם לא מגדיל,
	// ואז מרפד למרכז מסגרת בגודל הקבוע — כך האלמנטים נשמרים בגודלם האמיתי.
	if outW >= 2 && outH >= 2 {
		vf := fmt.Sprintf(
			"scale='min(iw,%d)':'min(ih,%d)':force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1",
			outW, outH, outW, outH,
		)
		ffArgs = append(ffArgs, "-vf", vf)
	}
	ffArgs = append(ffArgs,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		"-g", strconv.Itoa(fps), // keyframe כל שנייה — עמידות לקובץ מקוטע
	)
	if useMic {
		ffArgs = append(ffArgs, "-c:a", "aac", "-b:a", "128k")
	} else {
		ffArgs = append(ffArgs, "-an")
	}
	// MP4 מקוטע: ניתן לנגן גם אחרי סגירה בכוח (בלי moov בסוף הקובץ)
	ffArgs = append(ffArgs,
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		outPath,
	)

	return startFFmpegRecording(ff, ffArgs, outPath)
}

func recordStartAudio(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("הקלטה.התחל_קול מצפה לנתיב [, אפשרויות]")
	}
	outPath, ok := asString(args[0])
	if !ok || strings.TrimSpace(outPath) == "" {
		return errObj("הקלטה.התחל_קול: נתיב חייב להיות מחרוזת")
	}
	opts := map[string]object.Object{}
	if len(args) == 2 && args[1] != nil && args[1].Type() != object.NullObj {
		hsh, ok := args[1].(*object.Hash)
		if !ok {
			return errObj("הקלטה.התחל_קול: אפשרויות חייבות להיות מילון")
		}
		opts = hsh.Pairs
	}

	ff := findFFmpeg()
	if ff == "" {
		return ffmpegMissingErr()
	}

	micDev := ""
	if v, ok := opts["התקן"]; ok {
		if s, oks := asString(v); oks {
			micDev = strings.TrimSpace(s)
		}
	}
	if micDev == "" {
		mics := recordListMics()
		if err, isErr := mics.(*object.Error); isErr {
			return err
		}
		arr, _ := mics.(*object.Array)
		if arr == nil || len(arr.Elements) == 0 {
			return errObj("הקלטה.התחל_קול: לא נמצא מיקרופון")
		}
		micDev, _ = asString(arr.Elements[0])
	}

	outPath = resolveRecordPath(outPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return errObj("הקלטה.התחל_קול: לא ניתן ליצור תיקייה: " + err.Error())
	}

	ext := strings.ToLower(filepath.Ext(outPath))
	ffArgs := []string{"-y", "-f", "dshow", "-i", "audio=" + micDev}
	switch ext {
	case ".wav":
		ffArgs = append(ffArgs, "-c:a", "pcm_s16le", outPath)
	case ".mp3":
		ffArgs = append(ffArgs, "-c:a", "libmp3lame", "-b:a", "192k", outPath)
	default:
		ffArgs = append(ffArgs,
			"-c:a", "aac", "-b:a", "192k",
			"-movflags", "frag_keyframe+empty_moov+default_base_moof",
			outPath,
		)
	}

	return startFFmpegRecording(ff, ffArgs, outPath)
}

func startFFmpegRecording(ff string, ffArgs []string, outPath string) object.Object {
	recordMu.Lock()
	if recordRunning {
		recordMu.Unlock()
		return errObj("הקלטה: כבר מקליטים. קרא להקלטה.עצור() לפני התחלה חדשה")
	}
	// ניקוי שאריות מתהליך קודם
	if recordPID > 0 {
		pid := recordPID
		recordMu.Unlock()
		killPIDTree(pid)
		recordMu.Lock()
		recordPID = 0
	}

	cmd := exec.Command(ff, ffArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		recordMu.Unlock()
		return errObj("הקלטה: לא ניתן לפתוח stdin ל־ffmpeg: " + err.Error())
	}

	// חשוב: לא לחסום על stderr מלא (ffmpeg נתקע ולא מגיב ל־q)
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdin.Close()
		recordMu.Unlock()
		return errObj("הקלטה: לא ניתן לפתוח stderr: " + err.Error())
	}
	cmd.Stdout = stderrW
	cmd.Stderr = stderrW

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stderrW.Close()
		_ = stderrR.Close()
		recordMu.Unlock()
		return errObj("הקלטה: הפעלת ffmpeg נכשלה: " + err.Error())
	}
	_ = stderrW.Close() // הצד של ההורה נסגר; הילד ממשיך לכתוב ל־pipe

	pid := cmd.Process.Pid
	assignFFmpegToJob(pid)

	recordCmd = cmd
	recordStdin = stdin
	recordOutPath = outPath
	recordRunning = true
	recordPID = pid
	recordStderr = ""
	recordMu.Unlock()

	go drainRecordStderr(stderrR)
	go func(c *exec.Cmd, p int) {
		_ = c.Wait()
		recordMu.Lock()
		if recordCmd == c || recordPID == p {
			recordRunning = false
			recordCmd = nil
			recordPID = 0
			if recordStdin != nil {
				_ = recordStdin.Close()
				recordStdin = nil
			}
		}
		recordMu.Unlock()
	}(cmd, pid)

	time.Sleep(500 * time.Millisecond)
	recordMu.Lock()
	still := recordRunning && recordCmd == cmd
	errTail := recordStderr
	recordMu.Unlock()
	if !still {
		msg := strings.TrimSpace(errTail)
		if msg == "" {
			msg = "התהליך הסתיים מיד"
		}
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		return errObj("הקלטה: ffmpeg נכשל להתחיל: " + msg)
	}
	return object.Nil
}

func drainRecordStderr(r *os.File) {
	defer r.Close()
	buf := make([]byte, 4096)
	var ring bytes.Buffer
	for {
		n, err := r.Read(buf)
		if n > 0 {
			ring.Write(buf[:n])
			for ring.Len() > 8192 {
				_ = ring.Next(ring.Len() - 8192)
			}
			recordMu.Lock()
			recordStderr = ring.String()
			recordMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func recordStop(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.עצור מצפה ל־0 ארגומנטים")
	}
	out := stopFFmpegRecording(true)
	if out == "" {
		return object.Nil
	}
	return &object.String{Value: out}
}

// recordStopAll — נקרא בסגירת חלון ראשי.
func recordStopAll() {
	_ = stopFFmpegRecording(true)
}

func stopFFmpegRecording(finalize bool) string {
	recordMu.Lock()
	cmd := recordCmd
	stdin := recordStdin
	running := recordRunning
	out := recordOutPath
	pid := recordPID
	recordMu.Unlock()

	if !running && pid == 0 {
		return ""
	}

	if stdin != nil {
		_, _ = io.WriteString(stdin, "q\n")
		time.Sleep(50 * time.Millisecond)
		_ = stdin.Close()
	}

	// המתנה קצרה ליציאה נקייה
	waitUntilRecordProcessGone(cmd, pid, 800*time.Millisecond)

	// תמיד מוודאים שהתהליך מת — אחרת הקובץ נשאר נעול
	recordMu.Lock()
	still := recordRunning || (recordPID != 0 && recordPID == pid)
	recordMu.Unlock()
	if still || processAlive(pid) {
		killPIDTree(pid)
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitUntilRecordProcessGone(cmd, pid, 3*time.Second)
	}

	// המתנה לשחרור נעילת הקובץ ב־Windows
	if out != "" {
		waitFileUnlocked(out, 5*time.Second)
	}

	recordMu.Lock()
	recordRunning = false
	recordCmd = nil
	recordStdin = nil
	recordPID = 0
	recordMu.Unlock()

	if finalize && out != "" {
		remuxRecordingToCompatibleMP4(out)
	}
	return out
}

func waitUntilRecordProcessGone(cmd *exec.Cmd, pid int, max time.Duration) {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		recordMu.Lock()
		gone := !recordRunning && (recordPID == 0 || recordPID != pid)
		recordMu.Unlock()
		if gone && !processAlive(pid) {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(p)
	var code uint32
	err = windows.GetExitCodeProcess(p, &code)
	if err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

func killPIDTree(pid int) {
	if pid <= 0 {
		return
	}
	k := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	k.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = k.Run()
}

func waitFileUnlocked(path string, max time.Duration) {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			_ = f.Close()
			return
		}
		time.Sleep(80 * time.Millisecond)
	}
}

func remuxRecordingToCompatibleMP4(path string) {
	ff := findFFmpeg()
	if ff == "" {
		return
	}
	waitFileUnlocked(path, 3*time.Second)
	st, err := os.Stat(path)
	if err != nil || st.Size() < 32 {
		return
	}
	tmp := path + ".yodtmp.mp4"
	_ = os.Remove(tmp)
	cmd := exec.Command(ff, "-y", "-i", path, "-c", "copy", "-movflags", "+faststart", tmp)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return
	}
	tst, err := os.Stat(tmp)
	if err != nil || tst.Size() < 32 {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		// אם המחיקה/החלפה נכשלה — משאירים את tmp בשם חלופי
		_ = os.Rename(tmp, strings.TrimSuffix(path, filepath.Ext(path))+"_תקין"+filepath.Ext(path))
	}
}

// Job Object: כשהתהליך של יוד נסגר בכוח — ffmpeg נסגר איתו.
var (
	recordJobOnce windows.Handle
)

func assignFFmpegToJob(pid int) {
	if pid <= 0 {
		return
	}
	if recordJobOnce == 0 {
		name, _ := windows.UTF16PtrFromString("")
		h, err := windows.CreateJobObject(nil, name)
		if err != nil {
			return
		}
		var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		_, err = windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if err != nil {
			_ = windows.CloseHandle(h)
			return
		}
		recordJobOnce = h
	}
	p, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(p)
	_ = windows.AssignProcessToJobObject(recordJobOnce, p)
}
