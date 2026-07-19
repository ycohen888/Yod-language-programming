package stdlib

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"yod/internal/object"
)

type httpServerState struct {
	mu       sync.Mutex
	mux      *http.ServeMux
	srv      *http.Server
	addr     string
	running  bool
}

// רשת.שרת(פורט [, כתובת_האזנה]) — יוצר שרת HTTP (ברירת מחדל 127.0.0.1)
func netNewServer(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("רשת.שרת מצפה לפורט [, כתובת_האזנה]")
	}
	port := 0
	switch v := args[0].(type) {
	case *object.Number:
		port = int(v.Value)
	case *object.String:
		p, err := strconv.Atoi(v.Value)
		if err != nil {
			return errObj("רשת.שרת: פורט לא תקין")
		}
		port = p
	default:
		return errObj("רשת.שרת מצפה למספר פורט")
	}
	if port <= 0 || port > 65535 {
		return errObj("רשת.שרת: פורט מחוץ לטווח")
	}
	host := "127.0.0.1"
	if len(args) == 2 {
		h, ok := asString(args[1])
		if !ok {
			return errObj("רשת.שרת: כתובת האזנה חייבת להיות מחרוזת (127.0.0.1 / 0.0.0.0)")
		}
		h = strings.TrimSpace(h)
		if h == "" || h == "localhost" {
			host = "127.0.0.1"
		} else if h == "0.0.0.0" || h == "*" || h == "כל" {
			host = "0.0.0.0"
		} else {
			host = h
		}
	}
	st := &httpServerState{
		mux:  http.NewServeMux(),
		addr: fmt.Sprintf("%s:%d", host, port),
	}
	displayHost := host
	if host == "0.0.0.0" {
		displayHost = "127.0.0.1"
	}
	m := &object.Module{Name: "שרת_HTTP", Attrs: map[string]object.Object{}}
	m.Attrs["כתובת"] = &object.String{Value: "http://" + displayHost + fmt.Sprintf(":%d", port)}
	m.Attrs["כתובת_האזנה"] = &object.String{Value: st.addr}
	m.Attrs["מסלול"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return netServerRoute(st, a...)
	}}
	m.Attrs["הפעל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return netServerStart(st, false, a...)
	}}
	m.Attrs["הפעל_ברקע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return netServerStart(st, true, a...)
	}}
	m.Attrs["עצור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return netServerStop(st)
	}}
	m.Attrs["פועל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("שרת.פועל מצפה ל־0 ארגומנטים")
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		return &object.Boolean{Value: st.running}
	}}
	return m
}

func netServerRoute(st *httpServerState, args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("שרת.מסלול מצפה לנתיב ולפונקציה")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("שרת.מסלול: נתיב חייב להיות מחרוזת")
	}
	handler := args[1]
	switch handler.(type) {
	case *object.Function, *object.Closure, *object.CompiledFunction:
	default:
		return errObj("שרת.מסלול: מצפה לפונקציה מטפלת")
	}
	st.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		_ = r.Body.Close()
		headerHash := &object.Hash{Pairs: map[string]object.Object{}}
		for k, vals := range r.Header {
			headerHash.Pairs[k] = &object.String{Value: strings.Join(vals, ", ")}
		}
		queryHash := &object.Hash{Pairs: map[string]object.Object{}}
		for k, vals := range r.URL.Query() {
			if len(vals) == 1 {
				queryHash.Pairs[k] = &object.String{Value: vals[0]}
			} else {
				arr := &object.Array{Elements: make([]object.Object, len(vals))}
				for i, v := range vals {
					arr.Elements[i] = &object.String{Value: v}
				}
				queryHash.Pairs[k] = arr
			}
		}
		req := &object.Hash{Pairs: map[string]object.Object{
			"שיטה":    &object.String{Value: r.Method},
			"נתיב":    &object.String{Value: r.URL.Path},
			"גוף":     &object.String{Value: string(body)},
			"כותרות":  headerHash,
			"שאילתה":  queryHash,
		}}
		var res object.Object
		if object.InvokeCallable != nil {
			res = object.InvokeCallable(handler, []object.Object{req})
		} else if fn, ok := handler.(*object.Function); ok && object.InvokeFunction != nil {
			res = object.InvokeFunction(fn, []object.Object{req})
		} else {
			http.Error(w, "אין מנוע להרצת מטפל", 500)
			return
		}
		if errObj, ok := res.(*object.Error); ok {
			http.Error(w, errObj.Message, 500)
			return
		}
		writeHTTPResponse(w, res)
	})
	return &object.Null{}
}

func writeHTTPResponse(w http.ResponseWriter, res object.Object) {
	status := 200
	body := ""
	if h, ok := res.(*object.Hash); ok {
		if c, ok := h.Pairs["קוד"].(*object.Number); ok {
			status = int(c.Value)
		}
		if b, ok := h.Pairs["גוף"]; ok {
			if s, ok := b.(*object.String); ok {
				body = s.Value
			} else {
				body = b.Inspect()
			}
		}
		if headers, ok := h.Pairs["כותרות"].(*object.Hash); ok {
			for k, v := range headers.Pairs {
				if s, ok := v.(*object.String); ok {
					w.Header().Set(k, s.Value)
				} else {
					w.Header().Set(k, v.Inspect())
				}
			}
		}
	} else if s, ok := res.(*object.String); ok {
		body = s.Value
	} else if res != nil {
		body = res.Inspect()
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func netServerStart(st *httpServerState, background bool, args ...object.Object) object.Object {
	if len(args) != 0 {
		name := "שרת.הפעל"
		if background {
			name = "שרת.הפעל_ברקע"
		}
		return errObj(name + " מצפה ל־0 ארגומנטים")
	}
	st.mu.Lock()
	if st.running {
		st.mu.Unlock()
		return errObj("השרת כבר פועל")
	}
	st.srv = &http.Server{Addr: st.addr, Handler: st.mux}
	st.running = true
	st.mu.Unlock()

	ln, err := net.Listen("tcp", st.addr)
	if err != nil {
		st.mu.Lock()
		st.running = false
		st.srv = nil
		st.mu.Unlock()
		return errObj("לא הצלחתי להאזין: " + err.Error())
	}

	if background {
		go func() {
			_ = st.srv.Serve(ln)
			st.mu.Lock()
			st.running = false
			st.mu.Unlock()
		}()
		return object.Nil
	}

	err = st.srv.Serve(ln)
	st.mu.Lock()
	st.running = false
	st.mu.Unlock()
	if err != nil && err != http.ErrServerClosed {
		return errObj("שרת נעצר עם שגיאה: " + err.Error())
	}
	return object.Nil
}

func netServerStop(st *httpServerState) object.Object {
	st.mu.Lock()
	srv := st.srv
	st.mu.Unlock()
	if srv == nil {
		return &object.Null{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	return &object.Null{}
}
