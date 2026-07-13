//go:build windows

package stdlib

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/walk"
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
	// נתיב Windows מקומי
	if len(s) >= 2 && s[1] == ':' {
		return localFileURL(s)
	}
	if strings.HasPrefix(s, `\\`) || strings.Contains(s, `\`) {
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

func attachWebView2(host walk.Window, initialURL, initialHTML string) (*edge.Chromium, error) {
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
	cr.Resize()
	_ = cr.Show()

	load := func() {
		cr.Resize()
		_ = cr.Show()
		_ = cr.NotifyParentWindowPositionChanged()
		if strings.TrimSpace(initialHTML) != "" {
			cr.NavigateToString(initialHTML)
			return
		}
		if u := normalizeNavURL(initialURL); u != "" && u != "about:blank" {
			cr.Navigate(u)
		}
	}
	load()

	// אחרי ניווט — לרענן גבולות (WebView2 לפעמים נשאר ריק עד Resize)
	cr.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
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

// browserLoadInitial טוען את הכתובת/HTML שנשמרו — נקרא אחרי שהחלון מוצג
func browserLoadInitial(st *controlState) {
	if st == nil || st.browser == nil {
		return
	}
	st.browser.Resize()
	_ = st.browser.Show()
	_ = st.browser.NotifyParentWindowPositionChanged()
	if strings.TrimSpace(st.html) != "" {
		st.browser.NavigateToString(st.html)
		return
	}
	u := normalizeNavURL(st.url)
	if u != "" && u != "about:blank" {
		st.browser.Navigate(u)
	}
}

func browserNavigate(st *controlState, raw string) {
	st.html = ""
	st.url = normalizeNavURL(raw)
	if st.browser != nil && st.url != "" {
		st.browser.Navigate(st.url)
	}
}

func browserSetHTML(st *controlState, html string) {
	st.html = html
	st.url = "about:blank"
	if st.browser != nil {
		st.browser.NavigateToString(html)
	}
}

func browserRefresh(st *controlState) {
	if st.browser == nil {
		return
	}
	if strings.TrimSpace(st.html) != "" {
		st.browser.NavigateToString(st.html)
		return
	}
	if st.url != "" {
		st.browser.Navigate(st.url)
	}
}
