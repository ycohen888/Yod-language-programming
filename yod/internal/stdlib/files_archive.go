package stdlib

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/nwaples/rardecode/v2"

	"yod/internal/console"
	"yod/internal/object"
)

type archiveEntry struct {
	name   string
	size   int64
	packed int64
	isDir  bool
}

func filesArchive(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.ארוז", 2, args); err != nil {
		return err
	}
	sources, srcErr := archiveSources(args[0])
	if srcErr != nil {
		return errObj(srcErr.Error())
	}
	archivePath, ok := asString(args[1])
	if !ok {
		return errObj("קבצים.ארוז: נתיב הארכיון חייב להיות מחרוזת")
	}
	if len(sources) == 0 {
		return errObj("קבצים.ארוז: אין מה לארוז")
	}

	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".rar":
		if e := createRAR(archivePath, sources); e != nil {
			return errObj(e.Error())
		}
	case ".zip", "":
		if ext == "" {
			archivePath += ".zip"
		}
		if e := createZIP(archivePath, sources); e != nil {
			return errObj("לא הצלחתי לארוז ZIP: " + e.Error())
		}
	default:
		return errObj("פורמט ארכיון לא נתמך לאריזה: " + ext + " (נתמכים: .zip, וב־Windows גם .rar אם מותקן WinRAR)")
	}
	return &object.String{Value: archivePath}
}

func filesExtract(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.חלץ", 2, args); err != nil {
		return err
	}
	archivePath, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.חלץ: נתיב הארכיון חייב להיות מחרוזת")
	}
	dest, ok := asString(args[1])
	if !ok {
		return errObj("קבצים.חלץ: תיקיית היעד חייבת להיות מחרוזת")
	}
	if e := os.MkdirAll(dest, 0755); e != nil {
		return errObj("לא הצלחתי ליצור תיקיית יעד: " + e.Error())
	}

	format, err := detectArchiveFormat(archivePath)
	if err != nil {
		return errObj(err.Error())
	}
	switch format {
	case "zip":
		if e := extractZIP(archivePath, dest); e != nil {
			return errObj("לא הצלחתי לחלץ ZIP: " + e.Error())
		}
	case "rar":
		if e := extractRAR(archivePath, dest); e != nil {
			return errObj("לא הצלחתי לחלץ RAR: " + e.Error())
		}
	default:
		return errObj("פורמט ארכיון לא מזוהה או לא נתמך")
	}
	return &object.String{Value: dest}
}

func filesArchiveContents(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.תוכן_ארכיון", 1, args); err != nil {
		return err
	}
	archivePath, ok := asString(args[0])
	if !ok {
		return errObj("קבצים.תוכן_ארכיון מצפה לנתיב מחרוזת")
	}
	format, err := detectArchiveFormat(archivePath)
	if err != nil {
		return errObj(err.Error())
	}
	var entries []archiveEntry
	switch format {
	case "zip":
		entries, err = listZIP(archivePath)
	case "rar":
		entries, err = listRAR(archivePath)
	default:
		return errObj("פורמט ארכיון לא מזוהה או לא נתמך")
	}
	if err != nil {
		return errObj("לא הצלחתי לקרוא תוכן ארכיון: " + err.Error())
	}
	table := formatArchiveTable(archivePath, entries)
	console.Println(table)
	return object.Nil
}

func listZIP(path string) ([]archiveEntry, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out := make([]archiveEntry, 0, len(r.File))
	for _, f := range r.File {
		name := strings.TrimSuffix(f.Name, "/")
		if name == "" {
			continue
		}
		out = append(out, archiveEntry{
			name:   name,
			size:   int64(f.UncompressedSize64),
			packed: int64(f.CompressedSize64),
			isDir:  f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/"),
		})
	}
	return out, nil
}

func listRAR(path string) ([]archiveEntry, error) {
	files, err := rardecode.List(path)
	if err != nil {
		return nil, err
	}
	out := make([]archiveEntry, 0, len(files))
	for _, f := range files {
		name := strings.TrimSuffix(f.Name, "/")
		if name == "" {
			continue
		}
		out = append(out, archiveEntry{
			name:   name,
			size:   f.UnPackedSize,
			packed: f.PackedSize,
			isDir:  f.IsDir,
		})
	}
	return out, nil
}

func formatArchiveTable(archivePath string, entries []archiveEntry) string {
	type row struct {
		name   string
		size   string
		packed string
		kind   string
	}
	rows := make([]row, 0, len(entries))
	var totalSize, totalPacked int64
	var fileCount int
	for _, e := range entries {
		kind := "קובץ"
		sizeStr := formatBytesHE(e.size)
		packedStr := formatBytesHE(e.packed)
		if e.isDir {
			kind = "תיקייה"
			sizeStr = "—"
			packedStr = "—"
		} else {
			totalSize += e.size
			totalPacked += e.packed
			fileCount++
		}
		rows = append(rows, row{name: e.name, size: sizeStr, packed: packedStr, kind: kind})
	}

	hName, hSize, hPacked, hKind := "שם", "גודל", "דחוס", "סוג"
	wName := utf8.RuneCountInString(hName)
	wSize := utf8.RuneCountInString(hSize)
	wPacked := utf8.RuneCountInString(hPacked)
	wKind := utf8.RuneCountInString(hKind)
	for _, r := range rows {
		if n := utf8.RuneCountInString(r.name); n > wName {
			wName = n
		}
		if n := utf8.RuneCountInString(r.size); n > wSize {
			wSize = n
		}
		if n := utf8.RuneCountInString(r.packed); n > wPacked {
			wPacked = n
		}
		if n := utf8.RuneCountInString(r.kind); n > wKind {
			wKind = n
		}
	}
	if wName < 8 {
		wName = 8
	}

	var b strings.Builder
	b.WriteString("ארכיון: ")
	b.WriteString(archivePath)
	b.WriteByte('\n')
	sep := strings.Repeat("─", wName+wSize+wPacked+wKind+10)
	b.WriteString(sep)
	b.WriteByte('\n')
	b.WriteString(padRunes(hName, wName))
	b.WriteString(" │ ")
	b.WriteString(padRunes(hSize, wSize))
	b.WriteString(" │ ")
	b.WriteString(padRunes(hPacked, wPacked))
	b.WriteString(" │ ")
	b.WriteString(padRunes(hKind, wKind))
	b.WriteByte('\n')
	b.WriteString(sep)
	b.WriteByte('\n')
	if len(rows) == 0 {
		b.WriteString("(ריק)\n")
	} else {
		for _, r := range rows {
			b.WriteString(padRunes(r.name, wName))
			b.WriteString(" │ ")
			b.WriteString(padRunes(r.size, wSize))
			b.WriteString(" │ ")
			b.WriteString(padRunes(r.packed, wPacked))
			b.WriteString(" │ ")
			b.WriteString(padRunes(r.kind, wKind))
			b.WriteByte('\n')
		}
	}
	b.WriteString(sep)
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("סה״כ: %d קבצים · %s · דחוס %s",
		fileCount, formatBytesHE(totalSize), formatBytesHE(totalPacked)))
	return b.String()
}

func padRunes(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func formatBytesHE(n int64) string {
	if n < 0 {
		n = 0
	}
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f ג״ב", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f מ״ב", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f ק״ב", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d ב׳", n)
	}
}

func archiveSources(arg object.Object) ([]string, error) {
	switch v := arg.(type) {
	case *object.String:
		return []string{v.Value}, nil
	case *object.Array:
		out := make([]string, 0, len(v.Elements))
		for _, el := range v.Elements {
			s, ok := asString(el)
			if !ok {
				return nil, fmt.Errorf("קבצים.ארוז: רשימת מקורות חייבת להכיל מחרוזות")
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("קבצים.ארוז: מקור חייב להיות נתיב או רשימת נתיבים")
	}
}

func detectArchiveFormat(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip":
		return "zip", nil
	case ".rar":
		return "rar", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי לפתוח ארכיון: %w", err)
	}
	defer f.Close()
	var magic [8]byte
	n, _ := io.ReadFull(f, magic[:])
	if n >= 4 && magic[0] == 'P' && magic[1] == 'K' {
		return "zip", nil
	}
	if n >= 7 && string(magic[:7]) == "Rar!\x1a\x07\x00" {
		return "rar", nil
	}
	if n >= 8 && string(magic[:8]) == "Rar!\x1a\x07\x01" {
		return "rar", nil
	}
	return "", fmt.Errorf("לא הצלחתי לזהות פורמט ארכיון של %s", path)
}

func createZIP(archivePath string, sources []string) error {
	dir := filepath.Dir(archivePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	out, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, src := range sources {
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(src)
			err = filepath.Walk(src, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				rel, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}
				name := filepath.ToSlash(filepath.Join(base, rel))
				if rel == "." {
					name = base
				}
				if fi.IsDir() {
					if !strings.HasSuffix(name, "/") {
						name += "/"
					}
					_, err := zw.Create(name)
					return err
				}
				return addFileToZip(zw, path, name)
			})
			if err != nil {
				return err
			}
			continue
		}
		if err := addFileToZip(zw, src, filepath.ToSlash(filepath.Base(src))); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, diskPath, nameInZip string) error {
	w, err := zw.Create(nameInZip)
	if err != nil {
		return err
	}
	f, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func extractZIP(archivePath, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target, err := safeExtractPath(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractRAR(archivePath, dest string) error {
	rr, err := rardecode.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer rr.Close()

	for {
		hdr, err := rr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeExtractPath(dest, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.IsDir {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, rr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func createRAR(archivePath string, sources []string) error {
	rarExe, err := findRarExecutable()
	if err != nil {
		return fmt.Errorf("יצירת RAR דורשת WinRAR (לא נמצא). ארוז ל־ZIP במקום, או התקן WinRAR")
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil && filepath.Dir(archivePath) != "." {
		return err
	}
	_ = os.Remove(archivePath)
	args := []string{"a", "-r", "-ep1", archivePath}
	args = append(args, sources...)
	cmd := exec.Command(rarExe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("יצירת RAR נכשלה: %s", msg)
	}
	return nil
}

func findRarExecutable() (string, error) {
	candidates := []string{
		`C:\Program Files\WinRAR\Rar.exe`,
		`C:\Program Files (x86)\WinRAR\Rar.exe`,
		`C:\Program Files\WinRAR\rar.exe`,
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	if p, err := exec.LookPath("rar"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("Rar.exe"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("לא נמצא")
}

func safeExtractPath(destRoot, name string) (string, error) {
	name = strings.ReplaceAll(name, `\`, `/`)
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == "" {
		return destRoot, nil
	}
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("נתיב לא בטוח בארכיון: %s", name)
	}
	target := filepath.Join(destRoot, clean)
	root := filepath.Clean(destRoot)
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("נתיב לא בטוח בארכיון: %s", name)
	}
	return target, nil
}
