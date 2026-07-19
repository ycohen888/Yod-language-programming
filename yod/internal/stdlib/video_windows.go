//go:build windows

package stdlib

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"yod/internal/object"
)

var (
	videoHTTPOnce sync.Once
	videoHTTPBase string
	videoAllowMu  sync.Mutex
	videoAllow    = map[string]struct{}{}
)

func ensureVideoHTTPServer() string {
	videoHTTPOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return
		}
		videoHTTPBase = fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
		mux := http.NewServeMux()
		mux.HandleFunc("/media", func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get("p")
			if raw == "" {
				http.Error(w, "חסר נתיב", http.StatusBadRequest)
				return
			}
			path, err := url.QueryUnescape(raw)
			if err != nil || strings.TrimSpace(path) == "" {
				http.Error(w, "נתיב לא תקין", http.StatusBadRequest)
				return
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				http.Error(w, "נתיב לא תקין", http.StatusBadRequest)
				return
			}
			abs = filepath.Clean(abs)
			key := strings.ToLower(abs)
			videoAllowMu.Lock()
			_, ok := videoAllow[key]
			videoAllowMu.Unlock()
			if !ok {
				http.Error(w, "אין הרשאה", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, abs)
		})
		go func() {
			_ = http.Serve(ln, mux)
		}()
	})
	return videoHTTPBase
}

// videoHTTPURL — כתובת http מקומית לנגן (file:// נחסם מ־NavigateToString ב־WebView2).
func videoHTTPURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.Clean(abs)
	base := ensureVideoHTTPServer()
	if base == "" {
		return localFileURL(abs)
	}
	videoAllowMu.Lock()
	videoAllow[strings.ToLower(abs)] = struct{}{}
	videoAllowMu.Unlock()
	return base + "/media?p=" + url.QueryEscape(abs)
}

func videoCreatePanel(args ...object.Object) object.Object {
	if len(args) > 1 {
		return errObj("וידאו.חלונית מצפה ל־0 או 1 ארגומנטים (נתיב אופציונלי)")
	}
	full := ""
	if len(args) == 1 {
		if args[0] != nil && args[0].Type() != object.NullObj {
			path, ok := asString(args[0])
			if !ok {
				return errObj("וידאו.חלונית מצפה למחרוזת נתיב")
			}
			path = strings.TrimSpace(path)
			if path != "" {
				full = resolveMediaPath(path)
			}
		}
	}
	st := &controlState{
		kind:              "וידאו",
		url:               "about:blank",
		videoPath:         full,
		videoVol:          100,
		videoLoop:         false,
		videoShowOverlays: true,
		videoSelected:     -1,
		stretchFactor:     2,
	}
	st.html = buildVideoHTML(st, false)
	return wrapVideoWidget(st)
}

func wrapVideoWidget(st *controlState) *object.GuiWidget {
	w := &object.GuiWidget{Kind: "וידאו", Data: st, Attrs: map[string]object.Object{}}
	w.Attrs["נגן"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 0 {
			return errObj("וידאו.נגן מצפה ל־0 ארגומנטים")
		}
		videoEval(st, `var v=document.getElementById('v'); if(v){ var p=v.play(); if(p&&p.catch)p.catch(function(){}); }`)
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
		autoplay := false
		if len(a) < 1 || len(a) > 2 {
			return errObj("וידאו.טען מצפה לנתיב [, לנגן_אוטומטית]")
		}
		path, ok := asString(a[0])
		if !ok {
			return errObj("וידאו.טען מצפה למחרוזת")
		}
		if len(a) == 2 {
			if b, okb := a[1].(*object.Boolean); okb {
				autoplay = b.Value
			} else {
				return errObj("וידאו.טען: ארגומנט שני חייב להיות אמת/שקר")
			}
		}
		path = strings.TrimSpace(path)
		if path == "" {
			st.videoPath = ""
		} else {
			st.videoPath = resolveMediaPath(path)
		}
		st.html = buildVideoHTML(st, autoplay)
		if st.browser != nil {
			st.browser.NavigateToString(st.html)
		}
		return object.Nil
	}}
	w.Attrs["קבע_מתיחה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		return setStretchFactor(st, "וידאו.קבע_מתיחה", a...)
	}}
	w.Attrs["קבע_שכבות_טקסט"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.קבע_שכבות_טקסט מצפה לרשימה אחת")
		}
		if a[0] == nil || a[0].Type() == object.NullObj {
			st.videoOverlays = nil
		} else {
			arr, ok := a[0].(*object.Array)
			if !ok {
				return errObj("וידאו.קבע_שכבות_טקסט: נדרשת רשימה של מילונים")
			}
			if len(arr.Elements) == 0 {
				st.videoOverlays = nil
			} else {
				overlays, err := parseTextOverlays(arr)
				if err != nil {
					return errObj("וידאו.קבע_שכבות_טקסט: " + err.Error())
				}
				st.videoOverlays = overlays
			}
		}
		st.html = buildVideoHTML(st, false)
		if st.browser != nil {
			st.browser.NavigateToString(st.html)
		}
		return &object.Null{}
	}}
	w.Attrs["הצג_שכבות"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.הצג_שכבות מצפה לבוליאני אחד")
		}
		b, ok := a[0].(*object.Boolean)
		if !ok {
			return errObj("וידאו.הצג_שכבות מצפה ל־אמת/שקר")
		}
		st.videoShowOverlays = b.Value
		st.html = buildVideoHTML(st, false)
		if st.browser != nil {
			st.browser.NavigateToString(st.html)
		}
		return &object.Null{}
	}}
	w.Attrs["בהודעה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.בהודעה מצפה לפונקציה אחת")
		}
		if !isCallable(a[0]) {
			return errObj("וידאו.בהודעה מצפה לפונקציה")
		}
		st.onBrowserMsg = a[0]
		return &object.Null{}
	}}
	w.Attrs["בחר_שכבה"] = &object.Builtin{Fn: func(a ...object.Object) object.Object {
		if len(a) != 1 {
			return errObj("וידאו.בחר_שכבה מצפה למספר אינדקס (-1 = ללא)")
		}
		n, ok := a[0].(*object.Number)
		if !ok {
			return errObj("וידאו.בחר_שכבה מצפה למספר")
		}
		st.videoSelected = int(n.Value)
		videoEval(st, fmt.Sprintf("if(window.__yodSelectOverlay){window.__yodSelectOverlay(%d);}", st.videoSelected))
		return &object.Null{}
	}}
	return w
}

func buildVideoHTML(st *controlState, autoplay bool) string {
	ovHTML, ovJS := buildVideoOverlayLayer(st)
	hasVideo := st != nil && strings.TrimSpace(st.videoPath) != ""

	// אלמנט המשטח: וידאו אמיתי או משטח־עיצוב ריק (כדי שהטקסטים יוצגו גם בלי וידאו)
	surfaceHTML := ""
	controlsHTML := ""
	barText := "עורך טקסטים · אין וידאו מקושר"
	videoScript := ""
	if hasVideo {
		src := html.EscapeString(videoHTTPURL(st.videoPath))
		loop := ""
		if st.videoLoop {
			loop = " loop"
		}
		ap := ""
		if autoplay {
			ap = " autoplay"
		}
		barText = html.EscapeString(filepath.Base(st.videoPath))
		// בלי בקרות מובנות (controls) — משתמשים בסרגל מותאם ברוחב מלא בתחתית
		surfaceHTML = fmt.Sprintf(`<video id="v" src="%s" preload="auto"%s%s title="%s"></video>`,
			src, loop, ap, barText)
		controlsHTML = `<div class="pbar" id="pbar">
<button id="pp" class="pbtn" title="נגן / השהה" aria-label="נגן / השהה"></button>
<input id="seek" class="seek" type="range" min="0" max="1000" value="0" step="1" aria-label="מיקום">
<span id="tlabel" class="tlabel">0:00 / 0:00</span>
<button id="mute" class="mbtn" title="השתקה" aria-label="השתקה">קול</button>
<input id="vol" class="vol" type="range" min="0" max="100" value="100" step="1" aria-label="עוצמה">
</div>`
		playJS := ""
		if autoplay {
			playJS = `function tryPlay(){var p=v.play(); if(p&&p.catch)p.catch(function(){});}
v.addEventListener('canplay', tryPlay);
v.addEventListener('loadeddata', tryPlay);
tryPlay();`
		}
		vol := float64(st.videoVol) / 100.0
		videoScript = fmt.Sprintf(`var v=document.getElementById('v');
var err=document.getElementById('err');
if(v){
  v.volume=%g;
  v.addEventListener('error', function(){
    if(!err) return;
    err.hidden=false;
    err.textContent='לא ניתן לנגן את הקובץ (פורמט/נתיב). נסו לפתוח במנגן חיצוני.';
  });
  (function(){
    var pp=document.getElementById('pp'),seek=document.getElementById('seek'),
        tl=document.getElementById('tlabel'),mute=document.getElementById('mute'),
        volc=document.getElementById('vol');
    function fmt(t){t=Math.max(0,Math.floor(t||0));var m=Math.floor(t/60),s=t%%60;return m+':'+(s<10?'0':'')+s;}
    function upPP(){if(pp)pp.classList.toggle('playing',!v.paused);}
    function upTime(){
      if(seek&&v.duration&&isFinite(v.duration)){seek.value=String(Math.round(v.currentTime/v.duration*1000));}
      if(tl)tl.textContent=fmt(v.currentTime)+' / '+fmt(v.duration);
    }
    function upMute(){if(mute)mute.classList.toggle('muted',v.muted||v.volume===0);}
    if(pp)pp.addEventListener('click',function(){if(v.paused)v.play();else v.pause();});
    v.addEventListener('play',upPP);v.addEventListener('pause',upPP);
    v.addEventListener('timeupdate',upTime);
    v.addEventListener('loadedmetadata',upTime);
    v.addEventListener('durationchange',upTime);
    v.addEventListener('volumechange',function(){upMute();if(volc)volc.value=String(Math.round((v.muted?0:v.volume)*100));});
    if(seek)seek.addEventListener('input',function(){if(v.duration&&isFinite(v.duration))v.currentTime=seek.value/1000*v.duration;});
    if(mute)mute.addEventListener('click',function(){v.muted=!v.muted;});
    if(volc){volc.value=String(Math.round(v.volume*100));volc.addEventListener('input',function(){v.volume=volc.value/100;v.muted=(volc.value==0);});}
    v.addEventListener('click',function(e){if(e.target===v){if(v.paused)v.play();else v.pause();}});
    upPP();upTime();upMute();
  })();
  %s
}`, vol, playJS)
	} else {
		surfaceHTML = `<div id="novideo" class="novideo"><div class="nv-hint">אין וידאו מקושר<br><span>הטקסטים נשמרים בפרוייקט · הקליטו או קשרו וידאו קיים</span></div></div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="he" dir="rtl"><head><meta charset="utf-8"><style>
html,body{margin:0;height:100%%;background:#0b0f14;overflow:hidden}
.stage{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;background:#000}
.bar{position:absolute;top:0;left:0;right:0;z-index:4;padding:8px 12px;
background:linear-gradient(180deg,rgba(0,0,0,.65),transparent);color:#e8eef7;
font:600 12px Assistant,Segoe UI,sans-serif;pointer-events:none;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
video{max-width:100%%;max-height:100%%;width:auto;height:auto;object-fit:contain;background:#000;display:block}
.novideo{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;
background:repeating-linear-gradient(45deg,#0e131a,#0e131a 12px,#111823 12px,#111823 24px)}
.nv-hint{color:#8ea1b8;font:700 16px Assistant,Segoe UI,sans-serif;text-align:center;line-height:1.7;pointer-events:none}
.nv-hint span{font-weight:500;font-size:12px;color:#5f7288}
.ov-layer{position:absolute;z-index:3;pointer-events:none;overflow:hidden}
.ov{position:absolute;white-space:pre-wrap;line-height:1.25;font-family:Arial,'Segoe UI',Tahoma,sans-serif;
box-sizing:border-box;padding:.12em .38em;border-radius:4px;pointer-events:auto;cursor:pointer;
max-width:96%%;user-select:none;-webkit-user-select:none;touch-action:none}
.ov.sel{outline:2px solid #3dd6c6;outline-offset:2px;cursor:move;box-shadow:0 0 0 3px rgba(61,214,198,.22)}
.ov .ovhandle{position:absolute;width:10px;height:10px;border-radius:50%%;background:#3dd6c6;
box-shadow:0 0 0 2px #0b0f14;top:-6px;inset-inline-start:-6px;display:none}
.ov.sel .ovhandle{display:block}
.dhint{position:absolute;z-index:8;left:50%%;bottom:76px;transform:translateX(-50%%);
background:rgba(11,15,20,.82);color:#e8eef7;font:600 12px Assistant,Segoe UI,sans-serif;
padding:5px 12px;border-radius:999px;pointer-events:none;opacity:0;transition:opacity .15s}
.dhint.on{opacity:1}
.err{position:absolute;left:12px;right:12px;bottom:60px;z-index:5;color:#f87171;font:13px Assistant,Segoe UI,sans-serif;text-align:center}
.pbar{position:absolute;left:0;right:0;bottom:0;z-index:7;display:flex;align-items:center;gap:12px;
box-sizing:border-box;padding:12px 18px;
background:linear-gradient(0deg,rgba(0,0,0,.82),rgba(0,0,0,.35) 60%%,transparent)}
.pbtn{flex:0 0 auto;width:38px;height:38px;border-radius:50%%;border:0;cursor:pointer;
background:#3dd6c6;position:relative;padding:0}
.pbtn:hover{filter:brightness(1.08)}
.pbtn::before{content:'';position:absolute;top:50%%;left:50%%;transform:translate(-40%%,-50%%);
width:0;height:0;border-style:solid;border-width:8px 0 8px 13px;
border-color:transparent transparent transparent #04211d}
.pbtn.playing::before{border:0;width:5px;height:14px;transform:translate(-90%%,-50%%);
background:#04211d;box-shadow:8px 0 0 #04211d}
.seek{flex:1 1 auto;height:6px;accent-color:#3dd6c6;cursor:pointer;min-width:60px}
.tlabel{flex:0 0 auto;color:#e8eef7;font:600 12px ui-monospace,Consolas,monospace;
min-width:96px;text-align:center;white-space:nowrap}
.mbtn{flex:0 0 auto;border:0;cursor:pointer;background:rgba(255,255,255,.12);color:#e8eef7;
border-radius:8px;padding:7px 12px;font:600 12px Assistant,Segoe UI,sans-serif}
.mbtn:hover{background:rgba(255,255,255,.2)}
.mbtn.muted{background:rgba(248,113,113,.25);color:#fca5a5}
.vol{flex:0 0 96px;height:6px;accent-color:#3dd6c6;cursor:pointer}
</style></head><body>
<div class="bar">%s</div>
<div class="stage">
%s
%s
<div id="dhint" class="dhint"></div>
%s
</div>
<div id="err" class="err" hidden></div>
<script>%s</script>
<script>
%s
%s
</script>
</body></html>`, barText, surfaceHTML, ovHTML, controlsHTML, videoOverlayEngineJS, videoScript, ovJS)
}

type ovJSON struct {
	Text     string  `json:"text"`
	Size     float64 `json:"size"`
	Color    string  `json:"color"`
	Box      bool    `json:"box"`
	BoxColor string  `json:"boxColor"`
	BoxOp    float64 `json:"boxOp"`
	Shadow   bool    `json:"shadow"`
	Pos      string  `json:"pos"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
}

// buildVideoOverlayLayer בונה שכבת עורך טקסטים מקצועית: מיקום WYSIWYG (video-px),
// סימון, גרירה, לחיצה כפולה, ותצוגה תלוית־זמן. הנתונים מוזרקים כ־JSON וה־JS מרנדר.
func buildVideoOverlayLayer(st *controlState) (htmlLayer string, js string) {
	htmlLayer = `<div class="ov-layer" id="ov"></div>`
	if st == nil || !st.videoShowOverlays {
		return htmlLayer, `if(window.__yodOVInit){window.__yodOVInit([], -1);}`
	}
	data := make([]ovJSON, 0, len(st.videoOverlays))
	for _, o := range st.videoOverlays {
		size := o.Size
		if size <= 0 {
			size = 36
		}
		col := o.Color
		if col == "" {
			col = "#ffffff"
		}
		bc := o.BoxColor
		if bc == "" {
			bc = "#000000"
		}
		bop := o.BoxOpacity
		if bop <= 0 {
			bop = 0.45
		}
		pos := o.Position
		if pos == "" {
			pos = "למטה_מרכז"
		}
		data = append(data, ovJSON{
			Text: o.Text, Size: size, Color: col, Box: o.Box, BoxColor: bc, BoxOp: bop,
			Shadow: o.Shadow, Pos: pos, X: o.X, Y: o.Y, Start: o.Start, End: o.End,
		})
	}
	raw, err := json.Marshal(data)
	if err != nil {
		raw = []byte("[]")
	}
	sel := st.videoSelected
	js = "if(window.__yodOVInit){window.__yodOVInit(" + string(raw) + "," + fmt.Sprintf("%d", sel) + ");}"
	return htmlLayer, js
}

// videoOverlayEngineJS — מנוע העורך שמוזרק פעם אחת לכל דף. חשוף דרך window.__yodOVInit.
const videoOverlayEngineJS = `
(function(){
  if(window.__yodOVReady) return;
  window.__yodOVReady=true;
  var OVS=[], SEL=-1, EDIT=true;
  function bridgeSend(obj){
    try{
      var s=JSON.stringify(obj);
      if(window.__יוד_גשר){
        try{
          var xhr=new XMLHttpRequest();
          xhr.open('POST', window.__יוד_גשר, true);
          xhr.setRequestHeader('Content-Type','text/plain;charset=UTF-8');
          xhr.send(s); return;
        }catch(e){}
      }
      if(window.chrome&&window.chrome.webview&&window.chrome.webview.postMessage){
        window.chrome.webview.postMessage(s);
      }
    }catch(e){}
  }
  function hexRGBA(hex,a){
    hex=(hex||'').replace('#','');
    if(hex.length===3) hex=hex[0]+hex[0]+hex[1]+hex[1]+hex[2]+hex[2];
    if(hex.length!==6) return 'rgba(0,0,0,'+a+')';
    var r=parseInt(hex.substr(0,2),16),g=parseInt(hex.substr(2,2),16),b=parseInt(hex.substr(4,2),16);
    return 'rgba('+r+','+g+','+b+','+a+')';
  }
  function vid(){ return document.getElementById('v'); }
  function layerEl(){ return document.getElementById('ov'); }
  function stageEl(){ return document.querySelector('.stage'); }
  // משטח = וידאו אמיתי, או משטח־עיצוב וירטואלי 1280x720 (כשאין וידאו)
  function surface(){
    var v=vid();
    if(v && v.videoWidth>0 && v.videoHeight>0){
      return {rect:v.getBoundingClientRect(), vw:v.videoWidth, vh:v.videoHeight};
    }
    var stage=stageEl();
    if(stage){ return {rect:stage.getBoundingClientRect(), vw:1280, vh:720}; }
    return null;
  }
  function metrics(){
    var s=surface(); var stage=stageEl(); var lay=layerEl();
    if(!s||!stage||!lay) return null;
    var r=s.rect; var st=stage.getBoundingClientRect();
    var scale=Math.min(r.width/s.vw, r.height/s.vh);
    if(!isFinite(scale)||scale<=0) scale=1;
    var dispW=s.vw*scale, dispH=s.vh*scale;
    var offX=(r.left-st.left)+(r.width-dispW)/2;
    var offY=(r.top-st.top)+(r.height-dispH)/2;
    return {vw:s.vw,vh:s.vh,scale:scale,dispW:dispW,dispH:dispH,offX:offX,offY:offY};
  }
  function build(){
    var lay=layerEl(); if(!lay) return;
    lay.innerHTML='';
    for(var i=0;i<OVS.length;i++){
      (function(idx){
        var o=OVS[idx];
        var el=document.createElement('div');
        el.className='ov'+(idx===SEL?' sel':'');
        el.setAttribute('data-i',idx);
        el.textContent=o.text;
        el.style.color=o.color||'#fff';
        el.style.fontWeight='700';
        el.style.background=o.box?hexRGBA(o.boxColor,o.boxOp):'transparent';
        el.style.textShadow=o.shadow?'0 1px 2px rgba(0,0,0,.55)':'none';
        var h=document.createElement('span'); h.className='ovhandle'; el.appendChild(h);
        lay.appendChild(el);
        wireEl(el, idx);
      })(i);
    }
    layout();
  }
  function place(el, o, m){
    el.style.fontSize=Math.max(9,(o.size*m.scale))+'px';
    el.style.left=''; el.style.top=''; el.style.right=''; el.style.bottom=''; el.style.transform='';
    var pad=Math.round(40*m.scale);
    var pos=o.pos||'למטה_מרכז';
    if(pos==='מותאם'||pos==='מדויק'){
      el.style.left=Math.round(o.x*m.scale)+'px';
      el.style.top=Math.round(o.y*m.scale)+'px';
      return;
    }
    // אנכי
    if(pos.indexOf('למעלה')===0){ el.style.top=pad+'px'; }
    else if(pos.indexOf('אמצע')===0){ el.style.top='50%'; }
    else { el.style.bottom=pad+'px'; }
    // אופקי
    if(pos.indexOf('שמאל')>=0){ el.style.left=pad+'px'; }
    else if(pos.indexOf('ימין')>=0){ el.style.right=pad+'px'; }
    else { el.style.left='50%'; }
    // transform לפי מרכוז
    var tx=(pos.indexOf('שמאל')<0&&pos.indexOf('ימין')<0)?'translateX(-50%)':'';
    var ty=(pos.indexOf('אמצע')===0)?'translateY(-50%)':'';
    var tr=(tx+' '+ty).trim();
    if(tr) el.style.transform=tr;
  }
  function layout(){
    var m=metrics(); if(!m) return;
    var lay=layerEl();
    lay.style.left=m.offX+'px'; lay.style.top=m.offY+'px';
    lay.style.width=m.dispW+'px'; lay.style.height=m.dispH+'px';
    var nodes=lay.querySelectorAll('.ov');
    for(var i=0;i<nodes.length;i++){
      var idx=+nodes[i].getAttribute('data-i');
      place(nodes[i], OVS[idx], m);
    }
    syncTime();
  }
  function syncTime(){
    var lay=layerEl(); if(!lay) return;
    var v=vid();
    var hasV=!!(v && v.videoWidth>0);
    var t=(v&&v.currentTime)?v.currentTime:0;
    var nodes=lay.querySelectorAll('.ov');
    for(var i=0;i<nodes.length;i++){
      var idx=+nodes[i].getAttribute('data-i'); var o=OVS[idx];
      var show=true;
      if(hasV){
        if(o.start!=null && o.start>=0 && t<o.start) show=false;
        if(o.end!=null && o.end>=0 && t>o.end) show=false;
      }
      if(idx===SEL) show=true; // הנבחר תמיד גלוי בעריכה
      nodes[i].style.display=show?'inline-block':'none';
    }
  }
  function toAbsolute(el, m){
    var lay=layerEl();
    var er=el.getBoundingClientRect(); var lr=lay.getBoundingClientRect();
    var left=er.left-lr.left, top=er.top-lr.top;
    el.style.right=''; el.style.bottom=''; el.style.transform='';
    el.style.left=left+'px'; el.style.top=top+'px';
    return {left:left,top:top};
  }
  function hint(txt){
    var d=document.getElementById('dhint'); if(!d) return;
    if(txt){ d.textContent=txt; d.classList.add('on'); }
    else { d.classList.remove('on'); }
  }
  function wireEl(el, idx){
    el.addEventListener('click', function(ev){ ev.stopPropagation(); select(idx,true); });
    el.addEventListener('dblclick', function(ev){
      ev.stopPropagation(); ev.preventDefault();
      select(idx,true);
      bridgeSend({סוג:'עריכה', אינדקס:idx});
    });
    var drag=null;
    el.addEventListener('pointerdown', function(ev){
      if(!EDIT) return;
      if(ev.button!==0) return;
      select(idx,true);
      var m=metrics(); if(!m) return;
      var a=toAbsolute(el, m);
      drag={sx:ev.clientX, sy:ev.clientY, l:a.left, t:a.top, m:m, moved:false};
      try{ el.setPointerCapture(ev.pointerId); }catch(e){}
      ev.preventDefault();
    });
    el.addEventListener('pointermove', function(ev){
      if(!drag) return;
      var dx=ev.clientX-drag.sx, dy=ev.clientY-drag.sy;
      if(Math.abs(dx)+Math.abs(dy)>2) drag.moved=true;
      var nl=drag.l+dx, nt=drag.t+dy;
      nl=Math.max(0, Math.min(nl, drag.m.dispW-el.offsetWidth));
      nt=Math.max(0, Math.min(nt, drag.m.dispH-el.offsetHeight));
      el.style.left=nl+'px'; el.style.top=nt+'px';
      var vx=Math.round(nl/drag.m.scale), vy=Math.round(nt/drag.m.scale);
      hint('x='+vx+'  y='+vy);
      OVS[idx].pos='מותאם'; OVS[idx].x=vx; OVS[idx].y=vy;
    });
    function endDrag(ev){
      if(!drag) return;
      var moved=drag.moved; drag=null; hint('');
      try{ el.releasePointerCapture(ev.pointerId); }catch(e){}
      if(moved){
        bridgeSend({סוג:'גרירה', אינדקס:idx, x:OVS[idx].x, y:OVS[idx].y, מיקום:'מותאם'});
      }
    }
    el.addEventListener('pointerup', endDrag);
    el.addEventListener('pointercancel', endDrag);
  }
  function select(idx, notify){
    SEL=idx;
    var lay=layerEl(); if(lay){
      var nodes=lay.querySelectorAll('.ov');
      for(var i=0;i<nodes.length;i++){
        var j=+nodes[i].getAttribute('data-i');
        if(j===idx) nodes[i].classList.add('sel'); else nodes[i].classList.remove('sel');
      }
    }
    syncTime();
    if(notify) bridgeSend({סוג:'בחר', אינדקס:idx});
  }
  window.__yodOVInit=function(list, sel){
    OVS=Array.isArray(list)?list:[];
    SEL=(sel==null?-1:sel);
    build();
  };
  window.__yodSelectOverlay=function(i){ select(i,false); };
  window.__yodSetEditable=function(b){ EDIT=!!b; };
  function bindVideo(){
    var v=vid();
    if(v){
      v.addEventListener('loadedmetadata', layout);
      v.addEventListener('timeupdate', syncTime);
      v.addEventListener('seeked', syncTime);
      v.addEventListener('play', syncTime);
    }
    window.addEventListener('resize', layout);
    // רענון תקופתי קל למקרה של שינוי גודל חלון בלי אירוע resize
    setInterval(layout, 700);
    layout();
  }
  if(document.readyState!=='loading') bindVideo();
  else document.addEventListener('DOMContentLoaded', bindVideo);
})();
`

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
	st.html = buildVideoHTML(st, false)
	st.browser.Resize()
	_ = st.browser.Show()
	st.browser.NavigateToString(st.html)
}
