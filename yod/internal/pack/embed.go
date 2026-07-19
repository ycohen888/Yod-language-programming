package pack

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/project"
)

// סיומת הזנב: [קוד UTF-8][8 בתים אורך][8 בתים קסם]
const (
	embedMagic   = "YODPACK1"
	embedMetaLen = 16
)

// EXEOptions אפשרויות לאריזת EXE.
type EXEOptions struct {
	// Console=true משאיר חלון CMD (מתאים לתוכניות הדפסה בלבד).
	// ברירת מחדל false — בלי חלון קונסול (מתאים ל־חלונות).
	Console bool
	// SingleScript=true — פורמט ישן YODPACK1 (סקריפט כניסה בלבד). ברירת מחדל: YODBUND1.
	SingleScript bool
}

// EXE יוצר קובץ exe בודד: מנוע יוד + חבילת קבצים מוטמעת (YODBUND1).
// אם outPath ריק — נוצר <שם>.exe ליד קובץ המקור.
func EXE(srcPath string, outPath string) (string, error) {
	return EXEWithOptions(srcPath, outPath, EXEOptions{})
}

// EXEWithOptions כמו EXE, עם שליטה על חלון הקונסול ופורמט.
func EXEWithOptions(srcPath string, outPath string, opts EXEOptions) (string, error) {
	srcPath = filepath.Clean(srcPath)
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("לא הצלחתי לקרוא את %s: %v", srcPath, err)
	}

	appName := project.DefaultAppName(srcPath)
	if outPath == "" {
		outPath = filepath.Join(project.DefaultDistDir(srcPath), appName+".exe")
	} else if strings.ToLower(filepath.Ext(outPath)) != ".exe" {
		info, statErr := os.Stat(outPath)
		if statErr == nil && info.IsDir() {
			outPath = filepath.Join(outPath, appName+".exe")
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

	sub := uint16(subsystemGUI)
	if opts.Console {
		sub = subsystemConsole
	}
	if err := setPESubsystem(engine, sub); err != nil {
		return "", fmt.Errorf("לא הצלחתי להגדיר מצב חלון: %v", err)
	}

	// כותבים קודם רק את המנוע — מטמיעים איקון ב־PE, ורק אז מוסיפים overlay
	if err := os.WriteFile(outPath, engine, 0755); err != nil {
		return "", fmt.Errorf("לא הצלחתי ליצור %s: %v", outPath, err)
	}

	icoPath, icoTemp, icoErr := resolvePackIcon(srcPath)
	if icoErr != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("הכנת איקון נכשלה: %v", icoErr)
	}
	if icoTemp {
		defer os.Remove(icoPath)
	}
	if icoPath != "" {
		if err := applyIconToEXE(outPath, icoPath); err != nil {
			// לא נכשלים על איקון — EXE עדיין רץ עם איקון יוד המקורי
			_ = err
		}
	}

	out, err := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("לא הצלחתי לפתוח %s לכתיבת חבילה: %v", outPath, err)
	}

	if opts.SingleScript {
		script, err := os.ReadFile(srcPath)
		if err != nil {
			_ = out.Close()
			_ = os.Remove(outPath)
			return "", fmt.Errorf("לא הצלחתי לקרוא את %s: %v", srcPath, err)
		}
		script = rewriteSiblingIncludes(script)
		if _, err := out.Write(script); err != nil {
			_ = out.Close()
			_ = os.Remove(outPath)
			return "", err
		}
		var meta [embedMetaLen]byte
		binary.LittleEndian.PutUint64(meta[0:8], uint64(len(script)))
		copy(meta[8:16], embedMagic)
		if _, err := out.Write(meta[:]); err != nil {
			_ = out.Close()
			_ = os.Remove(outPath)
			return "", err
		}
	} else {
		bundle, err := CollectBundle(srcPath)
		if err != nil {
			_ = out.Close()
			_ = os.Remove(outPath)
			return "", fmt.Errorf("איסוף חבילה נכשל: %v", err)
		}
		if err := WriteBundleOverlay(out, bundle.Files, bundle.Entry); err != nil {
			_ = out.Close()
			_ = os.Remove(outPath)
			return "", fmt.Errorf("כתיבת חבילה נכשלה: %v", err)
		}
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(outPath)
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

	_ = copyProjectIcon(srcPath, filepath.Dir(outPath))

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
	return stripOverlay(data), nil
}
