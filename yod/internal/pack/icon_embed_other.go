//go:build !windows

package pack

func applyIconToEXE(exePath, icoPath string) error {
	return nil
}
