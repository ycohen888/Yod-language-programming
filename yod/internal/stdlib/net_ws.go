package stdlib

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"yod/internal/object"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsConnState struct {
	mu      sync.Mutex
	conn    *websocket.Conn
	onMsg   object.Object
	onClose object.Object
	closed  bool
	server  *wsServerState // optional back-ref
}

type wsServerState struct {
	mu        sync.Mutex
	addr      string
	path      string
	srv       *http.Server
	running   bool
	clients   map[*wsConnState]bool
	onConnect object.Object
	onMsg     object.Object // optional: all messages (שקע, טקסט)
}

func runOnUI(fn func()) {
	timerMu.Lock()
	syncFn := uiSyncFn
	timerMu.Unlock()
	if syncFn != nil {
		syncFn(fn)
		return
	}
	fn()
}

// רשת.שקע_שרת(פורט [, נתיב])
func netWSServer(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("רשת.שקע_שרת מצפה לפורט ולנתיב אופציונלי")
	}
	port := 0
	switch v := args[0].(type) {
	case *object.Number:
		port = int(v.Value)
	case *object.String:
		p, err := strconv.Atoi(v.Value)
		if err != nil {
			return errObj("רשת.שקע_שרת: פורט לא תקין")
		}
		port = p
	default:
		return errObj("רשת.שקע_שרת מצפה למספר פורט")
	}
	if port <= 0 || port > 65535 {
		return errObj("רשת.שקע_שרת: פורט מחוץ לטווח")
	}
	path := "/"
	if len(args) == 2 {
		s, ok := asString(args[1])
		if !ok || s == "" {
			return errObj("רשת.שקע_שרת: נתיב חייב להיות מחרוזת")
		}
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		path = s
	}
	st := &wsServerState{
		addr:    fmt.Sprintf(":%d", port),
		path:    path,
		clients: map[*wsConnState]bool{},
	}
	m := &object.Module{Name: "שקע_שרת", Attrs: map[string]object.Object{}}
	m.Attrs["כתובת"] = &object.String{Value: fmt.Sprintf("ws://127.0.0.1:%d%s", port, path)}
	m.Attrs["בהתחברות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("שקע_שרת.בהתחברות מצפה לפונקציה")
		}
		st.mu.Lock()
		st.onConnect = a[0]
		st.mu.Unlock()
		return object.Nil
	}}
	m.Attrs["בהודעה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("שקע_שרת.בהודעה מצפה לפונקציה (שקע, טקסט)")
		}
		st.mu.Lock()
		st.onMsg = a[0]
		st.mu.Unlock()
		return object.Nil
	}}
	m.Attrs["שדר"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsServerBroadcast(st, a...)
	}}
	m.Attrs["הפעל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsServerStart(st, false, a...)
	}}
	m.Attrs["הפעל_ברקע"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsServerStart(st, true, a...)
	}}
	m.Attrs["עצור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsServerStop(st, a...)
	}}
	m.Attrs["פועל"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("שקע_שרת.פועל מצפה ל־0 ארגומנטים")
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		return &object.Boolean{Value: st.running}
	}}
	m.Attrs["מספר_מחוברים"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("שקע_שרת.מספר_מחוברים מצפה ל־0 ארגומנטים")
		}
		st.mu.Lock()
		n := len(st.clients)
		st.mu.Unlock()
		return &object.Number{Value: float64(n)}
	}}
	return m
}

// רשת.שקע_התחבר(כתובת) — לקוח WebSocket
func netWSConnect(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("רשת.שקע_התחבר מצפה לכתובת אחת")
	}
	url, ok := asString(args[0])
	if !ok || url == "" {
		return errObj("רשת.שקע_התחבר מצפה למחרוזת")
	}
	if strings.HasPrefix(url, "http://") {
		url = "ws://" + strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = "wss://" + strings.TrimPrefix(url, "https://")
	} else if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		url = "ws://" + url
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return errObj("רשת.שקע_התחבר נכשל: " + err.Error())
	}
	st := &wsConnState{conn: conn}
	go wsReadLoop(st)
	return wsWrapConn(st)
}

func wsWrapConn(st *wsConnState) object.Object {
	m := &object.Module{Name: "שקע", Attrs: map[string]object.Object{}}
	m.Attrs["שלח"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsConnSend(st, a...)
	}}
	m.Attrs["קרא"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return errObj("שקע.קרא: השתמשו ב־בהודעה לאירועים אסינכרוניים")
	}}
	m.Attrs["סגור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return wsConnClose(st, a...)
	}}
	m.Attrs["בהודעה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("שקע.בהודעה מצפה לפונקציה")
		}
		st.mu.Lock()
		st.onMsg = a[0]
		st.mu.Unlock()
		return object.Nil
	}}
	m.Attrs["בסגירה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 || !isCallable(a[0]) {
			return errObj("שקע.בסגירה מצפה לפונקציה")
		}
		st.mu.Lock()
		st.onClose = a[0]
		st.mu.Unlock()
		return object.Nil
	}}
	m.Attrs["פתוח"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("שקע.פתוח מצפה ל־0 ארגומנטים")
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		return &object.Boolean{Value: !st.closed && st.conn != nil}
	}}
	return m
}

func wsConnSend(st *wsConnState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שקע.שלח מצפה להודעה אחת")
	}
	text, ok := asString(args[0])
	if !ok {
		text = args[0].Inspect()
	}
	st.mu.Lock()
	conn := st.conn
	closed := st.closed
	st.mu.Unlock()
	if closed || conn == nil {
		return errObj("שקע.שלח: השקע סגור")
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
		return errObj("שקע.שלח נכשל: " + err.Error())
	}
	return object.Nil
}

func wsConnClose(st *wsConnState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שקע.סגור מצפה ל־0 ארגומנטים")
	}
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return object.Nil
	}
	st.closed = true
	conn := st.conn
	st.conn = nil
	srv := st.server
	st.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	if srv != nil {
		srv.mu.Lock()
		delete(srv.clients, st)
		srv.mu.Unlock()
	}
	return object.Nil
}

func wsReadLoop(st *wsConnState) {
	defer func() {
		st.mu.Lock()
		wasClosed := st.closed
		st.closed = true
		onClose := st.onClose
		conn := st.conn
		st.conn = nil
		srv := st.server
		st.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		if srv != nil {
			srv.mu.Lock()
			delete(srv.clients, st)
			srv.mu.Unlock()
		}
		if !wasClosed && onClose != nil {
			runOnUI(func() { invokeYod(onClose, nil) })
		}
	}()
	for {
		st.mu.Lock()
		conn := st.conn
		st.mu.Unlock()
		if conn == nil {
			return
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		msg := string(data)
		st.mu.Lock()
		onMsg := st.onMsg
		srv := st.server
		st.mu.Unlock()
		sockObj := wsWrapConn(st)
		if onMsg != nil {
			runOnUI(func() {
				invokeYod(onMsg, []object.Object{&object.String{Value: msg}})
			})
		}
		if srv != nil {
			srv.mu.Lock()
			srvOnMsg := srv.onMsg
			srv.mu.Unlock()
			if srvOnMsg != nil {
				runOnUI(func() {
					invokeYod(srvOnMsg, []object.Object{sockObj, &object.String{Value: msg}})
				})
			}
		}
	}
}

func wsServerBroadcast(st *wsServerState, args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("שקע_שרת.שדר מצפה להודעה אחת")
	}
	text, ok := asString(args[0])
	if !ok {
		text = args[0].Inspect()
	}
	st.mu.Lock()
	list := make([]*wsConnState, 0, len(st.clients))
	for c := range st.clients {
		list = append(list, c)
	}
	st.mu.Unlock()
	var lastErr error
	for _, c := range list {
		c.mu.Lock()
		conn := c.conn
		closed := c.closed
		c.mu.Unlock()
		if closed || conn == nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return errObj("שקע_שרת.שדר: חלק מהשליחות נכשלו: " + lastErr.Error())
	}
	return object.Nil
}

func wsServerStart(st *wsServerState, background bool, args ...object.Object) object.Object {
	if len(args) != 0 {
		name := "שקע_שרת.הפעל"
		if background {
			name = "שקע_שרת.הפעל_ברקע"
		}
		return errObj(name + " מצפה ל־0 ארגומנטים")
	}
	st.mu.Lock()
	if st.running {
		st.mu.Unlock()
		return errObj("שקע_שרת כבר פועל")
	}
	mux := http.NewServeMux()
	mux.HandleFunc(st.path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		cs := &wsConnState{conn: conn, server: st}
		st.mu.Lock()
		st.clients[cs] = true
		onConnect := st.onConnect
		st.mu.Unlock()
		sock := wsWrapConn(cs)
		if onConnect != nil {
			runOnUI(func() {
				invokeYod(onConnect, []object.Object{sock})
			})
		}
		go wsReadLoop(cs)
	})
	st.srv = &http.Server{Addr: st.addr, Handler: mux}
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
		return errObj("שקע_שרת נעצר עם שגיאה: " + err.Error())
	}
	return object.Nil
}

func wsServerStop(st *wsServerState, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("שקע_שרת.עצור מצפה ל־0 ארגומנטים")
	}
	st.mu.Lock()
	srv := st.srv
	clients := make([]*wsConnState, 0, len(st.clients))
	for c := range st.clients {
		clients = append(clients, c)
	}
	st.mu.Unlock()
	for _, c := range clients {
		_ = wsConnClose(c)
	}
	if srv != nil {
		_ = srv.Close()
	}
	return object.Nil
}
