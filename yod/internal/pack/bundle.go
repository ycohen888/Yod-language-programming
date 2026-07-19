package pack

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"yod/internal/vfs"
)

const (
	bundleMagic   = "YODBUND1"
	bundleMetaLen = 24 // zipLen + metaLen + magic
)

// BundleMeta — מטא־נתונים בזנב החבילה.
type BundleMeta struct {
	Entry   string `json:"כניסה"`
	Version int    `json:"גרסה"`
}

// WriteBundleOverlay כותב ZIP+מטא+קסם ל־writer (אחרי המנוע).
func WriteBundleOverlay(w io.Writer, files map[string][]byte, entry string) error {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		name := strings.ReplaceAll(k, "\\", "/")
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(files[k]); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	zipData := zipBuf.Bytes()

	meta := BundleMeta{Entry: strings.ReplaceAll(entry, "\\", "/"), Version: 1}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if _, err := w.Write(zipData); err != nil {
		return err
	}
	if _, err := w.Write(metaJSON); err != nil {
		return err
	}
	var trailer [bundleMetaLen]byte
	binary.LittleEndian.PutUint64(trailer[0:8], uint64(len(zipData)))
	binary.LittleEndian.PutUint64(trailer[8:16], uint64(len(metaJSON)))
	copy(trailer[16:24], bundleMagic)
	_, err = w.Write(trailer[:])
	return err
}

// parseBundleOverlay מחלץ VFS ומטא מסוף קובץ EXE.
func parseBundleOverlay(data []byte) (*vfs.FS, BundleMeta, bool) {
	var meta BundleMeta
	if len(data) < bundleMetaLen {
		return nil, meta, false
	}
	if string(data[len(data)-8:]) != bundleMagic {
		return nil, meta, false
	}
	zipLen := binary.LittleEndian.Uint64(data[len(data)-24 : len(data)-16])
	metaLen := binary.LittleEndian.Uint64(data[len(data)-16 : len(data)-8])
	if zipLen == 0 || metaLen == 0 {
		return nil, meta, false
	}
	total := zipLen + metaLen + uint64(bundleMetaLen)
	if total > uint64(len(data)) {
		return nil, meta, false
	}
	start := len(data) - int(total)
	zipData := data[start : start+int(zipLen)]
	metaJSON := data[start+int(zipLen) : start+int(zipLen)+int(metaLen)]
	if err := json.Unmarshal(metaJSON, &meta); err != nil || meta.Entry == "" {
		return nil, meta, false
	}
	fs, err := fsFromZip(zipData)
	if err != nil {
		return nil, meta, false
	}
	return fs, meta, true
}

func fsFromZip(zipData []byte) (*vfs.FS, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	fs := vfs.New("")
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		fs.Add(f.Name, data)
	}
	return fs, nil
}

// overlayKind מזהה את סוג הזנב.
func overlayKind(data []byte) string {
	if len(data) >= 8 && string(data[len(data)-8:]) == bundleMagic {
		return bundleMagic
	}
	if len(data) >= 8 && string(data[len(data)-8:]) == embedMagic {
		return embedMagic
	}
	return ""
}

// stripOverlay מחזיר בתאי PE בלי זנב הטמעה.
func stripOverlay(data []byte) []byte {
	switch overlayKind(data) {
	case bundleMagic:
		if len(data) < bundleMetaLen {
			return data
		}
		zipLen := binary.LittleEndian.Uint64(data[len(data)-24 : len(data)-16])
		metaLen := binary.LittleEndian.Uint64(data[len(data)-16 : len(data)-8])
		total := int(zipLen + metaLen + uint64(bundleMetaLen))
		if total > len(data) || zipLen == 0 {
			return data
		}
		return data[:len(data)-total]
	case embedMagic:
		if script, ok := parseOverlay(data); ok {
			return data[:len(data)-embedMetaLen-len(script)]
		}
	}
	return data
}

// LoadEmbedded טוען תוכנית מוטמעת: YODBUND1 (VFS) או YODPACK1 (סקריפט יחיד).
// מחזיר את קוד הכניסה; אם יש bundle — מרכיב את ה־VFS עם root=תיקיית ה־EXE.
func LoadEmbedded(exePath string) (source string, virtualEntry string, ok bool, err error) {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return "", "", false, err
	}
	exeDir := filepath.Dir(exePath)
	if abs, e := filepath.Abs(exeDir); e == nil {
		exeDir = abs
	}

	if fs, meta, ok := parseBundleOverlay(data); ok {
		fs.Root = exeDir
		vfs.Mount(fs)
		key := vfs.Normalize(meta.Entry)
		body, found := fs.Get(key)
		if !found {
			return "", "", false, fmt.Errorf("קובץ כניסה חסר בחבילה: %s", meta.Entry)
		}
		return string(body), filepath.Join(exeDir, filepath.FromSlash(key)), true, nil
	}

	vfs.Clear()
	script, ok := parseOverlay(data)
	if !ok {
		return "", "", false, nil
	}
	entryName := filepath.Base(exePath)
	entryName = strings.TrimSuffix(entryName, filepath.Ext(entryName)) + ".יוד"
	return string(script), filepath.Join(exeDir, entryName), true, nil
}

// ReadEmbedded — תאימות לאחור: מחזיר רק את קוד הכניסה (ומרכיב VFS אם יש bundle).
func ReadEmbedded(exePath string) (source string, ok bool, err error) {
	src, _, ok, err := LoadEmbedded(exePath)
	return src, ok, err
}
