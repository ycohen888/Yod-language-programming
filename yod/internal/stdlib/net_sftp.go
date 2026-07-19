package stdlib

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"yod/internal/object"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// רשת.SFTP_התחבר(אפשרויות) — SFTP מעל SSH
func netSFTPConnect(args ...object.Object) object.Object {
	if err := expectArgs("רשת.SFTP_התחבר", 1, args); err != nil {
		return err
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("רשת.SFTP_התחבר מצפה למילון אפשרויות")
	}
	host := strings.TrimSpace(hashStr(h, "כתובת"))
	if host == "" {
		return errObj("רשת.SFTP_התחבר: חסרה כתובת")
	}
	port := hashInt(h, "פורט", 22)
	if port < 1 || port > 65535 {
		return errObj("רשת.SFTP_התחבר: פורט לא תקין")
	}
	user := strings.TrimSpace(hashStr(h, "משתמש"))
	if user == "" {
		return errObj("רשת.SFTP_התחבר: חסר משתמש")
	}
	pass := hashStr(h, "סיסמה")
	keyPath := strings.TrimSpace(hashStr(h, "מפתח"))
	keyPass := hashStr(h, "סיסמת_מפתח")
	timeoutSec := hashInt(h, "זמן_קצוב", 30)
	if timeoutSec < 1 {
		timeoutSec = 30
	}
	if timeoutSec > 600 {
		timeoutSec = 600
	}
	verifyHost := hashBool(h, "אמת_מארח", false)

	var auths []ssh.AuthMethod
	if keyPath != "" {
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return errObj("קריאת מפתח SSH נכשלה: " + err.Error())
		}
		var signer ssh.Signer
		if keyPass != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(keyPass))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return errObj("פענוח מפתח SSH נכשל: " + err.Error())
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if pass != "" {
		auths = append(auths, ssh.Password(pass))
	}
	if len(auths) == 0 {
		return errObj("רשת.SFTP_התחבר: יש לספק סיסמה או מפתח")
	}

	var hostKeyCallback ssh.HostKeyCallback
	if verifyHost {
		return errObj("אמת_מארח: אמת דורש known_hosts (עדיין לא מובנה). השתמשו ב־אמת_מארח: שקר (ברירת מחדל).")
	}
	hostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec // ברירת מחדל / מפורש ע״י המשתמש

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(timeoutSec) * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	sshClient, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return errObj("חיבור SSH נכשל: " + err.Error())
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return errObj("פתיחת SFTP נכשלה: " + err.Error())
	}
	return newSFTPConn(sftpClient, sshClient)
}

func newSFTPConn(sc *sftp.Client, sshc *ssh.Client) *object.Module {
	m := &object.Module{Name: "חיבור_SFTP", Attrs: map[string]object.Object{}}
	m.Attrs["סוג"] = &object.String{Value: "SFTP"}
	m.Attrs["רשימה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpList(sc, args...)
	}}
	m.Attrs["העלה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpUpload(sc, args...)
	}}
	registerSFTPUploadAsync(m, sc)
	m.Attrs["הורד"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpDownload(sc, args...)
	}}
	m.Attrs["צור_תיקייה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpMkdir(sc, args...)
	}}
	m.Attrs["מחק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpDelete(sc, args...)
	}}
	m.Attrs["מחק_תיקייה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpRemoveDir(sc, args...)
	}}
	m.Attrs["מחק_רקורסיבי"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpRemoveRecursive(sc, args...)
	}}
	m.Attrs["קיים"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpExists(sc, args...)
	}}
	m.Attrs["שנה_שם"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpRename(sc, args...)
	}}
	m.Attrs["בדוק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if err := expectArgs("חיבור.בדוק", 0, args); err != nil {
			return err
		}
		_, err := sc.Getwd()
		if err != nil {
			return errObj("בדיקת SFTP נכשלה: " + err.Error())
		}
		return &object.Boolean{Value: true}
	}}
	m.Attrs["נתק"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if err := expectArgs("חיבור.נתק", 0, args); err != nil {
			return err
		}
		sftpDirCacheClear(sc)
		_ = sc.Close()
		if sshc != nil {
			_ = sshc.Close()
		}
		return object.Nil
	}}
	return m
}

var (
	sftpEnsuredMu   sync.Mutex
	sftpEnsuredDirs = map[*sftp.Client]map[string]struct{}{}
)

func sftpDirCacheClear(sc *sftp.Client) {
	if sc == nil {
		return
	}
	sftpEnsuredMu.Lock()
	delete(sftpEnsuredDirs, sc)
	sftpEnsuredMu.Unlock()
}

func sftpDirIsCached(sc *sftp.Client, dir string) bool {
	sftpEnsuredMu.Lock()
	defer sftpEnsuredMu.Unlock()
	m := sftpEnsuredDirs[sc]
	if m == nil {
		return false
	}
	_, ok := m[dir]
	return ok
}

func sftpDirMarkCached(sc *sftp.Client, dir string) {
	sftpEnsuredMu.Lock()
	defer sftpEnsuredMu.Unlock()
	m := sftpEnsuredDirs[sc]
	if m == nil {
		m = map[string]struct{}{}
		sftpEnsuredDirs[sc] = m
	}
	m[dir] = struct{}{}
}

func remoteSFTPPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

func sftpList(sc *sftp.Client, args ...object.Object) object.Object {
	dir := "/"
	if len(args) >= 1 {
		s, ok := asString(args[0])
		if !ok {
			return errObj("חיבור.רשימה מצפה לנתיב מחרוזת")
		}
		dir = remoteSFTPPath(s)
	}
	entries, err := sc.ReadDir(dir)
	if err != nil {
		return errObj("רשימת SFTP נכשלה: " + err.Error())
	}
	out := &object.Array{Elements: make([]object.Object, 0, len(entries))}
	for _, e := range entries {
		typ := "קובץ"
		if e.IsDir() {
			typ = "תיקיה"
		}
		out.Elements = append(out.Elements, &object.Hash{Pairs: map[string]object.Object{
			"שם":  &object.String{Value: e.Name()},
			"סוג":  &object.String{Value: typ},
			"גודל": &object.Number{Value: float64(e.Size())},
			"זמן":  &object.Number{Value: float64(e.ModTime().Unix())},
		}})
	}
	return out
}

func sftpEnsureDirs(sc *sftp.Client, remoteFile string) error {
	dir := path.Dir(remoteSFTPPath(remoteFile))
	if dir == "/" || dir == "." {
		return nil
	}
	if sftpDirIsCached(sc, dir) {
		return nil
	}
	if err := sc.MkdirAll(dir); err != nil {
		return err
	}
	sftpDirMarkCached(sc, dir)
	return nil
}

func sftpUpload(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.העלה", 2, args); err != nil {
		return err
	}
	local, ok1 := asString(args[0])
	remote, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.העלה מצפה ל־(נתיב_מקומי, נתיב_מרוחק)")
	}
	remote = remoteSFTPPath(remote)
	if err := sftpEnsureDirs(sc, remote); err != nil {
		return errObj("יצירת תיקיות להעלאה נכשלה: " + err.Error())
	}
	src, err := os.Open(local)
	if err != nil {
		return errObj("לא ניתן לפתוח קובץ מקומי: " + err.Error())
	}
	defer src.Close()
	dst, err := sc.Create(remote)
	if err != nil {
		return errObj("יצירת קובץ מרוחק נכשלה: " + err.Error())
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return errObj("העלאת SFTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func sftpDownload(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.הורד", 2, args); err != nil {
		return err
	}
	remote, ok1 := asString(args[0])
	local, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.הורד מצפה ל־(נתיב_מרוחק, נתיב_מקומי)")
	}
	remote = remoteSFTPPath(remote)
	src, err := sc.Open(remote)
	if err != nil {
		return errObj("פתיחת קובץ מרוחק נכשלה: " + err.Error())
	}
	defer src.Close()
	if err := os.MkdirAll(filepathDir(local), 0o755); err != nil {
		return errObj("יצירת תיקייה מקומית נכשלה: " + err.Error())
	}
	dst, err := os.Create(local)
	if err != nil {
		return errObj("יצירת קובץ מקומי נכשלה: " + err.Error())
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return errObj("הורדת SFTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func sftpMkdir(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.צור_תיקייה", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.צור_תיקייה מצפה לנתיב מחרוזת")
	}
	p := remoteSFTPPath(s)
	if sftpDirIsCached(sc, p) {
		return object.Nil
	}
	if err := sc.MkdirAll(p); err != nil {
		return errObj("יצירת תיקייה ב־SFTP נכשלה: " + err.Error())
	}
	sftpDirMarkCached(sc, p)
	return object.Nil
}

func sftpDelete(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק מצפה לנתיב מחרוזת")
	}
	if err := sc.Remove(remoteSFTPPath(s)); err != nil {
		return errObj("מחיקת SFTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func sftpRemoveDir(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק_תיקייה", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק_תיקייה מצפה לנתיב מחרוזת")
	}
	if err := sc.Remove(remoteSFTPPath(s)); err != nil {
		return errObj("מחיקת תיקייה ב־SFTP נכשלה: " + err.Error())
	}
	return object.Nil
}

func sftpRemoveRecursive(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.מחק_רקורסיבי", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.מחק_רקורסיבי מצפה לנתיב מחרוזת")
	}
	p := remoteSFTPPath(s)
	if err := sc.RemoveAll(p); err != nil {
		if _, statErr := sc.Stat(p); statErr != nil {
			return object.Nil // כבר לא קיים
		}
		return errObj("מחיקה רקורסיבית ב־SFTP נכשלה: " + err.Error())
	}
	sftpDirCacheClear(sc)
	return object.Nil
}

func sftpExists(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.קיים", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("חיבור.קיים מצפה לנתיב מחרוזת")
	}
	_, err := sc.Stat(remoteSFTPPath(s))
	if err != nil {
		return &object.Boolean{Value: false}
	}
	return &object.Boolean{Value: true}
}

func sftpRename(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.שנה_שם", 2, args); err != nil {
		return err
	}
	from, ok1 := asString(args[0])
	to, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.שנה_שם מצפה ל־(נתיב_ישן, נתיב_חדש)")
	}
	from = remoteSFTPPath(from)
	to = remoteSFTPPath(to)
	if from == to {
		return object.Nil
	}
	if err := sc.Rename(from, to); err != nil {
		return errObj("שינוי שם ב־SFTP נכשל: " + err.Error())
	}
	return object.Nil
}

