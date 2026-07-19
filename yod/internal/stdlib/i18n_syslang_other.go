//go:build !windows

package stdlib

func detectSystemLangOS() string {
	return "en"
}
