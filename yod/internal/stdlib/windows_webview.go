//go:build windows

package stdlib

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/walk"

	"yod/internal/object"
)

// מניעת כפילות כשכמה גשרים (localhost / postMessage / yod.bridge) מצליחים יחד
var (
	browserBridgeMu   sync.Mutex
	browserBridgeLast string
	browserBridgeAt   time.Time
)

func webviewAppDataPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	p := filepath.Join(base, "Yod", "app-webview")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func normalizeNavURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "file:") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "about:") {
		return s
	}
	// נתיב Windows מקומי (מוחלט / UNC / יחסי שקיים על הדיסק)
	if len(s) >= 2 && s[1] == ':' {
		return localFileURL(s)
	}
	if strings.HasPrefix(s, `\\`) {
		return localFileURL(s)
	}
	if _, err := os.Stat(s); err == nil {
		return localFileURL(s)
	}
	if strings.Contains(s, `\`) || strings.Contains(s, "/") {
		// נתיב שנראה מקומי גם אם הקובץ עדיין לא נוצר
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") ||
			strings.HasSuffix(lower, ".svg") || strings.HasSuffix(lower, ".pdf") {
			return localFileURL(s)
		}
	}
	// קובץ HTML/מקומי בתיקייה הנוכחית בלי מפריד נתיב
	if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
		return localFileURL(s)
	}
	return "https://" + s
}

func localFileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)
	slash := filepath.ToSlash(abs)
	if len(slash) >= 2 && slash[1] == ':' {
		u := &url.URL{Scheme: "file", Path: "/" + slash}
		return u.String()
	}
	u := &url.URL{Scheme: "file", Path: slash}
	return u.String()
}

func attachWebView2(host walk.Window, initialURL, initialHTML string, winSt *windowState) (*edge.Chromium, error) {
	cr := edge.NewChromium()
	cr.DataPath = webviewAppDataPath()
	cr.SetPermission(edge.CoreWebView2PermissionKindClipboardRead, edge.CoreWebView2PermissionStateAllow)

	hwnd := uintptr(host.Handle())
	if hwnd == 0 {
		return nil, fmt.Errorf("אין חלון מארח לדפדפן")
	}
	if !cr.Embed(hwnd) {
		return nil, fmt.Errorf("WebView2 לא זמין — התקינו את Microsoft Edge WebView2 Runtime")
	}
	if settings, err := cr.GetSettings(); err == nil && settings != nil {
		_ = settings.PutIsWebMessageEnabled(true)
		_ = settings.PutAreDefaultContextMenusEnabled(true)
	}
	_ = cr.SetDefaultBackgroundColor(255, 14, 17, 22)
	cr.Resize()

	hasHTML := strings.TrimSpace(initialHTML) != ""
	deferWin := winSt != nil && winSt.deferShow && hasHTML
	if deferWin {
		cr.Init("window.__יוד_המתן_לחשיפה=true;")
	}

	// לא מנווטים כאן — קודם גשר ההודעות (wireBrowserBridge), אחרת טעינה_מוכנה הולכת לאיבוד

	firstPaintDone := false
	revealWeb := func() {
		cr.Resize()
		_ = cr.Show()
		_ = cr.NotifyParentWindowPositionChanged()
	}

	cr.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		if hasHTML && !firstPaintDone {
			firstPaintDone = true
			revealWeb()
			if deferWin {
				// גיבוי ארוך: החשיפה האמיתית מ־טעינה_מוכנה; כאן רק אם ההודעה אבדה
				go func() {
					time.Sleep(1500 * time.Millisecond)
					revealDeferredMainWindow(winSt)
				}()
			}
			return
		}
		cr.Resize()
		_ = cr.Show()
		_ = cr.NotifyParentWindowPositionChanged()
	}

	host.SizeChanged().Attach(func() {
		cr.Resize()
		_ = cr.NotifyParentWindowPositionChanged()
	})
	return cr, nil
}

// browserStartNavigation — ניווט ראשון אחרי שהגשר מוכן (כדי ש־Init של __יוד_גשר ייכנס למסמך).
func browserStartNavigation(st *controlState) {
	if st == nil || st.browser == nil {
		return
	}
	cr := st.browser
	hasHTML := strings.TrimSpace(st.html) != ""
	deferWin := st.parentWin != nil && st.parentWin.deferShow && hasHTML
	cr.Resize()
	_ = cr.NotifyParentWindowPositionChanged()
	// עם שקיפות: חייבים Show כדי ש־WebView יצייר מתחת ל־alpha=0
	if hasHTML && !deferWin {
		_ = cr.Hide()
	} else {
		_ = cr.Show()
	}
	if hasHTML {
		cr.NavigateToString(st.html)
		return
	}
	_ = cr.Show()
	if u := normalizeNavURL(st.url); u != "" && u != "about:blank" {
		cr.Navigate(u)
	}
}

// wireBrowserBridge מחבר הודעות JS↔יוד על וידג׳ט דפדפן.
func wireBrowserBridge(st *controlState) {
	if st == nil || st.browser == nil {
		return
	}

	deliverBrowserMsg := func(msg string) {
		text := strings.TrimSpace(strings.TrimPrefix(msg, "\ufeff"))
		if text == "" {
			return
		}
		// חשיפת חלון שקוף כשמסך הטעינה מצויר
		if st.parentWin != nil && st.parentWin.deferShow && messageIsSplashReady(text) {
			revealDeferredMainWindow(st.parentWin)
		}
		cb := st.onBrowserMsg
		if cb == nil {
			return
		}
		browserBridgeMu.Lock()
		if text == browserBridgeLast && time.Since(browserBridgeAt) < 400*time.Millisecond {
			browserBridgeMu.Unlock()
			return
		}
		browserBridgeLast = text
		browserBridgeAt = time.Now()
		browserBridgeMu.Unlock()

		go func(payload string, handler object.Object) {
			runOnUI(func() {
				invokeYod(handler, []object.Object{&object.String{Value: payload}})
			})
		}(text, cb)
	}

	// גשר אמין: שרת HTTP מקומי — JS שולח GET/POST, לא תלוי ב־postMessage
	bridgeURL := ""
	if ln, err := net.Listen("tcp", "127.0.0.1:0"); err == nil {
		bridgeURL = fmt.Sprintf("http://127.0.0.1:%d/e", ln.Addr().(*net.TCPAddr).Port)
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/e", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				if r.Method == http.MethodOptions {
					w.WriteHeader(204)
					return
				}
				payload := ""
				if r.Method == http.MethodPost {
					b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
					_ = r.Body.Close()
					payload = string(b)
				}
				if payload == "" {
					payload = r.URL.RawQuery
					if i := strings.Index(payload, "&_="); i >= 0 {
						payload = payload[:i]
					}
					if decoded, err2 := url.QueryUnescape(payload); err2 == nil {
						payload = decoded
					}
				}
				deliverBrowserMsg(payload)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(204)
			})
			_ = http.Serve(ln, mux)
		}()
		inj := fmt.Sprintf(`window.__יוד_גשר=%q;`, bridgeURL)
		st.browser.Init(inj)
	}

	prevNav := st.browser.NavigationCompletedCallback
	st.browser.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {
		if prevNav != nil {
			prevNav(sender, args)
		}
		st.browserReady = true
		if bridgeURL != "" {
			st.browser.Eval(fmt.Sprintf(`window.__יוד_גשר=%q;`, bridgeURL))
		}
		cb := st.onBrowserReady
		if cb != nil {
			runOnUI(func() { invokeYod(cb, nil) })
		}
		// אם מנוע התלת כבר רץ אבל הודעת «מוכן» הראשונה התפספסה — בקש שידור חוזר
		st.browser.Eval(`(function(){try{
  if(window.__יוד_מנוע&&window.__יוד_מנוע.post){
    window.__יוד_מנוע.post({סוג:"אירוע",שם:"מוכן",ערכים:{חוזר:true}});
  }
}catch(e){}})();`)
	}

	st.browser.MessageCallback = func(msg string) {
		// חשיפת splash חייבת לעבוד גם כשיש גשר — הקליפה שולחת לפני מנוע.js
		if messageIsSplashReady(msg) {
			deliverBrowserMsg(msg)
			return
		}
		// כשיש גשר localhost — לא לקבל גם postMessage רגיל (מונע כפילות)
		if bridgeURL != "" {
			return
		}
		deliverBrowserMsg(msg)
	}

	// גשר גיבוי ישן (yod.bridge) — רק אם אין localhost
	if bridgeURL == "" {
		st.browser.AddWebResourceRequestedFilter("https://yod.bridge/*", edge.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
		st.browser.WebResourceRequestedCallback = func(req *edge.ICoreWebView2WebResourceRequest, args *edge.ICoreWebView2WebResourceRequestedEventArgs) {
			if req == nil || args == nil {
				return
			}
			uri, err := req.GetUri()
			if err != nil || !strings.HasPrefix(uri, "https://yod.bridge/") {
				return
			}
			if env := st.browser.Environment(); env != nil {
				if resp, err2 := env.CreateWebResourceResponse(nil, 204, "No Content", "Content-Type: text/plain\r\n"); err2 == nil && resp != nil {
					_ = args.PutResponse(resp)
				}
			}
			u, err := url.Parse(uri)
			if err != nil {
				return
			}
			payload := u.RawQuery
			if payload == "" {
				return
			}
			if i := strings.Index(payload, "&_="); i >= 0 {
				payload = payload[:i]
			}
			if decoded, err2 := url.QueryUnescape(payload); err2 == nil {
				payload = decoded
			}
			deliverBrowserMsg(payload)
		}
	}
}

func browserEval(st *controlState, script string) object.Object {
	if st == nil || st.browser == nil {
		return errObj("דפדפן עדיין לא מוצג — קראו להרץ_js אחרי הצג")
	}
	st.browser.Eval(script)
	return object.Nil
}

// browserSend מעביר מחרוזת/JSON לדף דרך Eval → window.__יוד_קבל / אירוע yod-message.
func browserSend(st *controlState, payload string) object.Object {
	if st == nil || st.browser == nil {
		return errObj("דפדפן עדיין לא מוצג — קראו לשלח אחרי הצג")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return errObj("דפדפן.שלח: לא ניתן לקודד את ההודעה")
	}
	script := `(function(raw){
  var data = raw;
  try { data = JSON.parse(raw); } catch (e) {}
  if (typeof data === "string") {
    try { data = JSON.parse(data); } catch (e) {}
  }
  try {
    if (typeof window.__יוד_קבל === "function") {
      window.__יוד_קבל(data, raw);
    } else {
      window.dispatchEvent(new CustomEvent("yod-message", { detail: { data: data, raw: raw } }));
    }
  } catch (e) {}
})(` + string(raw) + `);`
	st.browser.Eval(script)
	return object.Nil
}

// browserLoadInitial טוען את הכתובת/HTML שנשמרו — נקרא אחרי שהחלון מוצג
func browserLoadInitial(st *controlState) {
	if st == nil || st.browser == nil {
		return
	}
	st.browser.Resize()
	_ = st.browser.NotifyParentWindowPositionChanged()
	hasHTML := strings.TrimSpace(st.html) != ""
	if hasHTML {
		// ניווט ראשון כבר ב־browserStartNavigation — אל תסתיר ותטען שוב (שובר splash שקוף)
		if st.parentWin != nil && st.parentWin.deferShow {
			_ = st.browser.Show()
			return
		}
		if st.browserReady {
			_ = st.browser.Show()
			return
		}
		st.browserReady = false
		_ = st.browser.Hide()
		st.browser.NavigateToString(st.html)
		return
	}
	st.browserReady = false
	_ = st.browser.Show()
	u := normalizeNavURL(st.url)
	if u != "" && u != "about:blank" {
		st.browser.Navigate(u)
	}
}

func browserNavigate(st *controlState, raw string) {
	st.html = ""
	st.url = normalizeNavURL(raw)
	st.browserReady = false
	if st.browser != nil && st.url != "" {
		st.browser.Navigate(st.url)
	}
}

func browserSetHTML(st *controlState, html string) {
	st.html = html
	st.url = "about:blank"
	st.browserReady = false
	if st.browser != nil {
		st.browser.NavigateToString(html)
	}
}

func browserRefresh(st *controlState) {
	if st.browser == nil {
		return
	}
	st.browserReady = false
	if strings.TrimSpace(st.html) != "" {
		st.browser.NavigateToString(st.html)
		return
	}
	if st.url != "" {
		st.browser.Navigate(st.url)
	}
}
