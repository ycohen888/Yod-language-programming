package highlight

import (
	"html"
	"strings"
)

// ToHTML מדגיש קוד יוד ל־HTML בצבעי PHP Dark+ (Cursor)
func ToHTML(source string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="he" dir="rtl"><head><meta charset="utf-8"/>`)
	b.WriteString(`<title>הדגשת תחביר — יוד</title><style>`)
	b.WriteString(`body{margin:0;background:` + CSSBg + `;color:` + CSSFg + `;`)
	b.WriteString(`font-family:Consolas,"Courier New",monospace;font-size:14px;line-height:1.5}`)
	b.WriteString(`pre{margin:0;padding:20px 24px;white-space:pre;overflow:auto}`)
	b.WriteString(`.c{color:` + CSSComment + `;font-style:italic}`)
	b.WriteString(`.s{color:` + CSSString + `}`)
	b.WriteString(`.n{color:` + CSSNumber + `}`)
	b.WriteString(`.k{color:` + CSSKeyword + `}`)
	b.WriteString(`.w{color:` + CSSControl + `}`)
	b.WriteString(`.f{color:` + CSSFunction + `}`)
	b.WriteString(`.v{color:` + CSSVariable + `}`)
	b.WriteString(`.t{color:` + CSSClass + `}`)
	b.WriteString(`.o{color:` + CSSOperator + `}`)
	b.WriteString(`</style></head><body><pre>`)

	spans := Spans(source)
	runes := []rune(source)
	// המרה מ־UTF-16 לאינדקס rune בקירוב BMP
	pos16 := 0
	spanI := 0
	i := 0
	for i < len(runes) {
		for spanI < len(spans) && spans[spanI].End <= pos16 {
			spanI++
		}
		r := runes[i]
		rLen := 1
		if r > 0xFFFF {
			rLen = 2
		}
		cls := ""
		if spanI < len(spans) && pos16 >= spans[spanI].Start && pos16 < spans[spanI].End {
			cls = kindClass(spans[spanI].Kind)
		}
		esc := html.EscapeString(string(r))
		if cls != "" {
			b.WriteString(`<span class="` + cls + `">` + esc + `</span>`)
		} else {
			b.WriteString(esc)
		}
		pos16 += rLen
		i++
	}
	b.WriteString(`</pre></body></html>`)
	return b.String()
}

func kindClass(k Kind) string {
	switch k {
	case KindComment:
		return "c"
	case KindString:
		return "s"
	case KindNumber:
		return "n"
	case KindControl:
		return "w"
	case KindKeyword, KindConstant:
		return "k"
	case KindFunction:
		return "f"
	case KindVariable:
		return "v"
	case KindClass:
		return "t"
	case KindOperator:
		return "o"
	default:
		return ""
	}
}
