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

// tryLaunchWebIDE — מפעיל את עורך Electron אם זמין (אפליקציה ארוזה או תיקיית פיתוח).
func tryLaunchWebIDE(openPath string) bool {
	if tryLaunchPackagedIDE(openPath) {
		return true
	}
	return tryLaunchDevIDE(openPath)
}

func tryLaunchPackagedIDE(openPath string) bool {
	exe := findPackagedIDEExe()
	if exe == "" {
		return false
	}
	var args []string
	if openPath != "" {
		if abs, err := filepath.Abs(openPath); err == nil {
			args = append(args, abs)
		} else {
			args = append(args, openPath)
		}
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	cmd.Env = append(os.Environ(), "ELECTRON_NO_ATTACH_CONSOLE=1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	if err := cmd.Start(); err != nil {
		return false
	}
	_ = cmd.Process.Release()
	return true
}

func tryLaunchDevIDE(openPath string) bool {
	ideDir := findYodIDEDir()
	if ideDir == "" {
		return false
	}
	electronExe := filepath.Join(ideDir, "node_modules", "electron", "dist", "יוד.exe")
	if _, err := os.Stat(electronExe); err != nil {
		electronExe = filepath.Join(ideDir, "node_modules", "electron", "dist", "electron.exe")
		if _, err := os.Stat(electronExe); err != nil {
			return false
		}
		_ = tryStampYodIcon(ideDir)
		if stamped := filepath.Join(ideDir, "node_modules", "electron", "dist", "יוד.exe"); fileExists(stamped) {
			electronExe = stamped
		}
	}
	distIndex := filepath.Join(ideDir, "dist", "index.html")
	if _, err := os.Stat(distIndex); err != nil {
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
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
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

func collectSearchDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		abs, err := filepath.Abs(d)
		if err != nil {
			return
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		dirs = append(dirs, abs)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		add(dir)
		add(filepath.Join(dir, "Yod IDE"))
		add(filepath.Join(dir, "עורך"))
		add(filepath.Join(dir, "yod-ide"))
		add(filepath.Join(dir, "yod-ide", "release", "win-unpacked"))
		add(filepath.Join(dir, ".."))
		add(filepath.Join(dir, "..", "yod-ide"))
		add(filepath.Join(dir, "..", "yod-ide", "release", "win-unpacked"))
	}
	if wd, err := os.Getwd(); err == nil {
		add(wd)
		add(filepath.Join(wd, "Yod IDE"))
		add(filepath.Join(wd, "עורך"))
		add(filepath.Join(wd, "yod-ide"))
		add(filepath.Join(wd, "yod-ide", "release", "win-unpacked"))
		add(filepath.Join(wd, "..", "yod-ide", "release", "win-unpacked"))
	}
	add(filepath.Join("..", "yod-ide", "release", "win-unpacked"))
	add("yod-ide")
	return dirs
}

func findPackagedIDEExe() string {
	names := []string{"Yod IDE.exe", "YodIDE.exe"}
	for _, dir := range collectSearchDirs() {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if !fileExists(p) {
				continue
			}
			// electron-builder: resources/ ליד ה־exe (או בתוך asar ב־portable)
			if fileExists(filepath.Join(dir, "resources")) {
				return p
			}
		}
	}
	return ""
}

func findYodIDEDir() string {
	for _, abs := range collectSearchDirs() {
		mainJS := filepath.Join(abs, "electron", "main.cjs")
		if _, err := os.Stat(mainJS); err == nil {
			return abs
		}
	}
	return ""
}

// webIDEHint — הודעה כשאין IDE מובנה.
func webIDEHint() string {
	return strings.TrimSpace(`עורך יוד לא נמצא.
אפשרויות:
  1) אפליקציה ארוזה: Yod IDE.exe ליד yod.exe
     (בנייה: cd yod-ide && npm run dist)
  2) מצב פיתוח:
     cd yod-ide
     npm install
     npm run build
ואז הריצו שוב: yod עורך`)
}
