package pack

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	"yod/internal/project"
)

// שמות איקון פרטיים קודם — אחר כך נפילה ללוגו יוד.
var privateIconNames = []string{"app.ico", "icon.ico", "לוגו.ico", "logo.ico"}
var yodIconNames = []string{"יוד.ico", "yod.ico"}
var privateImageNames = []string{
	"לוגו.png", "logo.png", "app.png", "icon.png",
	filepath.Join("נכסים", "לוגו.png"),
	filepath.Join("נכסים", "logo.png"),
	filepath.Join("assets", "logo.png"),
}
var privateB64Names = []string{
	filepath.Join("נכסים", "לוגו.b64"),
	filepath.Join("נכסים", "logo.b64"),
	"לוגו.b64",
	"logo.b64",
}

// resolvePackIcon מוצא איקון לפרויקט: פרטי אם קיים, אחרת של יוד.
// אם נמצא PNG/B64 — יוצר קובץ .ico זמני (הקורא אחראי למחוק אם temp).
func resolvePackIcon(srcPath string) (icoPath string, temp bool, err error) {
	dirs := packIconSearchDirs(srcPath)
	for _, d := range dirs {
		for _, name := range privateIconNames {
			p := filepath.Join(d, name)
			if fileExists(p) {
				return p, false, nil
			}
		}
	}
	for _, d := range dirs {
		for _, rel := range privateImageNames {
			p := filepath.Join(d, rel)
			if !fileExists(p) {
				continue
			}
			tmp, err := pngFileToTempICO(p)
			if err != nil {
				return "", false, err
			}
			return tmp, true, nil
		}
		for _, rel := range privateB64Names {
			p := filepath.Join(d, rel)
			if !fileExists(p) {
				continue
			}
			tmp, err := b64FileToTempICO(p)
			if err != nil {
				return "", false, err
			}
			return tmp, true, nil
		}
	}
	for _, d := range dirs {
		for _, name := range yodIconNames {
			p := filepath.Join(d, name)
			if fileExists(p) {
				return p, false, nil
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		for _, name := range yodIconNames {
			p := filepath.Join(dir, name)
			if fileExists(p) {
				return p, false, nil
			}
		}
		for _, p := range []string{
			filepath.Join(dir, "yod", "assets", "yod.ico"),
			filepath.Join(dir, "assets", "yod.ico"),
		} {
			if fileExists(p) {
				return p, false, nil
			}
		}
	}
	return "", false, nil
}

func packIconSearchDirs(srcPath string) []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d == "" {
			return
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		dirs = append(dirs, abs)
	}
	root := project.FindProjectRoot(srcPath)
	if root != "" {
		add(root)
	}
	add(filepath.Dir(srcPath))
	return dirs
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func pngFileToTempICO(pngPath string) (string, error) {
	f, err := os.Open(pngPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return "", fmt.Errorf("פענוח PNG לאיקון נכשל: %w", err)
	}
	return writeTempICOFromImage(img)
}

func b64FileToTempICO(b64Path string) (string, error) {
	raw, err := os.ReadFile(b64Path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(raw))
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(strings.ToLower(s[:i]), "base64") {
		s = s[i+1:]
	}
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return "", fmt.Errorf("פענוח לוגו.b64 נכשל: %w", err)
		}
	}
	img, err := png.Decode(bytes.NewReader(decoded))
	if err != nil {
		return "", fmt.Errorf("לוגו.b64 אינו PNG תקין: %w", err)
	}
	return writeTempICOFromImage(img)
}

func writeTempICOFromImage(img image.Image) (string, error) {
	tmp, err := os.CreateTemp("", "yod-icon-*.ico")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	_ = tmp.Close()
	if err := writeICOFromImage(path, img); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func writeICOFromImage(path string, src image.Image) error {
	sizes := []int{16, 32, 48, 64, 128, 256}
	pngs := make([][]byte, 0, len(sizes))
	for _, s := range sizes {
		img := resizeRGBA(src, s)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		pngs = append(pngs, buf.Bytes())
	}
	return writeICOFile(path, sizes, pngs)
}

func resizeRGBA(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func writeICOFile(path string, sizes []int, pngs [][]byte) error {
	n := len(sizes)
	headerSize := 6 + 16*n
	offset := headerSize
	out := make([]byte, headerSize)
	binary.LittleEndian.PutUint16(out[2:4], 1) // type icon
	binary.LittleEndian.PutUint16(out[4:6], uint16(n))
	for i, s := range sizes {
		entry := out[6+i*16 : 6+(i+1)*16]
		if s >= 256 {
			entry[0], entry[1] = 0, 0
		} else {
			entry[0], entry[1] = byte(s), byte(s)
		}
		binary.LittleEndian.PutUint16(entry[4:6], 1)
		binary.LittleEndian.PutUint16(entry[6:8], 32)
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngs[i])))
		binary.LittleEndian.PutUint32(entry[12:16], uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		out = append(out, p...)
	}
	return os.WriteFile(path, out, 0644)
}
