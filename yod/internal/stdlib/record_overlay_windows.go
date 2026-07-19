//go:build windows

package stdlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"yod/internal/object"
)

// הקלטה.החל_טקסטים(קלט, פלט, רשימת_מילונים)
// צורב שכבות טקסט לסרטון דרך ffmpeg drawtext (גופן עברי מ־Windows\Fonts).
func recordApplyTexts(args ...object.Object) object.Object {
	if len(args) != 3 {
		return errObj("הקלטה.החל_טקסטים מצפה ל־קלט, פלט, רשימת_טקסטים")
	}
	inPath, ok := asString(args[0])
	if !ok || strings.TrimSpace(inPath) == "" {
		return errObj("הקלטה.החל_טקסטים: נתיב קלט חייב להיות מחרוזת")
	}
	outPath, ok := asString(args[1])
	if !ok || strings.TrimSpace(outPath) == "" {
		return errObj("הקלטה.החל_טקסטים: נתיב פלט חייב להיות מחרוזת")
	}
	arr, ok := args[2].(*object.Array)
	if !ok {
		return errObj("הקלטה.החל_טקסטים: הארגומנט השלישי חייב להיות רשימה של מילונים")
	}
	overlays, err := parseTextOverlays(arr)
	if err != nil {
		return errObj("הקלטה.החל_טקסטים: " + err.Error())
	}

	ff := findFFmpeg()
	if ff == "" {
		return ffmpegMissingErr()
	}
	inPath = filepath.Clean(inPath)
	outPath = filepath.Clean(outPath)
	if st, err := os.Stat(inPath); err != nil || st.IsDir() || st.Size() < 32 {
		return errObj("הקלטה.החל_טקסטים: קובץ הקלט לא נמצא או ריק")
	}

	workDir := filepath.Dir(outPath)
	if workDir == "" || workDir == "." {
		workDir = filepath.Dir(inPath)
	}
	_ = os.MkdirAll(workDir, 0o755)

	sameFile := strings.EqualFold(filepath.Clean(inPath), filepath.Clean(outPath))
	target := outPath
	if sameFile {
		target = outPath + ".yodtxt.mp4"
	}

	filter, cleanup, err := buildDrawtextFilter(overlays, workDir)
	defer func() {
		for _, p := range cleanup {
			_ = os.Remove(p)
		}
	}()
	if err != nil {
		return errObj("הקלטה.החל_טקסטים: " + err.Error())
	}

	_ = os.Remove(target)
	var stderr bytes.Buffer
	cmd := exec.Command(ff,
		"-y", "-i", inPath,
		"-vf", filter,
		"-c:v", "libx264", "-preset", "fast", "-crf", "20",
		"-c:a", "aac", "-b:a", "192k",
		"-movflags", "+faststart",
		target,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// ניסיון בלי אודיו (הקלטות ללא מיקרופון)
		stderr.Reset()
		_ = os.Remove(target)
		cmd2 := exec.Command(ff,
			"-y", "-i", inPath,
			"-vf", filter,
			"-c:v", "libx264", "-preset", "fast", "-crf", "20",
			"-an",
			"-movflags", "+faststart",
			target,
		)
		cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd2.Stdout = io.Discard
		cmd2.Stderr = &stderr
		if err2 := cmd2.Run(); err2 != nil {
			msg := strings.TrimSpace(stderr.String())
			if len(msg) > 400 {
				msg = msg[len(msg)-400:]
			}
			return errObj(fmt.Sprintf("הקלטה.החל_טקסטים נכשל: %v — %s", err2, msg))
		}
	}

	tst, err := os.Stat(target)
	if err != nil || tst.Size() < 32 {
		_ = os.Remove(target)
		return errObj("הקלטה.החל_טקסטים: קובץ הפלט לא נוצר כראוי")
	}

	if sameFile {
		_ = os.Remove(outPath)
		if err := os.Rename(target, outPath); err != nil {
			alt := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + "_טקסט" + filepath.Ext(outPath)
			_ = os.Rename(target, alt)
			return &object.String{Value: alt}
		}
	}
	return &object.String{Value: outPath}
}

func recordDefaultFont(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.גופן_ברירת_מחדל מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: findHebrewFont("")}
}
