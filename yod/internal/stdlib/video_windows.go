//go:build windows

package stdlib

import (
	"fmt"
	"html"
	"path/filepath"

	"yod/internal/object"
)

func videoCreatePanel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("וידאו.חלונית מצפה לנתיב קובץ אחד")
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("וידאו.חלונית מצפה למחרוזת")
	}
	full := resolveMediaPath(path)
	st := &controlState{
		kind:        "וידאו",
		url:         "about:blank",
		videoPath:   full,
		videoVol:    100,
		videoLoop:   false,
		stretchFactor: 1,
	}
	st.html = buildVideoHTML(st)
	return wrapVideoWidget(st)
}

func wrapVideoWidget(st *controlState) *object.GuiWidget {
	w := &object.GuiWidget{Kind: "וידאו", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["נגן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("וידאו.נגן מצפה ל־0 ארגומנטים")
		}
		videoEval(st, "var v=document.getElementById('v'); if(v){v.play();}")
		return object.Nil
	}}
	w.Attrs["השהה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("וידאו.השהה מצפה ל־0 ארגומנטים")
		}
		videoEval(st, "var v=document.getElementById('v'); if(v){v.pause();}")
		return object.Nil
	}}
	w.Attrs["עצור"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("וידאו.עצור מצפה ל־0 ארגומנטים")
		}
		videoEval(st, "var v=document.getElementById('v'); if(v){v.pause();v.currentTime=0;}")
		return object.Nil
	}}
	w.Attrs["קבע_לולאה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.קבע_לולאה מצפה לבוליאני")
		}
		b, ok := a[0].(*object.Boolean)
		if !ok {
			return errObj("וידאו.קבע_לולאה מצפה לאמת/שקר")
		}
		st.videoLoop = b.Value
		if b.Value {
			videoEval(st, "var v=document.getElementById('v'); if(v){v.loop=true;}")
		} else {
			videoEval(st, "var v=document.getElementById('v'); if(v){v.loop=false;}")
		}
		return object.Nil
	}}
	w.Attrs["עוצמה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.עוצמה מצפה למספר 0–100")
		}
		n, ok := a[0].(*object.Number)
		if !ok {
			return errObj("וידאו.עוצמה מצפה למספר")
		}
		v := int(n.Value)
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		st.videoVol = v
		videoEval(st, fmt.Sprintf("var v=document.getElementById('v'); if(v){v.volume=%g;}", float64(v)/100.0))
		return object.Nil
	}}
	w.Attrs["טען"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.טען מצפה לנתיב")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("וידאו.טען מצפה למחרוזת")
		}
		st.videoPath = resolveMediaPath(path)
		st.html = buildVideoHTML(st)
		if st.browser != nil {
			st.browser.NavigateToString(st.html)
		}
		return object.Nil
	}}
	return w
}

func buildVideoHTML(st *controlState) string {
	src := localFileURL(st.videoPath)
	src = html.EscapeString(src)
	loop := ""
	if st.videoLoop {
		loop = " loop"
	}
	vol := float64(st.videoVol) / 100.0
	name := html.EscapeString(filepath.Base(st.videoPath))
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><style>
html,body{margin:0;height:100%%;background:#111;overflow:hidden}
video{width:100%%;height:100%%;object-fit:contain;background:#000}
</style></head><body>
<video id="v" src="%s" controls%s title="%s"></video>
<script>var v=document.getElementById('v'); if(v){v.volume=%g;}</script>
</body></html>`, src, loop, name, vol)
}

func videoEval(st *controlState, script string) {
	if st == nil || st.browser == nil {
		return
	}
	st.browser.Eval(script)
}

func videoLoadInitial(st *controlState) {
	if st == nil || st.browser == nil {
		return
	}
	st.html = buildVideoHTML(st)
	st.browser.Resize()
	_ = st.browser.Show()
	st.browser.NavigateToString(st.html)
}
