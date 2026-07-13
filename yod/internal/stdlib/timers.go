package stdlib

import (
	"sync"
	"sync/atomic"
	"time"

	"yod/internal/object"
)

var (
	timerMu     sync.Mutex
	timerNextID int64
	timersByID  = map[int64]*yodTimer{}
	uiSyncFn    func(func())
)

type yodTimer struct {
	id       int64
	fn       object.Object
	interval time.Duration
	once     bool
	stopCh   chan struct{}
	paused   atomic.Bool
	stopped  atomic.Bool
}

// setTimerUISync — כשחלון GUI פתוח, מריץ callbacks בחוט ה־UI.
func setTimerUISync(fn func(func())) {
	timerMu.Lock()
	uiSyncFn = fn
	timerMu.Unlock()
}

func NewTimersModule() *object.Module {
	m := &object.Module{Name: "טיימרים", Attrs: map[string]object.Object{}}
	m.Attrs["כל_כמה"] = &object.Builtin{Fn: timerEvery}
	m.Attrs["פעם_אחת"] = &object.Builtin{Fn: timerOnce}
	m.Attrs["עצור"] = &object.Builtin{Fn: timerStop}
	m.Attrs["עצור_הכל"] = &object.Builtin{Fn: timerStopAll}
	m.Attrs["השהה"] = &object.Builtin{Fn: timerPause}
	m.Attrs["המשך"] = &object.Builtin{Fn: timerResume}
	return m
}

func timerEvery(args ...object.Object) object.Object {
	return startTimer(args, false, "טיימרים.כל_כמה")
}

func timerOnce(args ...object.Object) object.Object {
	return startTimer(args, true, "טיימרים.פעם_אחת")
}

func startTimer(args []object.Object, once bool, name string) object.Object {
	if len(args) != 2 {
		return errObj(name + " מצפה לשניות ולפונקציה")
	}
	sec, ok := args[0].(*object.Number)
	if !ok || sec.Value <= 0 {
		return errObj(name + " מצפה למספר שניות חיובי")
	}
	if !isCallable(args[1]) {
		return errObj(name + " מצפה לפונקציה")
	}
	id := atomic.AddInt64(&timerNextID, 1)
	t := &yodTimer{
		id:       id,
		fn:       args[1],
		interval: time.Duration(sec.Value * float64(time.Second)),
		once:     once,
		stopCh:   make(chan struct{}),
	}
	if t.interval < 50*time.Millisecond {
		t.interval = 50 * time.Millisecond
	}
	timerMu.Lock()
	timersByID[id] = t
	timerMu.Unlock()

	go runTimer(t)
	return &object.Number{Value: float64(id)}
}

func runTimer(t *yodTimer) {
	defer func() {
		timerMu.Lock()
		delete(timersByID, t.id)
		timerMu.Unlock()
	}()

	if t.once {
		select {
		case <-time.After(t.interval):
			if !t.stopped.Load() && !t.paused.Load() {
				fireTimer(t)
			}
		case <-t.stopCh:
		}
		t.stopped.Store(true)
		return
	}

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if t.stopped.Load() {
				return
			}
			if t.paused.Load() {
				continue
			}
			fireTimer(t)
		case <-t.stopCh:
			return
		}
	}
}

func fireTimer(t *yodTimer) {
	timerMu.Lock()
	syncFn := uiSyncFn
	timerMu.Unlock()
	run := func() {
		if !t.stopped.Load() {
			invokeYod(t.fn, nil)
		}
	}
	if syncFn != nil {
		syncFn(run)
		return
	}
	run()
}

func timerStop(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טיימרים.עצור מצפה למזהה")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("טיימרים.עצור מצפה למספר מזהה")
	}
	id := int64(n.Value)
	timerMu.Lock()
	t, exists := timersByID[id]
	if exists {
		delete(timersByID, id)
	}
	timerMu.Unlock()
	if exists && t != nil && !t.stopped.Swap(true) {
		close(t.stopCh)
	}
	return object.Nil
}

func timerStopAll(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("טיימרים.עצור_הכל מצפה ל־0 ארגומנטים")
	}
	timerMu.Lock()
	list := make([]*yodTimer, 0, len(timersByID))
	for _, t := range timersByID {
		list = append(list, t)
	}
	timersByID = map[int64]*yodTimer{}
	timerMu.Unlock()
	for _, t := range list {
		if t != nil && !t.stopped.Swap(true) {
			close(t.stopCh)
		}
	}
	return object.Nil
}

func timerPause(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טיימרים.השהה מצפה למזהה")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("טיימרים.השהה מצפה למספר מזהה")
	}
	timerMu.Lock()
	t := timersByID[int64(n.Value)]
	timerMu.Unlock()
	if t != nil {
		t.paused.Store(true)
	}
	return object.Nil
}

func timerResume(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("טיימרים.המשך מצפה למזהה")
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("טיימרים.המשך מצפה למספר מזהה")
	}
	timerMu.Lock()
	t := timersByID[int64(n.Value)]
	timerMu.Unlock()
	if t != nil {
		t.paused.Store(false)
	}
	return object.Nil
}
