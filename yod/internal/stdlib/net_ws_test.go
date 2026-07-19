package stdlib

import (
	"sync"
	"testing"
	"time"

	"yod/internal/object"
)

func TestWebSocketEcho(t *testing.T) {
	object.InvokeCallable = func(fn object.Object, args []object.Object) object.Object {
		switch f := fn.(type) {
		case *object.Builtin:
			return f.Fn(args...)
		default:
			return object.Nil
		}
	}
	t.Cleanup(func() { object.InvokeCallable = nil })

	var gotMu sync.Mutex
	var got string
	var gotClose bool

	port := 18765
	s := netWSServer(&object.Number{Value: float64(port)}, &object.String{Value: "/echo"})
	sm, ok := s.(*object.Module)
	if !ok {
		t.Fatalf("server type %#v", s)
	}
	sm.Attrs["בהתחברות"].(*object.Builtin).Fn(&object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			t.Fatalf("connect args")
		}
		sock := a[0].(*object.Module)
		sock.Attrs["בהודעה"].(*object.Builtin).Fn(&object.Builtin{Fn: func(a ...object.Object) object.Object {
			if len(a) == 1 {
				if msg, ok := a[0].(*object.String); ok {
					_ = sock.Attrs["שלח"].(*object.Builtin).Fn(&object.String{Value: "echo:" + msg.Value})
				}
			}
			return object.Nil
		}})
		return object.Nil
	}})
	start := sm.Attrs["הפעל_ברקע"].(*object.Builtin).Fn()
	if _, ok := start.(*object.Error); ok {
		t.Fatalf("start: %#v", start)
	}
	t.Cleanup(func() { sm.Attrs["עצור"].(*object.Builtin).Fn() })
	time.Sleep(80 * time.Millisecond)

	cli := netWSConnect(&object.String{Value: "ws://127.0.0.1:18765/echo"})
	cm, ok := cli.(*object.Module)
	if !ok {
		t.Fatalf("client: %#v", cli)
	}
	cm.Attrs["בהודעה"].(*object.Builtin).Fn(&object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) == 1 {
			if s, ok := a[0].(*object.String); ok {
				gotMu.Lock()
				got = s.Value
				gotMu.Unlock()
			}
		}
		return object.Nil
	}})
	cm.Attrs["בסגירה"].(*object.Builtin).Fn(&object.Builtin{Fn: func(a ...object.Object) object.Object {
		gotMu.Lock()
		gotClose = true
		gotMu.Unlock()
		return object.Nil
	}})
	send := cm.Attrs["שלח"].(*object.Builtin).Fn(&object.String{Value: "שלום"})
	if _, ok := send.(*object.Error); ok {
		t.Fatalf("send: %#v", send)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gotMu.Lock()
		g := got
		gotMu.Unlock()
		if g == "echo:שלום" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	gotMu.Lock()
	g := got
	gotMu.Unlock()
	if g != "echo:שלום" {
		t.Fatalf("got %q", g)
	}
	_ = cm.Attrs["סגור"].(*object.Builtin).Fn()
	_ = gotClose
}
