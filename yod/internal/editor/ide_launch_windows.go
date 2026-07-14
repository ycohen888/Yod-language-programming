//go:build windows

package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const createNoWindow = 0x08000000 // CREATE_NO_WINDOW

// tryLaunchWebIDE — מפעיל את עורך Electron (yod-ide) אם זמין.
// מחזיר true אם הושק בהצלחה (והתהליך הקורא צריך לצאת).
func tryLaunchWebIDE(openPath string) bool {
	ideDir := findYodIDEDir()
	if ideDir == "" {
		return false
	}
	// מעדיפים יוד.exe (עם איקון) על פני electron.exe הגנרי
	electronExe := filepath.Join(ideDir, "node_modules", "electron", "dist", "יוד.exe")
	if _, err := os.Stat(electronExe); err != nil {
		electronExe = filepath.Join(ideDir, "node_modules", "electron", "dist", "electron.exe")
		if _, err := os.Stat(electronExe); err != nil {
			return false
		}
		// ניסיון לחתום איקון אם Node זמין (פעם אחת / אחרי npm install)
		_ = tryStampYodIcon(ideDir)
		if stamped := filepath.Join(ideDir, "node_modules", "electron", "dist", "יוד.exe"); fileExists(stamped) {
			electronExe = stamped
		}
	}
	distIndex := filepath.Join(ideDir, "dist", "index.html")
	if _, err := os.Stat(distIndex); err != nil {
		// אין build — לא מפעילים אוטומטית (נדרש npm run build)
		return false
	}

	args := []string{"."}
	if openPath != "" {
		if abs, err := filepath.Abs(openPath); err == nil {
			args = append(args, abs)
		} else {
			args = append(args, openPath)
		}
	}
	cmd := exec.Command(electronExe, args...)
	cmd.Dir = ideDir
	cmd.Env = append(os.Environ(), "ELECTRON_NO_ATTACH_CONSOLE=1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	// CREATE_NO_WINDOW בלבד — בלי HideWindow (HideWindow מסתיר גם את חלון ה־GUI של Electron)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow,
	}
	if err := cmd.Start(); err != nil {
		return false
	}
	_ = cmd.Process.Release()
	return true
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func tryStampYodIcon(ideDir string) error {
	script := filepath.Join(ideDir, "scripts", "stamp-electron-icon.cjs")
	if !fileExists(script) {
		return fmt.Errorf("no stamp script")
	}
	cmd := exec.Command("node", script)
	cmd.Dir = ideDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd.Run()
}

func findYodIDEDir() string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "yod-ide"),
			filepath.Join(dir, "..", "yod-ide"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "yod-ide"),
			filepath.Join(wd, "..", "yod-ide"),
		)
	}
	// יחסי לקוד המקור כשמריצים מ־yod/
	candidates = append(candidates,
		filepath.Join("..", "yod-ide"),
		"yod-ide",
	)

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		mainJS := filepath.Join(abs, "electron", "main.cjs")
		if _, err := os.Stat(mainJS); err == nil {
			return abs
		}
	}
	return ""
}

// webIDEHint — הודעה כשאין IDE מובנה.
func webIDEHint() string {
	return strings.TrimSpace(fmt.Sprintf(`עורך Electron לא נמצא או לא נבנה.
התקינו ובנו:
  cd yod-ide
  npm install
  npm run build
ואז הריצו שוב: yod עורך`))
}
