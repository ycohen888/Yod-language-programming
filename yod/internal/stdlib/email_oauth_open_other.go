//go:build !windows

package stdlib

import (
	"fmt"
	"os/exec"
	"runtime"
)

func runOpenURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("פתיחת דפדפן לא נתמכת ב־%s — פתחו ידנית: %s", runtime.GOOS, url)
	}
}
