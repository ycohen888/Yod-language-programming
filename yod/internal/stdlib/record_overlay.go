package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"yod/internal/object"
)

// textOverlay — שכבת טקסט לצריבה / תצוגה מקדימה.
type textOverlay struct {
	Text       string
	Size       float64
	Color      string
	Position   string
	X          float64
	Y          float64
	BoxColor   string
	BoxOpacity float64
	Box        bool
	Shadow     bool
	Start      float64 // שניות; <0 = מההתחלה
	End        float64 // שניות; <0 = עד הסוף
	Font       string
}

func parseTextOverlays(arr *object.Array) ([]textOverlay, error) {
	if arr == nil {
		return nil, fmt.Errorf("רשימת טקסטים ריקה")
	}
	out := make([]textOverlay, 0, len(arr.Elements))
	for i, el := range arr.Elements {
		h, ok := el.(*object.Hash)
		if !ok {
			return nil, fmt.Errorf("פריט %d חייב להיות מילון", i+1)
		}
		o, err := parseTextOverlayHash(h)
		if err != nil {
			return nil, fmt.Errorf("פריט %d: %w", i+1, err)
		}
		out = append(out, o)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("אין שכבות טקסט להחלה")
	}
	return out, nil
}

func parseTextOverlayHash(h *object.Hash) (textOverlay, error) {
	o := textOverlay{
		Size:       36,
		Color:      "#ffffff",
		Position:   "למטה_מרכז",
		BoxOpacity: 0.45,
		Shadow:     true,
		Start:      -1,
		End:        -1,
	}
	if h == nil || h.Pairs == nil {
		return o, fmt.Errorf("מילון ריק")
	}
	if v, ok := h.Pairs["טקסט"]; ok {
		if s, ok := asString(v); ok {
			o.Text = s
		} else {
			o.Text = v.Inspect()
		}
	}
	if strings.TrimSpace(o.Text) == "" {
		return o, fmt.Errorf("חסר שדה «טקסט»")
	}
	if v, ok := h.Pairs["גודל"]; ok {
		if n, ok := asNumber(v); ok && n > 0 {
			o.Size = n
		}
	}
	if v, ok := h.Pairs["צבע"]; ok {
		if s, ok := asString(v); ok && strings.TrimSpace(s) != "" {
			o.Color = normalizeOverlayColor(s)
		}
	}
	if v, ok := h.Pairs["מיקום"]; ok {
		if s, ok := asString(v); ok && strings.TrimSpace(s) != "" {
			o.Position = strings.TrimSpace(s)
		}
	}
	if v, ok := h.Pairs["x"]; ok {
		if n, ok := asNumber(v); ok {
			o.X = n
		}
	}
	if v, ok := h.Pairs["y"]; ok {
		if n, ok := asNumber(v); ok {
			o.Y = n
		}
	}
	if v, ok := h.Pairs["רקע"]; ok {
		if s, ok := asString(v); ok && strings.TrimSpace(s) != "" {
			o.Box = true
			o.BoxColor = normalizeOverlayColor(s)
		}
	}
	if v, ok := h.Pairs["שקיפות_רקע"]; ok {
		if n, ok := asNumber(v); ok {
			if n < 0 {
				n = 0
			}
			if n > 1 {
				n = 1
			}
			o.BoxOpacity = n
			if o.BoxColor == "" {
				o.Box = true
				o.BoxColor = "#000000"
			}
		}
	}
	if v, ok := h.Pairs["צל"]; ok {
		if b, ok := v.(*object.Boolean); ok {
			o.Shadow = b.Value
		}
	}
	if v, ok := h.Pairs["התחלה"]; ok {
		if n, ok := asNumber(v); ok {
			o.Start = n
		}
	}
	if v, ok := h.Pairs["סיום"]; ok {
		if n, ok := asNumber(v); ok {
			o.End = n
		}
	}
	if v, ok := h.Pairs["גופן"]; ok {
		if s, ok := asString(v); ok {
			o.Font = strings.TrimSpace(s)
		}
	}
	return o, nil
}

func normalizeOverlayColor(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "לבן", "white":
		return "#ffffff"
	case "שחור", "black":
		return "#000000"
	case "אדום", "red":
		return "#ef4444"
	case "ירוק", "green":
		return "#22c55e"
	case "כחול", "blue":
		return "#3b82f6"
	case "צהוב", "yellow":
		return "#eab308"
	case "כתום", "orange":
		return "#f97316"
	case "סגול", "purple":
		return "#a855f7"
	case "אפור", "gray", "grey":
		return "#94a3b8"
	}
	if strings.HasPrefix(s, "#") && len(s) >= 4 {
		return s
	}
	return s
}

func overlayXYExpr(o textOverlay) (xExpr, yExpr string) {
	pad := "40"
	switch o.Position {
	case "למעלה_שמאל", "שמאל_למעלה":
		return pad, pad
	case "למעלה_מרכז", "למעלה":
		return "(w-text_w)/2", pad
	case "למעלה_ימין", "ימין_למעלה":
		return "w-text_w-" + pad, pad
	case "אמצע_שמאל":
		return pad, "(h-text_h)/2"
	case "אמצע", "אמצע_מרכז", "מרכז":
		return "(w-text_w)/2", "(h-text_h)/2"
	case "אמצע_ימין":
		return "w-text_w-" + pad, "(h-text_h)/2"
	case "למטה_שמאל", "שמאל_למטה":
		return pad, "h-text_h-" + pad
	case "למטה_מרכז", "למטה":
		return "(w-text_w)/2", "h-text_h-" + pad
	case "למטה_ימין", "ימין_למטה":
		return "w-text_w-" + pad, "h-text_h-" + pad
	case "מותאם", "מדויק", "":
		return strconv.FormatFloat(o.X, 'f', -1, 64), strconv.FormatFloat(o.Y, 'f', -1, 64)
	default:
		// אם לא הוכר — נסה כמותאם
		return strconv.FormatFloat(o.X, 'f', -1, 64), strconv.FormatFloat(o.Y, 'f', -1, 64)
	}
}

func findHebrewFont(preferred string) string {
	if preferred != "" {
		if st, err := os.Stat(preferred); err == nil && !st.IsDir() {
			return preferred
		}
	}
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	fonts := filepath.Join(windir, "Fonts")
	candidates := []string{
		"arial.ttf",
		"arialbd.ttf",
		"segoeui.ttf",
		"segoeuib.ttf",
		"tahoma.ttf",
		"tahomabd.ttf",
		"david.ttf",
		"davidbd.ttf",
		"rubik-regular.ttf",
		"heebo-regular.ttf",
	}
	for _, name := range candidates {
		p := filepath.Join(fonts, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func ffmpegFilterEscapePath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, `\`, `/`)
	p = strings.ReplaceAll(p, ":", `\:`)
	p = strings.ReplaceAll(p, "'", `\'`)
	return p
}

func buildDrawtextFilter(overlays []textOverlay, workDir string) (filter string, cleanup []string, err error) {
	fontDefault := findHebrewFont("")
	if fontDefault == "" {
		return "", nil, fmt.Errorf("לא נמצא גופן עברי ב־Windows\\Fonts (Arial/Segoe UI)")
	}
	parts := make([]string, 0, len(overlays))
	for i, o := range overlays {
		font := findHebrewFont(o.Font)
		if font == "" {
			font = fontDefault
		}
		txtPath := filepath.Join(workDir, fmt.Sprintf("yod_txt_%d.txt", i))
		// UTF-8 עם BOM עוזר ל־ffmpeg ב־Windows עם עברית
		content := []byte("\ufeff" + o.Text)
		if werr := os.WriteFile(txtPath, content, 0o644); werr != nil {
			return "", cleanup, fmt.Errorf("כתיבת קובץ טקסט זמני נכשלה: %v", werr)
		}
		cleanup = append(cleanup, txtPath)

		xExpr, yExpr := overlayXYExpr(o)
		var b strings.Builder
		b.WriteString("drawtext=fontfile='")
		b.WriteString(ffmpegFilterEscapePath(font))
		b.WriteString("':textfile='")
		b.WriteString(ffmpegFilterEscapePath(txtPath))
		b.WriteString("':fontsize=")
		b.WriteString(strconv.FormatFloat(o.Size, 'f', -1, 64))
		b.WriteString(":fontcolor=")
		b.WriteString(o.Color)
		b.WriteString(":x=")
		b.WriteString(xExpr)
		b.WriteString(":y=")
		b.WriteString(yExpr)
		if o.Box {
			bc := o.BoxColor
			if bc == "" {
				bc = "#000000"
			}
			b.WriteString(":box=1:boxborderw=10:boxcolor=")
			b.WriteString(bc)
			b.WriteString("@")
			b.WriteString(strconv.FormatFloat(o.BoxOpacity, 'f', 2, 64))
		}
		if o.Shadow {
			b.WriteString(":shadowx=2:shadowy=2:shadowcolor=black@0.55")
		}
		if o.Start >= 0 || o.End >= 0 {
			start := o.Start
			end := o.End
			if start < 0 {
				start = 0
			}
			if end < 0 {
				end = 86400 // עד סוף סרטון מעשי
			}
			b.WriteString(":enable='between(t,")
			b.WriteString(strconv.FormatFloat(start, 'f', -1, 64))
			b.WriteString(",")
			b.WriteString(strconv.FormatFloat(end, 'f', -1, 64))
			b.WriteString(")'")
		}
		parts = append(parts, b.String())
	}
	return strings.Join(parts, ","), cleanup, nil
}

func overlaysToPreviewArray(overlays []textOverlay) *object.Array {
	arr := &object.Array{Elements: make([]object.Object, 0, len(overlays))}
	for _, o := range overlays {
		h := object.NewHash()
		h.Set("טקסט", &object.String{Value: o.Text})
		h.Set("גודל", &object.Number{Value: o.Size})
		h.Set("צבע", &object.String{Value: o.Color})
		h.Set("מיקום", &object.String{Value: o.Position})
		h.Set("x", &object.Number{Value: o.X})
		h.Set("y", &object.Number{Value: o.Y})
		h.Set("צל", &object.Boolean{Value: o.Shadow})
		if o.Box {
			h.Set("רקע", &object.String{Value: o.BoxColor})
			h.Set("שקיפות_רקע", &object.Number{Value: o.BoxOpacity})
		}
		if o.Start >= 0 {
			h.Set("התחלה", &object.Number{Value: o.Start})
		}
		if o.End >= 0 {
			h.Set("סיום", &object.Number{Value: o.End})
		}
		arr.Elements = append(arr.Elements, h)
	}
	return arr
}
