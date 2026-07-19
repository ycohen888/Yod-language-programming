package stdlib

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yod/internal/object"
)

// גרסאות אסינכרוניות של בקשות HTTP בדפוס handle (התחל → מצב → בטל), כדי לא לחסום את
// ה-UI. הבקשה רצה ב-goroutine; יוד מבצע פוללינג עם `רשת.מצב_בקשה` דרך טיימר.

type httpAsyncSession struct {
	mu     sync.Mutex
	done   bool
	errStr string
	result object.Object // *object.Hash עם קוד/גוף/כותרות בהצלחה
	ms     int64
}

var (
	httpAsyncMu   sync.Mutex
	httpAsyncNext atomic.Int64
	httpAsyncByID = map[int64]*httpAsyncSession{}
)

func registerHTTPAsync(m *object.Module) {
	m.Attrs["גש_התחל"] = &object.Builtin{Fn: netGetStart}
	m.Attrs["פרסם_התחל"] = &object.Builtin{Fn: netPostStart}
	m.Attrs["פרסם_json_התחל"] = &object.Builtin{Fn: netPostJSONStart}
	m.Attrs["בקשה_התחל"] = &object.Builtin{Fn: netRequestStart}
	m.Attrs["מצב_בקשה"] = &object.Builtin{Fn: netRequestStatus}
	m.Attrs["בטל_בקשה"] = &object.Builtin{Fn: netRequestCancel}
}

func startHTTPAsync(method, url, body string, headers map[string]string) object.Object {
	id := httpAsyncNext.Add(1)
	s := &httpAsyncSession{}
	httpAsyncMu.Lock()
	httpAsyncByID[id] = s
	httpAsyncMu.Unlock()

	go func() {
		t0 := time.Now()
		res := doHTTP(method, url, body, headers)
		ms := time.Since(t0).Milliseconds()
		s.mu.Lock()
		s.done = true
		s.ms = ms
		if e, ok := res.(*object.Error); ok {
			s.errStr = e.Message
		} else {
			s.result = res
		}
		s.mu.Unlock()
	}()

	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":    &object.Number{Value: float64(id)},
		"הסתיים": &object.Boolean{Value: false},
	}}
}

// רשת.גש_התחל(כתובת)
func netGetStart(args ...object.Object) object.Object {
	if err := expectArgs("רשת.גש_התחל", 1, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.גש_התחל מצפה לכתובת מחרוזת")
	}
	return startHTTPAsync("GET", url, "", nil)
}

// רשת.פרסם_התחל(כתובת, גוף)
func netPostStart(args ...object.Object) object.Object {
	if err := expectArgs("רשת.פרסם_התחל", 2, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.פרסם_התחל: כתובת חייבת להיות מחרוזת")
	}
	body := bodyString(args[1])
	return startHTTPAsync("POST", url, body, map[string]string{
		"Content-Type": "text/plain; charset=utf-8",
	})
}

// רשת.פרסם_json_התחל(כתובת, גוף)
func netPostJSONStart(args ...object.Object) object.Object {
	if err := expectArgs("רשת.פרסם_json_התחל", 2, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.פרסם_json_התחל: כתובת חייבת להיות מחרוזת")
	}
	body := bodyString(args[1])
	return startHTTPAsync("POST", url, body, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	})
}

// רשת.בקשה_התחל(שיטה, כתובת [, גוף [, כותרות]])
func netRequestStart(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 4 {
		return errObj("רשת.בקשה_התחל מצפה ל־2 עד 4 ארגומנטים: שיטה, כתובת [, גוף [, כותרות]]")
	}
	method, ok := asString(args[0])
	if !ok {
		return errObj("רשת.בקשה_התחל: שיטה חייבת להיות מחרוזת")
	}
	url, ok := asString(args[1])
	if !ok {
		return errObj("רשת.בקשה_התחל: כתובת חייבת להיות מחרוזת")
	}
	body := ""
	if len(args) >= 3 {
		body = bodyString(args[2])
	}
	var headers map[string]string
	if len(args) >= 4 {
		h, err := hashToStringMap(args[3])
		if err != nil {
			return err
		}
		headers = h
	}
	return startHTTPAsync(strings.ToUpper(method), url, body, headers)
}

// asyncHandleID — מחלץ מזהה ממספר או ממילון עם שדה "מזהה"
func asyncHandleID(arg object.Object) (int64, bool) {
	if f, ok := asNumber(arg); ok {
		return int64(f), true
	}
	if h, ok := arg.(*object.Hash); ok {
		if v, has := h.Pairs["מזהה"]; has {
			if f, ok := asNumber(v); ok {
				return int64(f), true
			}
		}
	}
	return 0, false
}

// רשת.מצב_בקשה(מזהה) — {הסתיים, הצליח, קוד, גוף, כותרות, שגיאה, זמן_ms}
func netRequestStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשת.מצב_בקשה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("רשת.מצב_בקשה: מזהה חייב להיות מספר")
	}
	httpAsyncMu.Lock()
	s := httpAsyncByID[id]
	httpAsyncMu.Unlock()
	if s == nil {
		return errObj("רשת.מצב_בקשה: מזהה בקשה לא קיים")
	}

	s.mu.Lock()
	done := s.done
	errStr := s.errStr
	result := s.result
	ms := s.ms
	s.mu.Unlock()

	out := map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"הצליח":  &object.Boolean{Value: done && errStr == ""},
		"שגיאה":  &object.String{Value: errStr},
		"קוד":     &object.Number{Value: 0},
		"גוף":     &object.String{Value: ""},
		"כותרות": object.Nil,
		"זמן_ms":  &object.Number{Value: float64(ms)},
	}
	if done {
		// מסירים מהמפה אחרי קריאה שהסתיימה — מונע דליפה ומרוץ פולים
		httpAsyncMu.Lock()
		delete(httpAsyncByID, id)
		httpAsyncMu.Unlock()
		if hh, ok := result.(*object.Hash); ok {
			if v, ok := hh.Pairs["קוד"]; ok {
				out["קוד"] = v
			}
			if v, ok := hh.Pairs["גוף"]; ok {
				out["גוף"] = v
			}
			if v, ok := hh.Pairs["כותרות"]; ok {
				out["כותרות"] = v
			}
		}
	}
	return &object.Hash{Pairs: out}
}

// רשת.בטל_בקשה(מזהה) — משחרר את ההַנְדֶל (הבקשה עצמה ממשיכה עד timeout)
func netRequestCancel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשת.בטל_בקשה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("רשת.בטל_בקשה: מזהה חייב להיות מספר")
	}
	httpAsyncMu.Lock()
	delete(httpAsyncByID, id)
	httpAsyncMu.Unlock()
	return object.Nil
}
