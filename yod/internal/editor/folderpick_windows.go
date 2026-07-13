//go:build windows

package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lxn/walk"
)

// pickFolder — דיאלוג תיקייה בתהליך PowerShell נפרד (לא נוגע ב־COM של walk).
func pickFolder(owner walk.Form, title, initial string) (string, bool, error) {
	_ = owner
	outFile := filepath.Join(os.TempDir(), "yod_pick_folder.txt")
	_ = os.Remove(outFile)

	esc := func(s string) string {
		return strings.ReplaceAll(s, "'", "''")
	}

	var b strings.Builder
	b.WriteString("Add-Type -AssemblyName System.Windows.Forms; ")
	b.WriteString("$f = New-Object System.Windows.Forms.FolderBrowserDialog; ")
	b.WriteString("$f.Description = '" + esc(title) + "'; ")
	b.WriteString("$f.ShowNewFolderButton = $true; ")
	if initial != "" {
		b.WriteString("if (Test-Path -LiteralPath '" + esc(initial) + "') { $f.SelectedPath = '" + esc(initial) + "' }; ")
	}
	b.WriteString("$r = $f.ShowDialog(); ")
	b.WriteString("if ($r -eq [System.Windows.Forms.DialogResult]::OK) { ")
	b.WriteString("[System.IO.File]::WriteAllText('" + esc(outFile) + "', $f.SelectedPath, (New-Object System.Text.UTF8Encoding $false)) ")
	b.WriteString("}")

	cmd := exec.Command("powershell.exe",
		"-NoProfile",
		"-STA",
		"-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden",
		"-Command", b.String(),
	)
	// CREATE_NO_WINDOW — בלי קונסולה, הדיאלוג עצמו עדיין מוצג
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}

	if err := cmd.Run(); err != nil {
		return "", false, err
	}

	data, err := os.ReadFile(outFile)
	_ = os.Remove(outFile)
	if err != nil {
		return "", false, nil // בוטל
	}
	path := strings.TrimSpace(string(data))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}
