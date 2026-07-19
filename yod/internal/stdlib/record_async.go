package stdlib

import (
	"sync"
	"sync/atomic"

	"yod/internal/object"
)

// גרסאות אסינכרוניות לפעולות הקלטה החוסמות (צריבת FFmpeg, מניית מיקרופונים, עצירה/remux)
// בדפוס handle — כדי לא לחסום את ה-UI. הפעולה רצה ב-goroutine; יוד עושה פוללינג.

type recAsyncSession struct {
	mu     sync.Mutex
	done   bool
	errStr string
	result object.Object
}

var (
	recAsyncMu   sync.Mutex
	recAsyncNext atomic.Int64
	recAsyncByID = map[int64]*recAsyncSession{}
)

func registerRecordAsync(m *object.Module) {
	m.Attrs["החל_טקסטים_התחל"] = &object.Builtin{Fn: recordApplyTextsStart}
	m.Attrs["מצב_צריבה"] = &object.Builtin{Fn: recordBurnStatus}
	m.Attrs["בטל_צריבה"] = &object.Builtin{Fn: recordBurnCancel}
	m.Attrs["רשימת_מיקרופונים_התחל"] = &object.Builtin{Fn: recordListMicsStart}
	m.Attrs["מצב_מיקרופונים"] = &object.Builtin{Fn: recordMicsStatus}
	m.Attrs["עצור_התחל"] = &object.Builtin{Fn: recordStopStart}
	m.Attrs["מצב_עצירה"] = &object.Builtin{Fn: recordStopStatus}
}

func startRecAsync(fn func(...object.Object) object.Object, args []object.Object) int64 {
	id := recAsyncNext.Add(1)
	s := &recAsyncSession{}
	recAsyncMu.Lock()
	recAsyncByID[id] = s
	recAsyncMu.Unlock()

	cp := make([]object.Object, len(args))
	for i, a := range args {
		cp[i] = object.DeepCopy(a)
	}
	go func() {
		res := fn(cp...)
		s.mu.Lock()
		s.done = true
		if e, ok := res.(*object.Error); ok {
			s.errStr = e.Message
		} else {
			s.result = res
		}
		s.mu.Unlock()
	}()
	return id
}

// recAsyncRead — מחזיר done/err/result, ומסיר מהמפה אם הסתיים.
func recAsyncRead(id int64) (done bool, errStr string, result object.Object, exists bool) {
	recAsyncMu.Lock()
	s := recAsyncByID[id]
	recAsyncMu.Unlock()
	if s == nil {
		return false, "", nil, false
	}
	s.mu.Lock()
	done = s.done
	errStr = s.errStr
	result = s.result
	s.mu.Unlock()
	if done {
		recAsyncMu.Lock()
		delete(recAsyncByID, id)
		recAsyncMu.Unlock()
	}
	return done, errStr, result, true
}

func startHandleHash(id int64) object.Object {
	return &object.Hash{Pairs: map[string]object.Object{
		"מזהה":    &object.Number{Value: float64(id)},
		"הסתיים": &object.Boolean{Value: false},
	}}
}

// הקלטה.החל_טקסטים_התחל(קלט, פלט, שכבות)
func recordApplyTextsStart(args ...object.Object) object.Object {
	if len(args) != 3 {
		return errObj("הקלטה.החל_טקסטים_התחל מצפה ל־קלט, פלט, רשימת_טקסטים")
	}
	return startHandleHash(startRecAsync(recordApplyTexts, args))
}

// הקלטה.מצב_צריבה(מזהה) — {הסתיים, הצליח, פלט, שגיאה}
func recordBurnStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הקלטה.מצב_צריבה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("הקלטה.מצב_צריבה: מזהה חייב להיות מספר")
	}
	done, errStr, result, exists := recAsyncRead(id)
	if !exists {
		return errObj("הקלטה.מצב_צריבה: מזהה צריבה לא קיים")
	}
	out := map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"הצליח":  &object.Boolean{Value: done && errStr == ""},
		"פלט":     &object.String{Value: ""},
		"שגיאה":  &object.String{Value: errStr},
	}
	if s, ok := result.(*object.String); ok {
		out["פלט"] = s
	}
	return &object.Hash{Pairs: out}
}

// הקלטה.בטל_צריבה(מזהה) — משחרר את ההַנְדֶל
func recordBurnCancel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הקלטה.בטל_צריבה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("הקלטה.בטל_צריבה: מזהה חייב להיות מספר")
	}
	recAsyncMu.Lock()
	delete(recAsyncByID, id)
	recAsyncMu.Unlock()
	return object.Nil
}

// הקלטה.רשימת_מיקרופונים_התחל()
func recordListMicsStart(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.רשימת_מיקרופונים_התחל מצפה ל־0 ארגומנטים")
	}
	return startHandleHash(startRecAsync(recordListMics, nil))
}

// הקלטה.מצב_מיקרופונים(מזהה) — {הסתיים, הצליח, רשימה, שגיאה}
func recordMicsStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הקלטה.מצב_מיקרופונים מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("הקלטה.מצב_מיקרופונים: מזהה חייב להיות מספר")
	}
	done, errStr, result, exists := recAsyncRead(id)
	if !exists {
		return errObj("הקלטה.מצב_מיקרופונים: מזהה לא קיים")
	}
	list := object.Object(&object.Array{Elements: []object.Object{}})
	if arr, ok := result.(*object.Array); ok {
		list = arr
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"הצליח":  &object.Boolean{Value: done && errStr == ""},
		"רשימה":  list,
		"שגיאה":  &object.String{Value: errStr},
	}}
}

// הקלטה.עצור_התחל() — עוצר את ההקלטה ומבצע remux ברקע
func recordStopStart(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.עצור_התחל מצפה ל־0 ארגומנטים")
	}
	return startHandleHash(startRecAsync(recordStop, nil))
}

// הקלטה.מצב_עצירה(מזהה) — {הסתיים, קובץ, שגיאה}
func recordStopStatus(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הקלטה.מצב_עצירה מצפה למזהה")
	}
	id, ok := asyncHandleID(args[0])
	if !ok {
		return errObj("הקלטה.מצב_עצירה: מזהה חייב להיות מספר")
	}
	done, errStr, result, exists := recAsyncRead(id)
	if !exists {
		return errObj("הקלטה.מצב_עצירה: מזהה לא קיים")
	}
	file := ""
	if s, ok := result.(*object.String); ok {
		file = s.Value
	}
	return &object.Hash{Pairs: map[string]object.Object{
		"הסתיים": &object.Boolean{Value: done},
		"קובץ":    &object.String{Value: file},
		"שגיאה":  &object.String{Value: errStr},
	}}
}
