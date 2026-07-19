package stdlib

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yod/internal/object"
)

func TestNetGetAndPostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			w.WriteHeader(200)
			_, _ = w.Write([]byte("pong"))
		case "/echo":
			if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
				http.Error(w, "bad content type", 400)
				return
			}
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			w.Header().Set("X-Echo", "1")
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := httpClient.Timeout
	httpClient.Timeout = 0
	defer func() { httpClient.Timeout = old }()

	res := netGet(&object.String{Value: srv.URL + "/ping"})
	h, ok := res.(*object.Hash)
	if !ok {
		t.Fatalf("GET: expected hash, got %T", res)
	}
	if c, _ := h.Pairs["קוד"].(*object.Number); c == nil || int(c.Value) != 200 {
		t.Fatalf("GET status: %+v", h.Pairs["קוד"])
	}
	if g, _ := h.Pairs["גוף"].(*object.String); g == nil || g.Value != "pong" {
		t.Fatalf("GET body: %+v", h.Pairs["גוף"])
	}

	res = netPostJSON(&object.String{Value: srv.URL + "/echo"}, &object.String{Value: `{"ok":true}`})
	h, ok = res.(*object.Hash)
	if !ok {
		t.Fatalf("POST JSON: expected hash, got %T", res)
	}
	if g, _ := h.Pairs["גוף"].(*object.String); g == nil || g.Value != `{"ok":true}` {
		t.Fatalf("POST body: %+v", h.Pairs["גוף"])
	}
}

func TestNetSetTimeout(t *testing.T) {
	res := netSetTimeout(&object.Number{Value: 5})
	if res != object.Nil {
		t.Fatalf("set timeout: %v", res)
	}
	if httpTimeoutSec != 5 {
		t.Fatalf("timeout sec = %v", httpTimeoutSec)
	}
	netSetTimeout(&object.Number{Value: 30})
}
