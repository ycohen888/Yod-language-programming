package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/console"
	"yod/internal/pack"
	"yod/internal/project"
)

// packApp — ברירת מחדל EXE עצמאי (YODBUND1) ב־dist_exe בלי מקורות; --קונסול / --תיקייה / --רשימה.
func packApp(srcPath string, rest []string) error {
	srcPath, err := project.ResolveEntry(srcPath)
	if err != nil {
		return fmt.Errorf("שגיאה: %v", err)
	}

	folder := false
	keepConsole := false
	listOnly := false
	out := ""
	for _, a := range rest {
		switch a {
		case "--תיקייה", "--folder", "-d":
			folder = true
		case "--קונסול", "--console", "-c":
			keepConsole = true
		case "--רשימה", "--list", "-l":
			listOnly = true
		default:
			if out == "" {
				out = a
			}
		}
	}

	if listOnly {
		keys, entry, err := pack.ListBundlePaths(srcPath)
		if err != nil {
			return err
		}
		console.Printf("כניסה: %s\n", entry)
		console.Printf("קבצים בחבילה (%d):\n", len(keys))
		for _, k := range keys {
			console.Printf("  %s\n", k)
		}
		return nil
	}

	if err := clearOldPackDist(srcPath, out, folder); err != nil {
		return err
	}

	if folder {
		if out == "" {
			out = project.DefaultDistDir(srcPath)
		}
		created, err := pack.App(srcPath, out)
		if err != nil {
			return err
		}
		if err := pack.PrepareCleanProjectDist(srcPath, created); err != nil {
			return fmt.Errorf("הכנת הפצה נקיה נכשלה: %v", err)
		}
		console.Printf("נוצרה תיקיית הפצה נקיה: %s\n", created)
		console.Printf("להרצה: %s\\הרץ.bat\n", created)
		return nil
	}

	if out == "" {
		out = project.DefaultEXEPath(srcPath)
	} else {
		info, statErr := os.Stat(out)
		isDir := statErr == nil && info.IsDir()
		if isDir || strings.EqualFold(filepath.Base(out), project.DistDirName) ||
			(filepath.Ext(out) == "" && !strings.HasSuffix(strings.ToLower(out), ".exe")) {
			base := project.DefaultAppName(srcPath)
			out = filepath.Join(out, base+".exe")
		}
	}

	created, err := pack.EXEWithOptions(srcPath, out, pack.EXEOptions{Console: keepConsole})
	if err != nil {
		return err
	}
	distDir := filepath.Dir(created)
	if err := pack.PrepareBinaryDist(srcPath, distDir); err != nil {
		return fmt.Errorf("הכנת הפצה בינארית נכשלה: %v", err)
	}
	console.Printf("נוצר EXE עצמאי (בלי מקורות): %s\n", created)
	console.Printf("תיקיית הפצה: %s\n", distDir)
	if keepConsole {
		console.Printf("מצב קונסול — יוצג חלון CMD.\n")
	} else {
		console.Printf("בלי חלון CMD (מתאים לתוכניות עם חלונות).\n")
	}
	return nil
}

// clearOldPackDist מוחק את תיקיית ההפצה הישנה לפני אריזה מחדש.
func clearOldPackDist(srcPath, out string, folder bool) error {
	dist := ""
	switch {
	case out == "":
		dist = project.DefaultDistDir(srcPath)
	case folder:
		dist = filepath.Clean(out)
	default:
		cleaned := filepath.Clean(out)
		if strings.EqualFold(filepath.Base(cleaned), project.DistDirName) {
			dist = cleaned
		} else {
			parent := filepath.Dir(cleaned)
			if strings.EqualFold(filepath.Base(parent), project.DistDirName) {
				dist = parent
			}
		}
	}
	if dist == "" {
		return nil
	}
	info, err := os.Stat(dist)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("בדיקת תיקיית הפצה ישנה נכשלה: %v", err)
	}
	if !info.IsDir() {
		return nil
	}
	if err := os.RemoveAll(dist); err != nil {
		return fmt.Errorf("לא ניתן למחוק את תיקיית ההפצה הישנה (%s).\nסגרו את ה־EXE אם הוא רץ משם, ונסו שוב.\n%v", dist, err)
	}
	console.Printf("נמחקה תיקיית הפצה ישנה: %s\n", dist)
	return nil
}
