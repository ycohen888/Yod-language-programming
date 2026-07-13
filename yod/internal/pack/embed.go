package pack

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// סיומת הזנב: [קוד UTF-8][8 בתים אורך][8 בתים קסם]
const (
	embedMagic   = "YODPACK1"
	embedMetaLen = 16
)

// ReadEmbedded קורא תוכנית יוד שמוטמעת בסוף קובץ EXE (אם קיימת).
func ReadEmbedded(exePath string) (source string, ok bool, err error) {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return "", false, err
	}
	script, ok := parseOverlay(data)
	if !ok {
		return "", false, nil
	}
	return string(script), true, nil
}

// EXEOptions אפשרויות לאריזת EXE.
type EXEOptions struct {
	// Console=true משאיר חלון CMD (מתאים לתוכניות הדפסה בלבד).
	// ברירת מחדל false — בלי חלון קונסול (מתאים ל־חלונות).
	Console bool
}

// EXE יוצר קובץ exe בודד: מנוע יוד + קוד המקור מוטמע בסוף.
// אם outPath ריק — נוצר <שם>.exe ליד קובץ המקור.
func EXE(srcPath string, outPath string) (string, error) {
	return EXEWithOptions(srcPath, outPath, EXEOptions{})
}

// EXEWithOptions כמו EXE, עם שליטה על חלון הקונסול.
func EXEWithOptions(srcPath string, outPath string, opts EXEOptions) (string, error) {
	srcPath = filepath.Clean(srcPath)
	script, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי לקרוא את %s: %v", srcPath, err)
	}

	base := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	if base == "" {
		base = "תוכנית"
	}
	if outPath == "" {
		outPath = filepath.Join(filepath.Dir(srcPath), base+".exe")
	} else if strings.ToLower(filepath.Ext(outPath)) != ".exe" {
		info, statErr := os.Stat(outPath)
		if statErr == nil && info.IsDir() {
			outPath = filepath.Join(outPath, base+".exe")
		} else if filepath.Ext(outPath) == "" {
			outPath += ".exe"
		}
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil && filepath.Dir(outPath) != "." {
		return "", fmt.Errorf("לא הצלחתי ליצור תיקייה: %v", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי למצוא את yod.exe: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	engine, err := engineBytes(exePath)
	if err != nil {
		return "", err
	}

	// ברירת מחדל: בלי CMD. --קונסול משאיר קונסול.
	sub := uint16(subsystemGUI)
	if opts.Console {
		sub = subsystemConsole
	}
	if err := setPESubsystem(engine, sub); err != nil {
		return "", fmt.Errorf("לא הצלחתי להגדיר מצב חלון: %v", err)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי ליצור %s: %v", outPath, err)
	}
	defer out.Close()

	if _, err := out.Write(engine); err != nil {
		return "", err
	}
	if _, err := out.Write(script); err != nil {
		return "", err
	}
	var meta [embedMetaLen]byte
	binary.LittleEndian.PutUint64(meta[0:8], uint64(len(script)))
	copy(meta[8:16], embedMagic)
	if _, err := out.Write(meta[:]); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}

	manifestDest := outPath + ".manifest"
	wroteManifest := false
	for _, m := range []string{
		exePath + ".manifest",
		filepath.Join(filepath.Dir(exePath), "yod.exe.manifest"),
		"yod.exe.manifest",
	} {
		if _, err := os.Stat(m); err == nil {
			if copyFile(m, manifestDest) == nil {
				wroteManifest = true
				break
			}
		}
	}
	if !wroteManifest {
		_ = os.WriteFile(manifestDest, []byte(DefaultManifest), 0644)
	}

	// איקון ליד ה־EXE — החלון והפס משימות יטענו אותו אוטומטית
	_ = copyProjectIcon(filepath.Dir(srcPath), filepath.Dir(outPath))

	return outPath, nil
}

func parseOverlay(data []byte) ([]byte, bool) {
	if len(data) < embedMetaLen {
		return nil, false
	}
	if string(data[len(data)-8:]) != embedMagic {
		return nil, false
	}
	scriptLen := binary.LittleEndian.Uint64(data[len(data)-16 : len(data)-8])
	if scriptLen == 0 || scriptLen > uint64(len(data)-embedMetaLen) {
		return nil, false
	}
	start := len(data) - embedMetaLen - int(scriptLen)
	if start < 0 {
		return nil, false
	}
	return data[start : start+int(scriptLen)], true
}

// engineBytes מחזיר את בתאי ה־PE בלי הטמעה קודמת (אם קיימת).
func engineBytes(exePath string) ([]byte, error) {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return nil, fmt.Errorf("העתקת מנוע נכשלה: %v", err)
	}
	if script, ok := parseOverlay(data); ok {
		return data[:len(data)-embedMetaLen-len(script)], nil
	}
	return data, nil
}
