package stdlib

import (
	"io"
	"net/http"
	"strings"
	"time"

	"yod/internal/object"
)

var (
	httpClient     = &http.Client{Timeout: 30 * time.Second}
	httpTimeoutSec = 30.0
)

func NewNetModule() *object.Module {
	m := &object.Module{Name: "רשת", Attrs: map[string]object.Object{}}
	m.Attrs["גש"] = &object.Builtin{Fn: netGet}
	m.Attrs["פרסם"] = &object.Builtin{Fn: netPost}
	m.Attrs["פרסם_json"] = &object.Builtin{Fn: netPostJSON}
	m.Attrs["בקשה"] = &object.Builtin{Fn: netRequest}
	registerHTTPAsync(m)
	m.Attrs["קבע_זמן_קצוב"] = &object.Builtin{Fn: netSetTimeout}
	m.Attrs["שרת"] = &object.Builtin{Fn: netNewServer}
	m.Attrs["שקע_שרת"] = &object.Builtin{Fn: netWSServer}
	m.Attrs["שקע_התחבר"] = &object.Builtin{Fn: netWSConnect}
	m.Attrs["FTP_התחבר"] = &object.Builtin{Fn: netFTPConnect}
	registerFTPConnectAsync(m)
	m.Attrs["SFTP_התחבר"] = &object.Builtin{Fn: netSFTPConnect}
	m.Attrs["חשב_מהירות"] = &object.Builtin{Fn: netCalcSpeed}
	return m
}

// רשת.קבע_זמן_קצוב(שניות) — ברירת מחדל 30
func netSetTimeout(args ...object.Object) object.Object {
	if err := expectArgs("רשת.קבע_זמן_קצוב", 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("קבע_זמן_קצוב מצפה למספר שניות")
	}
	sec := n.Value
	if sec < 0 {
		sec = 0
	}
	if sec > 600 {
		sec = 600
	}
	httpTimeoutSec = sec
	if sec <= 0 {
		httpClient.Timeout = 0
	} else {
		httpClient.Timeout = time.Duration(sec * float64(time.Second))
	}
	return object.Nil
}

// רשת.פרסם_json(כתובת, גוף) — POST עם Content-Type: application/json
func netPostJSON(args ...object.Object) object.Object {
	if err := expectArgs("רשת.פרסם_json", 2, args); err != nil {
		return err
	}
	url, ok := asString(args[0])
	if !ok {
		return errObj("רשת.פרסם_json: כתובת חייבת להיות מחרוזת")
	}
	body := bodyString(args[1])
	return doHTTP("POST", url, body, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	})
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

	// שחרור מנעול הריצה בזמן ה-I/O החוסם — כך אם הקריאה עטופה ב־`משימה`, ה־UI ממשיך
	// להגיב. הקטע הזה הוא Go טהור ואינו נוגע במצב יוד, לכן בטוח לשחרר.
	var resp *http.Response
	var data []byte
	object.WithoutYodLock(func() {
		resp, err = httpClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // עד 8MB
	})
	if err != nil {
		if resp == nil {
			return errObj("בקשת רשת נכשלה: " + err.Error())
		}
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
