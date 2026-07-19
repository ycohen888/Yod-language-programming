//go:build windows

package stdlib

import "golang.org/x/sys/windows"

func detectSystemLangOS() string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(langs) == 0 {
		return "he"
	}
	return normalizeLangCode(langs[0])
}
