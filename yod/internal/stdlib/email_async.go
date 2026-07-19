package stdlib

import (
	"sync"
	"sync/atomic"

	"yod/internal/object"
)

// גרסאות אסינכרוניות לשליחת אימייל (SMTP חוסם) בדפוס handle — כדי לא לחסום את ה-UI.

type emailAsyncSession struct {
	mu     sync.Mutex
	done   bool
	errStr string
}

var (
	emailAsyncMu   sync.Mutex
	emailAsyncNext atomic.Int64
	emailAsyncByID = map[int64]*emailAsyncSession{}
)

func registerEmailAsync(m *object.Module) {
	m.Attrs["שלח_התחל"] = &object.Builtin{Fn: emailSendStart}
	m.Attrs["גימייל_התחל"] = &object.Builtin{Fn: emailGmailStart}
	m.Attrs["מצב_שליחה"] = &object.Builtin{Fn: emailSendStatus}
}

func startEmailAsync(fn func(...object.Object) object.Object, opts object.Object) object.Object {
	id := emailAsyncNext.Add(1)
	s := &emailAsyncSession{}
	emailAsyncMu.Lock()
	emailAsyncByID[id] = s
	emailAsyncMu.Unlock()

	// העתקה עמוקה של המילון — הקורא עלול לשנות אותו בזמן שה-goroutine עובד
	optsCopy := object.DeepCopy(opts)

	go func() {
		res := fn(optsCopy)
		s.mu.Lock()
		s.done = true
		if e, ok := res.(*object.Error); ok {
			s.errStr = e.Message
		}
		s.mu.Unlock()
	}()

	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":    &object.Number{Value: float64(id)},
		"הסתיים": &object.Boolean{Value: false},
	}}
}

// אימייל.שלח_התחל(אפשרויות)
func emailSendStart(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.שלח_התחל מצפה למילון אפשרויות אחד")
	}
	if _, ok := args[0].(*object.Hash); !ok {
		return errObj("אימייל.שלח_התחל מצפה למילון")
	}
	return startEmailAsync(emailSend, args[0])
}

// אימייל.גימייל_התחל(אפשרויות)
func emailGmailStart(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.גימייל_התחל מצפה למילון אפשרויות אחד")
	}
	if _, ok := args[0].(*object.Hash); !ok {
		return errObj("אימייל.גימייל_התחל מצפה למילון")
	}
	return startEmailAsync(emailGmail, args[0])
}

// אימייל.מצב_שליחה(מזהה) — {הסתיים, הצליח, שגיאה}
func emailSendStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.מצב_שליחה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("אימייל.מצב_שליחה: מזהה חייב להיות מספר")
	}
	emailAsyncMu.Lock()
	s := emailAsyncByID[id]
	emailAsyncMu.Unlock()
	if s == nil {
		return errObj("אימייל.מצב_שליחה: מזהה שליחה לא קיים")
	}

	s.mu.Lock()
	done := s.done
	errStr := s.errStr
	s.mu.Unlock()

	if done {
		emailAsyncMu.Lock()
		delete(emailAsyncByID, id)
		emailAsyncMu.Unlock()
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"הצליח":  &object.Boolean{Value: done && errStr == ""},
		"שגיאה":  &object.String{Value: errStr},
	}}
}
