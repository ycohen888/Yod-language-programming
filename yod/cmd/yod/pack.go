package main

import (
	"yod/internal/console"
	"yod/internal/pack"
)

// packApp — ברירת מחדל EXE בודד בלי CMD; --קונסול / --תיקייה לפי הצורך.
func packApp(srcPath string, rest []string) error {
	folder := false
	keepConsole := false
	out := ""
	for _, a := range rest {
		switch a {
		case "--תיקייה", "--folder", "-d":
			folder = true
		case "--קונסול", "--console", "-c":
			keepConsole = true
		default:
			if out == "" {
				out = a
			}
		}
	}
	if folder {
		created, err := pack.App(srcPath, out)
		if err != nil {
			return err
		}
		console.Printf("נוצרה תיקיית הפצה: %s\n", created)
		console.Printf("להרצה: %s\\הרץ.bat\n", created)
		return nil
	}
	created, err := pack.EXEWithOptions(srcPath, out, pack.EXEOptions{Console: keepConsole})
	if err != nil {
		return err
	}
	console.Printf("נוצר קובץ EXE: %s\n", created)
	if keepConsole {
		console.Printf("מצב קונסול — יוצג חלון CMD.\n")
	} else {
		console.Printf("בלי חלון CMD (מתאים לתוכניות עם חלונות).\n")
	}
	return nil
}
