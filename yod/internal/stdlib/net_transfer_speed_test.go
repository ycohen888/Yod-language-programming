package stdlib

import (
	"testing"
	"time"

	"yod/internal/object"
)

func TestNetCalcSpeed(t *testing.T) {
	got := netCalcSpeed(&object.Number{Value: 1024}, &object.Number{Value: 2})
	n, ok := got.(*object.Number)
	if !ok || n.Value != 512 {
		t.Fatalf("חשב_מהירות(בתים,שניות): got %#v", got)
	}
	got = netCalcSpeed(&object.Number{Value: 100}, &object.Number{Value: 1100}, &object.Number{Value: 2})
	n, ok = got.(*object.Number)
	if !ok || n.Value != 500 {
		t.Fatalf("חשב_מהירות(קודם,עכשיו,שניות): got %#v", got)
	}
	got = netCalcSpeed(&object.Number{Value: 100}, &object.Number{Value: 0})
	n, ok = got.(*object.Number)
	if !ok || n.Value != 0 {
		t.Fatalf("חשב_מהירות עם שניות=0: got %#v", got)
	}
}

func TestTransferSpeedTracker(t *testing.T) {
	tr := newTransferSpeedTracker()
	inst, avg := tr.sample(0)
	if inst != 0 || avg != 0 {
		t.Fatalf("דגימה ראשונה ריקה: instant=%v avg=%v", inst, avg)
	}
	past := time.Now().Add(-time.Second)
	tr.started = past
	tr.lastTime = past
	tr.lastBytes = 0
	inst, avg = tr.sample(2048)
	if inst < 2000 || inst > 2100 {
		t.Fatalf("מהירות רגעית צפויה ~2048: got %v", inst)
	}
	if avg <= 0 {
		t.Fatalf("ממוצע צריך להיות חיובי: %v", avg)
	}
	// דגימה בלי בתים חדשים — לא מאפסים
	tr.lastTime = time.Now().Add(-time.Second)
	prev := inst
	inst2, _ := tr.sample(2048)
	if inst2 != prev {
		t.Fatalf("בלי התקדמות צריך לשמור מהירות קודמת: prev=%v got=%v", prev, inst2)
	}
}
