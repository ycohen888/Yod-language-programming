package evaluator

import (
	"os"
	"path/filepath"
	"strings"

	"yod/internal/vfs"
)

func resolveUserFile(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(path), ".יוד"), ".yod")
	candidates := []string{
		filepath.Join(base, path),
		filepath.Join(base, ".יוד_חבילות", path),
		filepath.Join(base, ".יוד_חבילות", path+".יוד"),
		filepath.Join(base, ".יוד_חבילות", path+".yod"),
		filepath.Join(base, ".יוד_חבילות", stem, filepath.Base(path)),
		filepath.Join(base, ".יוד_חבילות", stem, "התחל.יוד"),
	}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
		if fs := vfs.Active(); fs != nil && fs.Exists(c) && !fs.IsDir(c) {
			return c
		}
	}
	return filepath.Clean(filepath.Join(base, path))
}
