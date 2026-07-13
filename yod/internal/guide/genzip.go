//go:build ignore

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "מדריך שפת יוד"))
	if err != nil {
		fatal(err)
	}
	out := "guide.zip"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}

	f, err := os.Create(out)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	n := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".") {
			return nil
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, r)
		_ = r.Close()
		if copyErr != nil {
			return copyErr
		}
		n++
		return nil
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s (%d files) from %s\n", out, n, root)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
