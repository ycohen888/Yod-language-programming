//go:build !windows

package editor

import "fmt"

// Run — העורך זמין ב־Windows בלבד.
func Run(path string) error {
	return fmt.Errorf("עורך יוד זמין כרגע ב־Windows בלבד")
}
