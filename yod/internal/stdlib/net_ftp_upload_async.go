package stdlib

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"yod/internal/object"

	"github.com/jlaffaye/ftp"
)

type ftpUploadSession struct {
	written atomic.Int64
	total   int64
	done    atomic.Bool
	cancel  atomic.Bool
	err     atomic.Value // string
	speed   *transferSpeedTracker
}

type countingReader struct {
	r      io.Reader
	n      *atomic.Int64
	cancel *atomic.Bool
}

func (c *countingReader) Read(p []byte) (int, error) {
	if c.cancel != nil && c.cancel.Load() {
		return 0, io.ErrClosedPipe
	}
	n, err := c.r.Read(p)
	if n > 0 && c.n != nil {
		c.n.Add(int64(n))
	}
	return n, err
}

var (
	ftpUpMu   sync.Mutex
	ftpUpNext atomic.Int64
	ftpUpByID = map[int64]*ftpUploadSession{}
)

func registerFTPUploadAsync(m *object.Module, c *ftp.ServerConn, mu *sync.Mutex) {
	m.Attrs["התחל_העלאה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return ftpStartUploadAsync(c, mu, args...)
	}}
	m.Attrs["מצב_העלאה"] = &object.Builtin{Fn: ftpUploadStatus}
	m.Attrs["בטל_העלאה"] = &object.Builtin{Fn: ftpCancelUpload}
}

func ftpStartUploadAsync(c *ftp.ServerConn, mu *sync.Mutex, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.התחל_העלאה", 2, args); err != nil {
		return err
	}
	local, ok1 := asString(args[0])
	remote, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.התחל_העלאה מצפה ל־(נתיב_מקומי, נתיב_מרוחק)")
	}
	remote = remoteFTPPath(remote)
	info, err := os.Stat(local)
	if err != nil {
		return errObj("חיבור.התחל_העלאה: מקור לא נמצא: " + err.Error())
	}
	id := ftpUpNext.Add(1)
	s := &ftpUploadSession{total: info.Size(), speed: newTransferSpeedTracker()}
	s.err.Store("")
	ftpUpMu.Lock()
	ftpUpByID[id] = s
	ftpUpMu.Unlock()

	go func() {
		defer func() {
			s.done.Store(true)
		}()
		if mu != nil {
			mu.Lock()
			defer mu.Unlock()
		}
		if err := ftpEnsureDirs(c, remote); err != nil {
			s.err.Store(err.Error())
			return
		}
		f, err := os.Open(local)
		if err != nil {
			s.err.Store(err.Error())
			return
		}
		r := &countingReader{r: f, n: &s.written, cancel: &s.cancel}
		err = c.Stor(remote, r)
		_ = f.Close()
		if err != nil && !s.cancel.Load() {
			// חלק מהשרתים דורשים מחיקה אחרי קובץ חלקי (450) — ניסיון שני בלבד
			_ = c.Delete(remote)
			f2, err2 := os.Open(local)
			if err2 != nil {
				s.err.Store(err.Error())
				return
			}
			s.written.Store(0)
			r2 := &countingReader{r: f2, n: &s.written, cancel: &s.cancel}
			err = c.Stor(remote, r2)
			_ = f2.Close()
		}
		if err != nil {
			if s.cancel.Load() {
				s.err.Store("בוטל")
			} else {
				s.err.Store(err.Error())
			}
			return
		}
	}()

	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":            &object.Number{Value: float64(id)},
		"סהכ":             &object.Number{Value: float64(info.Size())},
		"הועתק":           &object.Number{Value: 0},
		"הסתיים":          &object.Boolean{Value: false},
		"מהירות":          &object.Number{Value: 0},
		"מהירות_ממוצעת": &object.Number{Value: 0},
	}}
}

func ftpUploadStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("חיבור.מצב_העלאה מצפה למזהה")
	}
	idF, ok := asNumber(args[0])
	if !ok {
		if h, okh := args[0].(*object.Hash); okh {
			if v, has := h.Pairs["מזהה"]; has {
				idF, ok = asNumber(v)
			}
		}
		if !ok {
			return errObj("חיבור.מצב_העלאה: מזהה חייב להיות מספר")
		}
	}
	id := int64(idF)
	ftpUpMu.Lock()
	s := ftpUpByID[id]
	ftpUpMu.Unlock()
	if s == nil {
		return errObj("חיבור.מצב_העלאה: מזהה העלאה לא קיים")
	}
	written := s.written.Load()
	done := s.done.Load()
	errStr, _ := s.err.Load().(string)
	pct := 0.0
	if s.total > 0 {
		pct = 100.0 * float64(written) / float64(s.total)
		if pct > 100 {
			pct = 100
		}
	} else if done && errStr == "" {
		pct = 100
	}
	if done && errStr == "" {
		pct = 100
	}
	instant, average := 0.0, 0.0
	if s.speed != nil {
		instant, average = s.speed.sample(written)
	}
	if done {
		ftpUpMu.Lock()
		delete(ftpUpByID, id)
		ftpUpMu.Unlock()
	}
	pairs := map[string]object.Object{
		"הועתק":  &object.Number{Value: float64(written)},
		"סהכ":    &object.Number{Value: float64(s.total)},
		"הסתיים": &object.Boolean{Value: done},
		"אחוז":   &object.Number{Value: pct},
		"שגיאה":  &object.String{Value: errStr},
	}
	for k, v := range speedStatusFields(instant, average) {
		pairs[k] = v
	}
	return &object.Hash{Pairs: pairs}
}

func ftpCancelUpload(args ...object.Object) object.Object {
	if err := expectArgs("חיבור.בטל_העלאה", 1, args); err != nil {
		return err
	}
	idF, ok := asNumber(args[0])
	if !ok {
		if h, okh := args[0].(*object.Hash); okh {
			if v, has := h.Pairs["מזהה"]; has {
				idF, ok = asNumber(v)
			}
		}
		if !ok {
			return errObj("חיבור.בטל_העלאה: מזהה חייב להיות מספר")
		}
	}
	id := int64(idF)
	ftpUpMu.Lock()
	s := ftpUpByID[id]
	ftpUpMu.Unlock()
	if s != nil {
		s.cancel.Store(true)
	}
	return object.Nil
}
