//go:build windows

package stdlib

import "os/exec"

func runOpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", "", url).Start()
}
