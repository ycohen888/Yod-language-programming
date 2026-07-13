// Package guide משבץ את מדריך השפה בבינארי ומחלץ אותו לדיסק כשאין עותק מקומי.
package guide

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"yod/internal/version"
)

//go:generate go run genzip.go

//go:embed guide.zip
var zipBytes []byte

const indexName = "מדריך שפת יוד.html"

// DirName — שם תיקיית המדריך בשורש הריפו / ליד ההרצה.
const DirName = "מדריך שפת יוד"

// IndexRel — נתיב יחסי לקובץ הפתיחה.
func IndexRel() string {
	return filepath.Join(DirName, indexName)
}

// FindOnDisk מחפש עותק מקומי של המדריך ליד ההרצה או בתיקייה הנוכחית.
func FindOnDisk() string {
	rel := IndexRel()
	var cands []string
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, rel),
			filepath.Join(dir, "..", rel),
			filepath.Join(dir, "..", "..", rel),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands,
			filepath.Join(wd, rel),
			filepath.Join(wd, "..", rel),
			filepath.Join(wd, "..", "..", rel),
		)
	}
	seen := map[string]bool{}
	for _, c := range cands {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
			return abs
		}
	}
	return ""
}

// EnsureIndex מחזיר נתיב לקובץ המדריך: מקומי אם קיים, אחרת מחלץ מהבינארי.
func EnsureIndex() (string, error) {
	if p := FindOnDisk(); p != "" {
		return p, nil
	}
	dir, err := extractToAppData()
	if err != nil {
		return "", err
	}
	index := filepath.Join(dir, indexName)
	if fi, err := os.Stat(index); err != nil || fi.IsDir() {
		return "", fmt.Errorf("קובץ המדריך חסר אחרי חילוץ: %s", index)
	}
	return index, nil
}

func appDataRoot() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "Yod", "guide", "v"+version.String)
}

func extractToAppData() (string, error) {
	dest := appDataRoot()
	marker := filepath.Join(dest, ".yod-guide-ok")
	index := filepath.Join(dest, indexName)
	if fi, err := os.Stat(index); err == nil && !fi.IsDir() {
		if _, err := os.Stat(marker); err == nil {
			return dest, nil
		}
	}
	_ = os.RemoveAll(dest)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return "", err
	}
	destClean := filepath.Clean(dest)
	for _, f := range zr.File {
		name := filepath.Clean(filepath.FromSlash(f.Name))
		if name == "." || name == "" || strings.HasPrefix(name, "..") {
			continue
		}
		target := filepath.Join(destClean, name)
		rel, err := filepath.Rel(destClean, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return "", copyErr
		}
	}
	if err := os.WriteFile(marker, []byte(version.String+"\n"), 0o644); err != nil {
		return "", err
	}
	return dest, nil
}
