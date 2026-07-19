package stdlib

import (
	"os"
	"strings"
)

func detectSystemLang() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		if v := os.Getenv(key); v != "" {
			v = strings.Split(v, ".")[0]
			v = strings.Split(v, ":")[0]
			if v != "" && strings.ToLower(v) != "c" {
				return normalizeLangCode(v)
			}
		}
	}
	return detectSystemLangOS()
}
