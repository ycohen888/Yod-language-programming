package stdlib

import (
	"sync/atomic"
	"testing"
	"time"

	"yod/internal/object"
)

func TestOnceTimerFiresAfterAsyncUISync(t *testing.T) {
	var ran atomic.Bool
	prev := object.InvokeCallable
	object.InvokeCallable = func(fn object.Object, args []object.Object) object.Object {
		if b, ok := fn.(*object.Builtin); ok && b.Fn != nil {
			return b.Fn(args...)
		}
		return object.Nil
	}
	t.Cleanup(func() { object.InvokeCallable = prev })

	setTimerUISync(func(fn func()) {
		// מדמה Synchronize של Walk — לא מריץ מיד
		go func() {
			time.Sleep(20 * time.Millisecond)
			fn()
		}()
	})
	t.Cleanup(func() { setTimerUISync(nil) })

	fn := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		ran.Store(true)
		return object.Nil
	}}
	id := registerTimer(fn, 30*time.Millisecond, true)
	if _, ok := id.(*object.Number); !ok {
		t.Fatalf("expected timer id number, got %#v", id)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ran.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("טיימרים.אחרי לא הריץ את הקולבק אחרי Synchronize אסינכרוני")
}
