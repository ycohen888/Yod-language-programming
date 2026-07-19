package stdlib

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"yod/internal/object"
)

type copySession struct {
	in      *os.File
	out     *os.File
	total   int64
	written int64
	done    bool
	err     string
}

var (
	copyMu      sync.Mutex
	copyNextID  atomic.Int64
	copyByID    = map[int64]*copySession{}
)

func registerCopyFns(m *object.Module) {
	m.Attrs["התחל_העתקה"] = &object.Builtin{Fn: filesStartCopy}
	m.Attrs["צעד_העתקה"] = &object.Builtin{Fn: filesStepCopy}
	m.Attrs["בטל_העתקה"] = &object.Builtin{Fn: filesCancelCopy}
}

func filesStartCopy(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.התחל_העתקה", 2, args); err != nil {
		return err
	}
	src, ok1 := asString(args[0])
	dst, ok2 := asString(args[1])
	if !ok1 || !ok2 {
		return errObj("קבצים.התחל_העתקה מצפה ל־(מקור, יעד)")
	}
	if dir := filepath.Dir(dst); dir != "" && dir != "." {
		if e := os.MkdirAll(dir, 0755); e != nil {
			return errObj("קבצים.התחל_העתקה: לא הצלחתי ליצור תיקיית יעד: " + e.Error())
		}
	}
	info, e := os.Stat(src)
	if e != nil {
		return errObj("קבצים.התחל_העתקה: מקור לא נמצא: " + e.Error())
	}
	in, e := os.Open(src)
	if e != nil {
		return errObj("קבצים.התחל_העתקה: לא הצלחתי לפתוח מקור: " + e.Error())
	}
	out, e := os.Create(dst)
	if e != nil {
		in.Close()
		return errObj("קבצים.התחל_העתקה: לא הצלחתי ליצור יעד: " + e.Error())
	}
	id := copyNextID.Add(1)
	copyMu.Lock()
	copyByID[id] = &copySession{in: in, out: out, total: info.Size()}
	copyMu.Unlock()
	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":   &object.Number{Value: float64(id)},
		"סהכ":    &object.Number{Value: float64(info.Size())},
		"הועתק":  &object.Number{Value: 0},
		"הסתיים": &object.Boolean{Value: false},
	}}
}

func filesStepCopy(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("קבצים.צעד_העתקה מצפה ל־(מזהה, [מקסימום_בתים])")
	}
	idF, ok := asNumber(args[0])
	if !ok {
		// accept hash from התחל_העתקה
		if h, okh := args[0].(*object.Hash); okh {
			if v, has := h.Pairs["מזהה"]; has {
				idF, ok = asNumber(v)
			}
		}
		if !ok {
			return errObj("קבצים.צעד_העתקה: מזהה חייב להיות מספר")
		}
	}
	maxBytes := int64(256 * 1024)
	if len(args) == 2 {
		if n, ok2 := asNumber(args[1]); ok2 && n > 0 {
			maxBytes = int64(n)
		}
	}
	id := int64(idF)
	copyMu.Lock()
	s := copyByID[id]
	copyMu.Unlock()
	if s == nil {
		return errObj("קבצים.צעד_העתקה: מזהה העתקה לא קיים")
	}
	if s.done {
		return copyStatusHash(s)
	}
	buf := make([]byte, maxBytes)
	n, e := s.in.Read(buf)
	if n > 0 {
		wn, we := s.out.Write(buf[:n])
		s.written += int64(wn)
		if we != nil {
			s.err = we.Error()
			s.done = true
			closeCopySession(id, s)
			return copyStatusHash(s)
		}
	}
	if e == io.EOF {
		s.done = true
		closeCopySession(id, s)
	} else if e != nil {
		s.err = e.Error()
		s.done = true
		closeCopySession(id, s)
	}
	return copyStatusHash(s)
}

func filesCancelCopy(args ...object.Object) object.Object {
	if err := expectArgs("קבצים.בטל_העתקה", 1, args); err != nil {
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
			return errObj("קבצים.בטל_העתקה: מזהה חייב להיות מספר")
		}
	}
	id := int64(idF)
	copyMu.Lock()
	s := copyByID[id]
	delete(copyByID, id)
	copyMu.Unlock()
	if s != nil {
		s.in.Close()
		s.out.Close()
	}
	return object.Nil
}

func closeCopySession(id int64, s *copySession) {
	s.in.Close()
	s.out.Close()
	copyMu.Lock()
	delete(copyByID, id)
	copyMu.Unlock()
}

func copyStatusHash(s *copySession) object.Object {
	pairs := map[string]object.Object{
		"הועתק":  &object.Number{Value: float64(s.written)},
		"סהכ":    &object.Number{Value: float64(s.total)},
		"הסתיים": &object.Boolean{Value: s.done},
	}
	if s.err != "" {
		pairs["שגיאה"] = &object.String{Value: s.err}
	} else {
		pairs["שגיאה"] = &object.String{Value: ""}
	}
	pct := 0.0
	if s.total > 0 {
		pct = 100.0 * float64(s.written) / float64(s.total)
		if pct > 100 {
			pct = 100
		}
	} else if s.done {
		pct = 100
	}
	pairs["אחוז"] = &object.Number{Value: pct}
	return &object.Hash{Pairs: pairs}
}

func asNumber(o object.Object) (float64, bool) {
	n, ok := o.(*object.Number)
	if !ok {
		return 0, false
	}
	return n.Value, true
}
