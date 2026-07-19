package stdlib

import (
	"strings"
	"testing"

	"yod/internal/object"
)

func TestFTPConnectRequiresAddress(t *testing.T) {
	res := netFTPConnect(&object.Hash{Pairs: map[string]object.Object{}})
	err, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", res)
	}
	if !strings.Contains(err.Message, "כתובת") {
		t.Fatalf("message: %q", err.Message)
	}
}

func TestFTPConnectRejectsBadPort(t *testing.T) {
	res := netFTPConnect(&object.Hash{Pairs: map[string]object.Object{
		"כתובת": &object.String{Value: "127.0.0.1"},
		"פורט":  &object.Number{Value: 0},
	}})
	err, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", res)
	}
	if !strings.Contains(err.Message, "פורט") {
		t.Fatalf("message: %q", err.Message)
	}
}

func TestFTPConnectExpectsHash(t *testing.T) {
	res := netFTPConnect(&object.String{Value: "x"})
	if _, ok := res.(*object.Error); !ok {
		t.Fatalf("expected error, got %T", res)
	}
}

func TestSFTPConnectRequiresAddress(t *testing.T) {
	res := netSFTPConnect(&object.Hash{Pairs: map[string]object.Object{}})
	err, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", res)
	}
	if !strings.Contains(err.Message, "כתובת") {
		t.Fatalf("message: %q", err.Message)
	}
}

func TestSFTPConnectRequiresUser(t *testing.T) {
	res := netSFTPConnect(&object.Hash{Pairs: map[string]object.Object{
		"כתובת": &object.String{Value: "127.0.0.1"},
	}})
	err, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", res)
	}
	if !strings.Contains(err.Message, "משתמש") {
		t.Fatalf("message: %q", err.Message)
	}
}

func TestSFTPConnectRequiresAuth(t *testing.T) {
	res := netSFTPConnect(&object.Hash{Pairs: map[string]object.Object{
		"כתובת":  &object.String{Value: "127.0.0.1"},
		"משתמש": &object.String{Value: "u"},
	}})
	err, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", res)
	}
	if !strings.Contains(err.Message, "סיסמה") && !strings.Contains(err.Message, "מפתח") {
		t.Fatalf("message: %q", err.Message)
	}
}

func TestRemoteFTPPath(t *testing.T) {
	if got := remoteFTPPath(`a\b`); got != "/a/b" {
		t.Fatalf("got %q", got)
	}
	if got := remoteFTPPath("/x/"); got != "/x" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeFTPAddress(t *testing.T) {
	host, port, secure := normalizeFTPAddress("FTP ://Backup.servxx.com", 21, false)
	if host != "Backup.servxx.com" || port != 21 || secure {
		t.Fatalf("got %q %d %v", host, port, secure)
	}
	host, port, secure = normalizeFTPAddress("ftps://h.example.com:990/path", 21, false)
	if host != "h.example.com" || port != 990 || !secure {
		t.Fatalf("got %q %d %v", host, port, secure)
	}
	host, _, _ = normalizeFTPAddress("ftp://user:pass@host.com/dir", 21, false)
	if host != "host.com" {
		t.Fatalf("got %q", host)
	}
}
