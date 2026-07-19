package object

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// מנעול ריצה גלובלי (GIL) לשפת יוד.
//
// המנוע מריץ קוד יוד על מצב משותף (globals, stack, environment) שאינו בטוח־thread.
// כשמשתמשים ב־`משימה` (goroutine ברקע), הקוד ברקע עלול לרוץ במקביל ל־callbacks של
// ה־UI (לחיצות/טיימרים) וליצור data race או קריסה. המנעול הזה מבטיח שרק "מבצע" אחד
// מריץ קוד יוד בכל רגע.
//
// המנעול הוא re-entrant לפי goroutine: אותה goroutine יכולה לנעול שוב בלי deadlock
// (חשוב כי callback עלול לגרור callback נוסף באותה goroutine). שחרור מלא זמני
// (WithoutYodLock) משמש בזמן פעולות חוסמות (mw.Run, המתן, I/O) כדי לאפשר למבצעים
// אחרים לרוץ בינתיים.
type reentrantLock struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner int64 // goroutine id של המחזיק; 0 = פנוי
	depth int
}

func newReentrantLock() *reentrantLock {
	l := &reentrantLock{}
	l.cond = sync.NewCond(&l.mu)
	return l
}

var yodLock = newReentrantLock()

func (l *reentrantLock) lock() {
	gid := goID()
	l.mu.Lock()
	for l.owner != 0 && l.owner != gid {
		l.cond.Wait()
	}
	l.owner = gid
	l.depth++
	l.mu.Unlock()
}

func (l *reentrantLock) unlock() {
	gid := goID()
	l.mu.Lock()
	if l.owner != gid {
		// שחרור בלי בעלות — התעלמות בטוחה (לא אמור לקרות)
		l.mu.Unlock()
		return
	}
	l.depth--
	if l.depth <= 0 {
		l.depth = 0
		l.owner = 0
		l.cond.Signal()
	}
	l.mu.Unlock()
}

// fullyRelease — משחרר לגמרי את המנעול (אם ה־goroutine הנוכחית מחזיקה בו) ומחזיר את
// העומק כדי לשחזר אחר כך. אם אינה מחזיקה — מחזיר 0 ולא נוגע במנעול.
func (l *reentrantLock) fullyRelease() int {
	gid := goID()
	l.mu.Lock()
	if l.owner != gid {
		l.mu.Unlock()
		return 0
	}
	d := l.depth
	l.depth = 0
	l.owner = 0
	l.cond.Signal()
	l.mu.Unlock()
	return d
}

func (l *reentrantLock) reacquire(depth int) {
	if depth <= 0 {
		return
	}
	gid := goID()
	l.mu.Lock()
	for l.owner != 0 && l.owner != gid {
		l.cond.Wait()
	}
	l.owner = gid
	l.depth = depth
	l.mu.Unlock()
}

// LockYod — נועל את מנעול הריצה של יוד (re-entrant). כל כניסה חיצונית להרצת קוד יוד
// (הרצת התוכנית הראשית, callback של UI/טיימר, goroutine של משימה) צריכה לעטוף בו.
func LockYod() { yodLock.lock() }

// UnlockYod — משחרר שכבה אחת של המנעול.
func UnlockYod() { yodLock.unlock() }

// WithoutYodLock — משחרר את המנעול לגמרי בזמן פעולה חוסמת (mw.Run, המתן, I/O),
// מריץ את fn, ואז תופס מחדש. מאפשר למבצעים אחרים (UI/משימות) לרוץ בינתיים.
// אם ה־goroutine הנוכחית אינה מחזיקה במנעול — פשוט מריץ את fn.
func WithoutYodLock(fn func()) {
	d := yodLock.fullyRelease()
	defer yodLock.reacquire(d)
	fn()
}

// goID — מזהה ה־goroutine הנוכחית. משמש רק בגבולות (callbacks/משימות/פעולות חוסמות),
// לא בלולאת הפקודות, ולכן העלות זניחה.
func goID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// פורמט: "goroutine 123 [running]:\n..."
	s := buf[:n]
	s = bytes.TrimPrefix(s, []byte("goroutine "))
	i := bytes.IndexByte(s, ' ')
	if i < 0 {
		return 0
	}
	id, _ := strconv.ParseInt(string(s[:i]), 10, 64)
	return id
}
