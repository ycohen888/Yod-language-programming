package stdlib

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"yod/internal/object"

	"github.com/jlaffaye/ftp"
)

// רשת.FTP_התחבר(אפשרויות) — FTP או FTPS (מאובטח=אמת → TLS מפורש)
// חוסם עד dial+login — לממשק השתמשו ב־FTP_התחל_התחברות / FTP_מצב_התחברות.
func netFTPConnect(args ...object.Object) object.Object {
	if err := expectArgs("רשת.FTP_התחבר", 1, args); err != nil {
		return err
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("רשת.FTP_התחבר מצפה למילון אפשרויות")
	}
	mod, err := dialFTPFromHash(h)
	if err != nil {
		return errObj(err.Error())
	}
	return mod
}

// normalizeFTPAddress מנקה ftp:// / ftps://, רווחים, נתיב ומשתמש מכתובת שהמשתמש הדביק.
func normalizeFTPAddress(raw string, port int, secure bool) (string, int, bool) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "ftps://"):
		secure = true
		s = s[len("ftps://"):]
	case strings.HasPrefix(lower, "ftp://"):
		s = s[len("ftp://"):]
	case strings.HasPrefix(lower, "ftps:"):
		secure = true
		s = s[len("ftps:"):]
		s = strings.TrimPrefix(s, "//")
	case strings.HasPrefix(lower, "ftp:"):
		s = s[len("ftp:"):]
		s = strings.TrimPrefix(s, "//")
	}
	if at := strings.Index(s, "@"); at >= 0 {
		s = s[at+1:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if host, pStr, err := splitHostPortLoose(s); err == nil {
		if p, e := strconv.Atoi(pStr); e == nil && p > 0 {
			port = p
		}
		return host, port, secure
	}
	return strings.Trim(s, "[]"), port, secure
}

func splitHostPortLoose(s string) (host, port string, err error) {
	// host:port או [ipv6]:port
	if strings.HasPrefix(s, "[") {
		return net.SplitHostPort(s)
	}
	// אל תפצל IPv6 בלי סוגריים
	if strings.Count(s, ":") == 1 {
		return net.SplitHostPort(s)
	}
	return "", "", fmt.Errorf("no port")
}

func newFTPConn(c *ftp.ServerConn, secure bool) *object.Module {
	kind := "FTP"
	if secure {
		kind = "FTPS"
	}
	var mu sync.Mutex
	m := &object.Module{Name: "חיבור_FTP", Attrs: map[string]object.Object{}}
	m.Attrs["סוג"] = &object.String{Value: kind}
	m.Attrs["רשימה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpList(c, args...)
	}}
	m.Attrs["העלה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpUpload(c, args...)
	}}
	registerFTPUploadAsync(m, c, &mu)
	m.Attrs["הורד"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpDownload(c, args...)
	}}
	m.Attrs["צור_תיקייה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpMkdir(c, args...)
	}}
	m.Attrs["מחק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpDelete(c, args...)
	}}
	m.Attrs["מחק_תיקייה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpRemoveDir(c, args...)
	}}
	m.Attrs["מחק_רקורסיבי"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpRemoveRecursive(c, args...)
	}}
	m.Attrs["קיים"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpExists(c, args...)
	}}
	m.Attrs["שנה_שם"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		mu.Lock()
		defer mu.Unlock()
		return ftpRename(c, args...)
	}}
	m.Attrs["בדוק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if err := expectArgs("חיבור.בדוק", 0, args); err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		if err := c.NoOp(); err != nil {
			return errObj("בדיקת FTP נכשלה: " + err.Error())
		}
		return &object.Boolean{Value: true}
	}}
	m.Attrs["נתק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if err := expectArgs("חיבור.נתק", 0, args); err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		ftpQuitSoft(c)
		return object.Nil
	}}
	return m
}

// מטמון תיקיות שכבר נוצרו בחיבור — חוסך MKD חוזר לכל קובץ (כמו FileZilla).
var (
	ftpEnsuredMu   sync.Mutex
	ftpEnsuredDirs = map[*ftp.ServerConn]map[string]struct{}{}
)

func ftpDirCacheClear(c *ftp.ServerConn) {
	if c == nil {
		return
	}
	ftpEnsuredMu.Lock()
	delete(ftpEnsuredDirs, c)
	ftpEnsuredMu.Unlock()
}

func ftpDirIsCached(c *ftp.ServerConn, dir string) bool {
	ftpEnsuredMu.Lock()
	defer ftpEnsuredMu.Unlock()
	m := ftpEnsuredDirs[c]
	if m == nil {
		return false
	}
	_, ok := m[dir]
	return ok
}

func ftpDirMarkCached(c *ftp.ServerConn, dir string) {
	ftpEnsuredMu.Lock()
	defer ftpEnsuredMu.Unlock()
	m := ftpEnsuredDirs[c]
	if m == nil {
		m = map[string]struct{}{}
		ftpEnsuredDirs[c] = m
	}
	m[dir] = struct{}{}
}

// ftpQuitSoft — QUIT עלול לתקוע שרתים; מגבילים ל־2 שניות ולא מחזירים שגיאה.
func ftpQuitSoft(c *ftp.ServerConn) {
	if c == nil {
		return
	}
	ftpDirCacheClear(c)
	done := make(chan struct{})
	go func() {
		_ = c.Quit()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func remoteFTPPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

func ftpList(c *ftp.ServerConn, args ...object.Object) object.Object {
	dir := "/"
	if len(args) >= 1 {
		s, ok := asString(args[0])
		if !ok {
			return errObj("חיבור.רשימה מצפה לנתיב מחרוזת")
		}
		dir = remoteFTPPath(s)
	}
	entries, err := c.List(dir)
	if err != nil {
		return errObj("רשימת FTP נכשלה: " + err.Error())
	}
	out := &object.Array{Elements: make([]object.Object, 0, len(entries))}
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		typ := "קובץ"
		if e.Type == ftp.EntryTypeFolder {
			typ = "תיקיה"
		} else if e.Type == ftp.EntryTypeLink {
			typ = "קישור"
		}
		out.Elements = append(out.Elements, &object.Hash{Pairs: map[string]object.Object{
			"שם":  &object.String{Value: e.Name},
			"סוג":  &object.String{Value: typ},
			"גודל": &object.Number{Value: float64(e.Size)},
			"זמן":  &object.Number{Value: float64(e.Time.Unix())},
		}})
	}
	return out
}

func ftpEnsureDirs(c *ftp.ServerConn, remoteFile string) error {
	dir := path.Dir(remoteFTPPath(remoteFile))
	if dir == "/" || dir == "." || dir == "" {
		return nil
	}
	if ftpDirIsCached(c, dir) {
		return nil
	}
	parts := strings.Split(strings.Trim(dir, "/"), "/")
	cur := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur += "/" + p
		if ftpDirIsCached(c, cur) {
			continue
		}
		if err := c.MakeDir(cur); err != nil {
			// קיים כבר — ממשיכים; שגיאה אחרת — לא מסמנים במטמון
			if !ftpDirProbablyExists(err) {
				return fmt.Errorf("יצירת תיקייה בשרת נכשלה (%s): %w", cur, err)
			}
		}
		ftpDirMarkCached(c, cur)
	}
	return nil
}

func ftpDirProbablyExists(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "exists") ||
		strings.Contains(s, "already") ||
		strings.Contains(s, "file exist") ||
		strings.Contains(s, "directory exist")
}

func ftpUpload(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.העלה", 2, args); err != nil {
		return err
	}
	local, ok1 := asString(args[0])
	remote, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.העלה מצפה ל־(נתיב_מקומי, נתיב_מרוחק)")
	}
	remote = remoteFTPPath(remote)
	if err := ftpEnsureDirs(c, remote); err != nil {
		return errObj("יצירת תיקיות להעלאה נכשלה: " + err.Error())
	}
	f, err := os.Open(local)
	if err != nil {
		return errObj("לא ניתן לפתוח קובץ מקומי: " + err.Error())
	}
	defer f.Close()
	if err := c.Stor(remote, f); err != nil {
		return errObj("העלאת FTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func ftpDownload(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.הורד", 2, args); err != nil {
		return err
	}
	remote, ok1 := asString(args[0])
	local, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.הורד מצפה ל־(נתיב_מרוחק, נתיב_מקומי)")
	}
	remote = remoteFTPPath(remote)
	resp, err := c.Retr(remote)
	if err != nil {
		return errObj("הורדת FTP נכשלה: " + err.Error())
	}
	defer resp.Close()
	if err := os.MkdirAll(path.Dir(strings.ReplaceAll(local, "\\", "/")), 0o755); err != nil {
		// Windows paths — try parent via filepath
	}
	if err := os.MkdirAll(filepathDir(local), 0o755); err != nil {
		return errObj("יצירת תיקייה מקומית נכשלה: " + err.Error())
	}
	out, err := os.Create(local)
	if err != nil {
		return errObj("יצירת קובץ מקומי נכשלה: " + err.Error())
	}
	defer out.Close()
	if _, err := io.Copy(out, resp); err != nil {
		return errObj("כתיבת קובץ מקומי נכשלה: " + err.Error())
	}
	return object.Nil
}

func filepathDir(p string) string {
	p = strings.ReplaceAll(p, "/", string(os.PathSeparator))
	i := strings.LastIndex(p, string(os.PathSeparator))
	if i <= 0 {
		return "."
	}
	return p[:i]
}

func ftpMkdir(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.צור_תיקייה", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.צור_תיקייה מצפה לנתיב מחרוזת")
	}
	p := remoteFTPPath(s)
	parts := strings.Split(strings.Trim(p, "/"), "/")
	cur := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur += "/" + part
		if ftpDirIsCached(c, cur) {
			continue
		}
		if err := c.MakeDir(cur); err != nil {
			if !ftpDirProbablyExists(err) {
				return errObj("יצירת תיקייה בשרת נכשלה (" + cur + "): " + err.Error())
			}
		}
		ftpDirMarkCached(c, cur)
	}
	return object.Nil
}

func ftpDelete(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק מצפה לנתיב מחרוזת")
	}
	p := remoteFTPPath(s)
	if err := c.Delete(p); err != nil {
		return errObj("מחיקת FTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func ftpRemoveDir(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק_תיקייה", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק_תיקייה מצפה לנתיב מחרוזת")
	}
	p := remoteFTPPath(s)
	if err := c.RemoveDir(p); err != nil {
		return errObj("מחיקת תיקייה ב־FTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func ftpRemoveRecursive(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק_רקורסיבי", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק_רקורסיבי מצפה לנתיב מחרוזת")
	}
	p := remoteFTPPath(s)
	if err := ftpRemovePathRecursive(c, p); err != nil {
		return errObj("מחיקה רקורסיבית ב־FTP נכשלה: " + err.Error())
	}
	ftpDirCacheClear(c)
	return object.Nil
}

// ftpRemovePathRecursive מוחק קובץ או תיקייה (כולל תוכן) בנתיבים מוחלטים בלבד —
// בלי ChangeDir, כדי לא להשאיר את החיבור בתיקייה שגויה אחרי כשל.
func ftpRemovePathRecursive(c *ftp.ServerConn, p string) error {
	entries, listErr := c.List(p)
	if listErr != nil {
		if err := c.Delete(p); err == nil {
			return nil
		}
		if _, err := c.GetEntry(p); err != nil {
			return nil // כבר לא קיים
		}
		return listErr
	}
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		child := path.Join(p, e.Name)
		if e.Type == ftp.EntryTypeFolder {
			if err := ftpRemovePathRecursive(c, child); err != nil {
				return err
			}
			continue
		}
		if err := c.Delete(child); err != nil {
			// חלק משרתים מדווחים תיקייה כקובץ ב־LIST — מנסים רקורסיה רק אם LIST מצליח
			if _, listErr := c.List(child); listErr != nil {
				return err
			}
			if err2 := ftpRemovePathRecursive(c, child); err2 != nil {
				return err
			}
		}
	}
	if err := c.RemoveDir(p); err != nil {
		if err2 := c.Delete(p); err2 == nil {
			return nil
		}
		if _, err2 := c.GetEntry(p); err2 != nil {
			return nil
		}
		return err
	}
	return nil
}

func ftpExists(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.קיים", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.קיים מצפה לנתיב מחרוזת")
	}
	p := remoteFTPPath(s)
	if _, err := c.GetEntry(p); err != nil {
		return &object.Boolean{Value: false}
	}
	return &object.Boolean{Value: true}
}

func ftpRename(c *ftp.ServerConn, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.שנה_שם", 2, args); err != nil {
		return err
	}
	from, ok1 := asString(args[0])
	to, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.שנה_שם מצפה ל־(נתיב_ישן, נתיב_חדש)")
	}
	from = remoteFTPPath(from)
	to = remoteFTPPath(to)
	if from == to {
		return object.Nil
	}
	if err := c.Rename(from, to); err != nil {
		return errObj("שינוי שם ב־FTP נכשל: " + err.Error())
	}
	return object.Nil
}
