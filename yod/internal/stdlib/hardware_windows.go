//go:build windows

package stdlib

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"time"

	"go.bug.st/serial"

	"yod/internal/object"
)

func hardwareList(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("חומרה.רשימה מצפה ל־0 ארגומנטים")
	}
	ports, err := serial.GetPortsList()
	if err != nil {
		return errObj("רשימת פורטים נכשלה: " + err.Error())
	}
	els := make([]object.Object, len(ports))
	for i, p := range ports {
		els[i] = &object.String{Value: p}
	}
	return &object.Array{Elements: els}
}

type serialPortHandle struct {
	mu      sync.Mutex
	port    serial.Port
	reader  *bufio.Reader
	timeout time.Duration
	closed  bool
}

func hardwareOpen(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 3 {
		return errObj("חומרה.פתח מצפה ל־(שם_פורט, מהירות [, timeout_מ״ש])")
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("שם פורט חייב להיות מחרוזת (למשל \"COM3\")")
	}
	baudN, ok := args[1].(*object.Number)
	if !ok {
		return errObj("מהירות (baud) חייבת להיות מספר")
	}
	timeoutMs := 1000
	if len(args) == 3 {
		t, ok := args[2].(*object.Number)
		if !ok {
			return errObj("timeout חייב להיות מספר במילישניות")
		}
		timeoutMs = int(t.Value)
		if timeoutMs < 1 {
			timeoutMs = 1
		}
	}
	mode := &serial.Mode{BaudRate: int(baudN.Value)}
	port, err := serial.Open(name, mode)
	if err != nil {
		return errObj(fmt.Sprintf("פתיחת פורט %s נכשלה: %v", name, err))
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond
	_ = port.SetReadTimeout(timeout)
	h := &serialPortHandle{
		port:    port,
		reader:  bufio.NewReader(port),
		timeout: timeout,
	}
	return newSerialHandle(h, name)
}

func newSerialHandle(h *serialPortHandle, name string) *object.Module {
	m := &object.Module{Name: "פורט_" + name, Attrs: map[string]object.Object{}}
	m.Attrs["כתוב"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return serialWrite(h, args...)
	}}
	m.Attrs["קרא_שורה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return serialReadLine(h, args...)
	}}
	m.Attrs["זמין"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return serialAvailable(h, args...)
	}}
	m.Attrs["סגור"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return serialClose(h, args...)
	}}
	return m
}

func serialWrite(h *serialPortHandle, args ...object.Object) object.Object {
	if err := expectArgs("כתוב", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("כתוב מצפה למחרוזת")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errObj("הפורט כבר סגור")
	}
	n, err := h.port.Write([]byte(s))
	if err != nil {
		return errObj("כתיבה לפורט נכשלה: " + err.Error())
	}
	return &object.Number{Value: float64(n)}
}

func serialReadLine(h *serialPortHandle, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("קרא_שורה מצפה ל־0 ארגומנטים")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errObj("הפורט כבר סגור")
	}
	_ = h.port.SetReadTimeout(h.timeout)
	line, err := h.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return errObj("קריאה מפורט נכשלה: " + err.Error())
	}
	return &object.String{Value: line}
}

func serialAvailable(h *serialPortHandle, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("זמין מצפה ל־0 ארגומנטים")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errObj("הפורט כבר סגור")
	}
	n := h.reader.Buffered()
	return &object.Number{Value: float64(n)}
}

func serialClose(h *serialPortHandle, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("סגור מצפה ל־0 ארגומנטים")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return &object.Null{}
	}
	if err := h.port.Close(); err != nil {
		return errObj("סגירת פורט נכשלה: " + err.Error())
	}
	h.closed = true
	return &object.Null{}
}
