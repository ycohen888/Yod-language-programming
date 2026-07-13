package stdlib

import (
	"io"
	"net/http"
	"strings"
	"time"

	"yod/internal/object"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func NewNetModule() *object.Module {
	m := &object.Module{Name: "רשת", Attrs: map[string]object.Object{}}
	m.Attrs["גש"] = &object.Builtin{Fn: netGet}
	m.Attrs["פרסם"] = &object.Builtin{Fn: netPost}
	m.Attrs["בקשה"] = &object.Builtin{Fn: netRequest}
	m.Attrs["שרת"] = &object.Builtin{Fn: netNewServer}
	return m
}

// רשת.גש(כתובת) — GET
func netGet(args ...object.Object) object.Object {
	if err := expectArgs("רשת.גש", 1, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.גש מצפה לכתובת מחרוזת")
	}
	return doHTTP("GET", url, "", nil)
}

// רשת.פרסם(כתובת, גוף) — POST
func netPost(args ...object.Object) object.Object {
	if err := expectArgs("רשת.פרסם", 2, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.פרסם: כתובת חייבת להיות מחרוזת")
	}
	body := bodyString(args[1])
	return doHTTP("POST", url, body, map[string]string{
		"Content-Type": "text/plain; charset=utf-8",
	})
}

// רשת.בקשה(שיטה, כתובת [, גוף [, כותרות]])
func netRequest(args ...object.Object) object.Object {
	if len(args) < 2 || len(args) > 4 {
		return errObj("רשת.בקשה מצפה ל־2 עד 4 ארגומנטים: שיטה, כתובת [, גוף [, כותרות]]")
	}
	method, ok := asString(args[0])
	if !ok {
		return errObj("רשת.בקשה: שיטה חייבת להיות מחרוזת (GET, POST וכו')")
	}
	url, ok := asString(args[1])
	if !ok {
		return errObj("רשת.בקשה: כתובת חייבת להיות מחרוזת")
	}
	body := ""
	if len(args) >= 3 {
		body = bodyString(args[2])
	}
	var headers map[string]string
	if len(args) >= 4 {
		h, err := hashToStringMap(args[3])
		if err != nil {
			return err
		}
		headers = h
	}
	return doHTTP(strings.ToUpper(method), url, body, headers)
}

func bodyString(obj object.Object) string {
	if s, ok := obj.(*object.String); ok {
		return s.Value
	}
	if _, ok := obj.(*object.Null); ok {
		return ""
	}
	return obj.Inspect()
}

func hashToStringMap(obj object.Object) (map[string]string, *object.Error) {
	h, ok := obj.(*object.Hash)
	if !ok {
		return nil, errObj("כותרות חייבות להיות מילון")
	}
	out := map[string]string{}
	for k, v := range h.Pairs {
		if s, ok := v.(*object.String); ok {
			out[k] = s.Value
		} else {
			out[k] = v.Inspect()
		}
	}
	return out, nil
}

func doHTTP(method, url, body string, headers map[string]string) object.Object {
	var reader io.Reader
	if body != "" || method == "POST" || method == "PUT" || method == "PATCH" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return errObj("בקשת רשת לא תקינה: " + err.Error())
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if reader != nil && req.Header.Get("Content-Type") == "" && body != "" {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errObj("בקשת רשת נכשלה: " + err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // עד 8MB
	if err != nil {
		return errObj("קריאת תשובה נכשלה: " + err.Error())
	}

	headerHash := &object.Hash{Pairs: map[string]object.Object{}}
	for k, vals := range resp.Header {
		headerHash.Pairs[k] = &object.String{Value: strings.Join(vals, ", ")}
	}

	return &object.Hash{Pairs: map[string]object.Object{
		"קוד":     &object.Number{Value: float64(resp.StatusCode)},
		"גוף":     &object.String{Value: string(data)},
		"כותרות": headerHash,
	}}
}
