package stdlib

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yod/internal/object"

	"github.com/jlaffaye/ftp"
)

type ftpConnectSession struct {
	mu        sync.Mutex
	done      bool
	err       string
	conn      *object.Module
	cancelled bool
}

var (
	ftpConnMu   sync.Mutex
	ftpConnNext atomic.Int64
	ftpConnByID = map[int64]*ftpConnectSession{}
)

func registerFTPConnectAsync(m *object.Module) {
	m.Attrs["FTP_התחל_התחברות"] = &object.Builtin{Fn: netFTPStartConnectAsync}
	m.Attrs["FTP_מצב_התחברות"] = &object.Builtin{Fn: netFTPConnectStatus}
	m.Attrs["FTP_בטל_התחברות"] = &object.Builtin{Fn: netFTPCancelConnect}
}

func ftpTLSConfig(host string, skipVerify bool) *tls.Config {
	return &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
	}
}

func ftpTimeoutFromHash(h *object.Hash) time.Duration {
	timeoutSec := hashInt(h, "זמן_קצוב", 30)
	if timeoutSec < 1 {
		timeoutSec = 30
	}
	if timeoutSec > 600 {
		timeoutSec = 600
	}
	return time.Duration(timeoutSec) * time.Second
}

func ftpSecureFromHash(h *object.Hash) bool {
	if hashBool(h, "מאובטח", false) {
		return true
	}
	// גיבוי — אם המילוון מגיע עם סוג FTPS בלי דגל מאובטח
	s := strings.TrimSpace(strings.ToUpper(hashStr(h, "סוג")))
	return s == "FTPS"
}

// dialFTPFromHash — Dial+Login; להרצה בגורוטינה כדי לא לחסום את ה-UI.
func dialFTPFromHash(h *object.Hash) (*object.Module, error) {
	host := strings.TrimSpace(hashStr(h, "כתובת"))
	if host == "" {
		return nil, fmt.Errorf("חסרה כתובת")
	}
	port := hashInt(h, "פורט", 21)
	secure := ftpSecureFromHash(h)
	host, port, secure = normalizeFTPAddress(host, port, secure)
	if host == "" {
		return nil, fmt.Errorf("חסרה כתובת")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("פורט לא תקין")
	}
	user := hashStr(h, "משתמש")
	if user == "" {
		user = "anonymous"
	}
	pass := hashStr(h, "סיסמה")
	timeout := ftpTimeoutFromHash(h)
	addr := fmt.Sprintf("%s:%d", host, port)

	opts := []ftp.DialOption{
		ftp.DialWithTimeout(timeout),
		ftp.DialWithDisabledEPSV(true),
	}
	if secure {
		skip := hashBool(h, "דלג_אימות_תעודה", true)
		tlsCfg := ftpTLSConfig(host, skip)
		// מרומז=אמת או פורט 990 → TLS מיד; אחרת Explicit (AUTH TLS) — הנפוץ ב־cPanel
		implicit := hashBool(h, "מרומז", false) || port == 990
		if implicit {
			opts = append(opts, ftp.DialWithTLS(tlsCfg))
		} else {
			opts = append(opts, ftp.DialWithExplicitTLS(tlsCfg))
		}
	}

	conn, err := ftp.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("חיבור FTP נכשל: %w", err)
	}
	if err := conn.Login(user, pass); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("התחברות FTP נכשלה: %w", err)
	}
	return newFTPConn(conn, secure), nil
}

func copyFTPOptsHash(h *object.Hash) *object.Hash {
	optsCopy := &object.Hash{Pairs: map[string]object.Object{}}
	for _, k := range []string{
		"כתובת", "פורט", "משתמש", "סיסמה", "זמן_קצוב",
		"מאובטח", "דלג_אימות_תעודה", "סוג", "מרומז",
	} {
		if v, has := h.Pairs[k]; has {
			optsCopy.Pairs[k] = v
		}
	}
	return optsCopy
}

func netFTPStartConnectAsync(args ...object.Object) object.Object {
	if err := expectArgs("רשת.FTP_התחל_התחברות", 1, args); err != nil {
		return err
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("רשת.FTP_התחל_התחברות מצפה למילון אפשרויות")
	}
	optsCopy := copyFTPOptsHash(h)
	timeout := ftpTimeoutFromHash(optsCopy)
	// מרווח ביטחון מעבר ל־DialWithTimeout — TLS לפעמים לא מכבד את ה־deadline
	guard := timeout + 5*time.Second

	id := ftpConnNext.Add(1)
	s := &ftpConnectSession{}
	ftpConnMu.Lock()
	ftpConnByID[id] = s
	ftpConnMu.Unlock()

	go func() {
		type dialResult struct {
			mod *object.Module
			err error
		}
		ch := make(chan dialResult, 1)
		go func() {
			mod, err := dialFTPFromHash(optsCopy)
			ch <- dialResult{mod, err}
		}()

		var res dialResult
		select {
		case res = <-ch:
		case <-time.After(guard):
			res = dialResult{err: fmt.Errorf("התחברות FTP נכשלה: חריגה מזמן קצוב (%v)", timeout)}
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		s.done = true
		if s.cancelled {
			if res.mod != nil {
				// לא מוסרים למשתמש — סוגרים בשקט
				if c, ok := res.mod.Attrs["נתק"].(*object.Builtin); ok && c.Fn != nil {
					_ = c.Fn()
				}
			}
			return
		}
		if res.err != nil {
			s.err = res.err.Error()
			return
		}
		s.conn = res.mod
	}()

	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":    &object.Number{Value: float64(id)},
		"הסתיים": &object.Boolean{Value: false},
	}}
}

func netFTPConnectStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשת.FTP_מצב_התחברות מצפה למזהה")
	}
	idF, ok := asNumber(args[0])
	if !ok {
		if h, okh := args[0].(*object.Hash); okh {
			if v, has := h.Pairs["מזהה"]; has {
				idF, ok = asNumber(v)
			}
		}
		if !ok {
			return errObj("רשת.FTP_מצב_התחברות: מזהה חייב להיות מספר")
		}
	}
	id := int64(idF)
	ftpConnMu.Lock()
	s := ftpConnByID[id]
	ftpConnMu.Unlock()
	if s == nil {
		return errObj("רשת.FTP_מצב_התחברות: מזהה התחברות לא קיים")
	}

	s.mu.Lock()
	done := s.done
	errStr := s.err
	conn := s.conn
	cancelled := s.cancelled
	if done && !cancelled {
		// מוסרים מהמפה אחרי קריאה אחת מוצלחת — מונע מרוץ של שני פולים
		ftpConnMu.Lock()
		delete(ftpConnByID, id)
		ftpConnMu.Unlock()
		s.conn = nil
	}
	s.mu.Unlock()

	out := map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"שגיאה":  &object.String{Value: errStr},
		"חיבור":  object.Nil,
	}
	if cancelled && done {
		out["שגיאה"] = &object.String{Value: "התחברות בוטלה"}
	} else if done && errStr == "" && conn != nil {
		out["חיבור"] = conn
	}
	return &object.Hash{Pairs: out}
}

func netFTPCancelConnect(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשת.FTP_בטל_התחברות מצפה למזהה")
	}
	idF, ok := asNumber(args[0])
	if !ok {
		if h, okh := args[0].(*object.Hash); okh {
			if v, has := h.Pairs["מזהה"]; has {
				idF, ok = asNumber(v)
			}
		}
		if !ok {
			return errObj("רשת.FTP_בטל_התחברות: מזהה חייב להיות מספר")
		}
	}
	id := int64(idF)
	ftpConnMu.Lock()
	s := ftpConnByID[id]
	ftpConnMu.Unlock()
	if s != nil {
		s.mu.Lock()
		s.cancelled = true
		s.mu.Unlock()
	}
	return object.Nil
}
