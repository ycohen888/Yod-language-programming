//go:build windows

package editor

import "fmt"

// Run פותח את עורך יוד (Electron / yod-ide בלבד).
func Run(path string) error {
		if tryLaunchWebIDE(path) {
			return nil
		}
	return fmt.Errorf("%s", webIDEHint())
}
