package stdlib

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"yod/internal/object"

	"github.com/pkg/sftp"
)

type sftpUploadSession struct {
	written atomic.Int64
	total   int64
	done    atomic.Bool
	cancel  atomic.Bool
	err     atomic.Value // string
	speed   *transferSpeedTracker
}

var (
	sftpUpMu   sync.Mutex
	sftpUpNext atomic.Int64
	sftpUpByID = map[int64]*sftpUploadSession{}
)

func registerSFTPUploadAsync(m *object.Module, sc *sftp.Client) {
	m.Attrs["התחל_העלאה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return sftpStartUploadAsync(sc, args...)
	}}
	m.Attrs["מצב_העלאה"] = &object.Builtin{Fn: sftpUploadStatus}
	m.Attrs["בטל_העלאה"] = &object.Builtin{Fn: sftpCancelUpload}
}

func sftpStartUploadAsync(sc *sftp.Client, args ...object.Object) object.Object {
	if err := expectArgs("חיבור.התחל_העלאה", 2, args); err != nil {
		return err
	}
	local, ok1 := asString(args[0])
	remote, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("חיבור.התחל_העלאה מצפה ל־(נתיב_מקומי, נתיב_מרוחק)")
	}
	remote = remoteSFTPPath(remote)
	info, err := os.Stat(local)
	if err != nil {
		return errObj("חיבור.התחל_העלאה: מקור לא נמצא: " + err.Error())
	}
	id := sftpUpNext.Add(1)
	s := &sftpUploadSession{total: info.Size(), speed: newTransferSpeedTracker()}
	s.err.Store("")
	sftpUpMu.Lock()
	sftpUpByID[id] = s
	sftpUpMu.Unlock()

	go func() {
		defer func() {
			s.done.Store(true)
		}()
		if err := sftpEnsureDirs(sc, remote); err != nil {
			s.err.Store(err.Error())
			return
		}
		src, err := os.Open(local)
		if err != nil {
			s.err.Store(err.Error())
			return
		}
		defer src.Close()
		dst, err := sc.Create(remote)
		if err != nil {
			s.err.Store(err.Error())
			return
		}
		defer dst.Close()
		r := &countingReader{r: src, n: &s.written, cancel: &s.cancel}
		if _, err := io.Copy(dst, r); err != nil {
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

func sftpUploadStatus(args ...object.Object) object.Object {
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
	sftpUpMu.Lock()
	s := sftpUpByID[id]
	sftpUpMu.Unlock()
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
		sftpUpMu.Lock()
		delete(sftpUpByID, id)
		sftpUpMu.Unlock()
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

func sftpCancelUpload(args ...object.Object) object.Object {
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
	sftpUpMu.Lock()
	s := sftpUpByID[id]
	sftpUpMu.Unlock()
	if s != nil {
		s.cancel.Store(true)
	}
	return object.Nil
}
