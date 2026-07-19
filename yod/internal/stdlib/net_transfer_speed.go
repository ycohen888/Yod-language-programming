package stdlib

import (
	"sync"
	"time"

	"yod/internal/object"
)

// transferSpeedTracker — קצב העברה לפי דגימות בין קריאות מצב.
type transferSpeedTracker struct {
	mu         sync.Mutex
	started    time.Time
	lastBytes  int64
	lastTime   time.Time
	instantBps float64
}

func newTransferSpeedTracker() *transferSpeedTracker {
	now := time.Now()
	return &transferSpeedTracker{started: now, lastTime: now}
}

// sample מחזיר מהירות רגעית (בין דגימות) וממוצעת מתחילת ההעברה — בבתים לשנייה.
func (t *transferSpeedTracker) sample(written int64) (instant, average float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if t.started.IsZero() {
		t.started = now
		t.lastTime = now
		t.lastBytes = written
	}
	elapsed := now.Sub(t.lastTime).Seconds()
	if elapsed >= 0.08 {
		delta := written - t.lastBytes
		if delta < 0 {
			delta = 0
		}
		if delta > 0 || written == 0 {
			t.instantBps = float64(delta) / elapsed
		}
		// אם לא זרמו בתים בדגימה — משאירים את המהירות הקודמת (לא מאפסים)
		t.lastBytes = written
		t.lastTime = now
	}
	instant = t.instantBps
	totalSec := now.Sub(t.started).Seconds()
	if totalSec > 0.05 && written > 0 {
		average = float64(written) / totalSec
	}
	// לתצוגה: אם הרגעית עדיין 0 — משתמשים בממוצע
	if instant <= 0 && average > 0 {
		instant = average
	}
	return
}

func speedStatusFields(instant, average float64) map[string]object.Object {
	display := instant
	if display <= 0 {
		display = average
	}
	return map[string]object.Object{
		"מהירות":         &object.Number{Value: display},
		"מהירות_רגעית":  &object.Number{Value: instant},
		"מהירות_ממוצעת": &object.Number{Value: average},
	}
}

// רשת.חשב_מהירות(בתים, שניות) או (בתים_קודם, בתים_עכשיו, שניות) → בתים/שנייה
func netCalcSpeed(args ...object.Object) object.Object {
	if len(args) != 2 && len(args) != 3 {
		return errObj("רשת.חשב_מהירות מצפה ל־(בתים, שניות) או (בתים_קודם, בתים_עכשיו, שניות)")
	}
	var bytes float64
	var secs float64
	if len(args) == 2 {
		b, ok1 := asNumber(args[0])
		s, ok2 := asNumber(args[1])
		if !ok1 || !ok2 {
			return errObj("רשת.חשב_מהירות: כל הארגומנטים חייבים להיות מספרים")
		}
		bytes = b
		secs = s
	} else {
		b0, ok0 := asNumber(args[0])
		b1, ok1 := asNumber(args[1])
		s, ok2 := asNumber(args[2])
		if !ok0 || !ok1 || !ok2 {
			return errObj("רשת.חשב_מהירות: כל הארגומנטים חייבים להיות מספרים")
		}
		bytes = b1 - b0
		secs = s
	}
	if secs <= 0 {
		return &object.Number{Value: 0}
	}
	if bytes < 0 {
		bytes = 0
	}
	return &object.Number{Value: bytes / secs}
}
