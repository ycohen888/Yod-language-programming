//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/format"
)

func main() {
	root := filepath.Join("..", "פרוייקט דוגמה")
	changed := 0
	total := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".יוד" && !strings.EqualFold(ext, ".yod") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		out := format.Source(src)
		normIn := strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
		if !strings.HasSuffix(normIn, "\n") {
			normIn += "\n"
		}
		total++
		if out == normIn {
			fmt.Println("ok:", filepath.Base(path))
			return nil
		}
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Println("formatted:", rel)
		changed++
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("done: %d/%d files changed\n", changed, total)
}
