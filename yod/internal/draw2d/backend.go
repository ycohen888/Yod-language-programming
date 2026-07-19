// Package draw2d — שכבת ציור 2D ליוד (Backend: CPU או Direct2D).
package draw2d

import (
	"image"
	"image/color"
)

// Align — יישור טקסט (תואם drawBoard.align).
const (
	AlignLeft   = 0
	AlignRight  = 1
	AlignCenter = 2
)

// Backend2D — ממשק מאיץ לציור משטח (יחידות: פיקסלי התקן אחרי DPR).
type Backend2D interface {
	// Resize משנה את גודל יעד הציור בפיקסלים טבעיים.
	Resize(physW, physH, dpi int) error
	// BindHWND מחבר חלון ל־Present (אופציונלי; CPU מתעלם).
	BindHWND(hwnd uintptr) error
	// Name מחזיר שם הבקאנד ("cpu" / "direct2d").
	Name() string

	Clear(c color.RGBA)
	FillRect(x, y, w, h int, c color.RGBA)
	StrokeRect(x, y, w, h, thickness int, c color.RGBA)
	FillRoundedRect(x, y, w, h, radius int, c color.RGBA)
	StrokeLine(x0, y0, x1, y1, thickness int, c color.RGBA)
	FillCircle(cx, cy, radius int, c color.RGBA)
	StrokeCircle(cx, cy, radius, thickness int, c color.RGBA)
	StrokeEllipse(cx, cy, rx, ry, thickness int, c color.RGBA)
	DrawImage(src image.Image, x, y, dw, dh int, angleDeg float64, flipH bool)
	DrawText(s string, x, y int, fontSize float64, c color.RGBA, align int)
	FloodFill(x, y int, c color.RGBA)

	// Present מרענן את התצוגה ל־HWND אם מחובר.
	Present() error
	// SnapshotRGBA מחזיר עותק לפיקסלים (שמור / קרא_צבע / בטל).
	SnapshotRGBA() (*image.RGBA, error)
	// ReplacePixels מחליף את כל תוכן הלוח (למשל אחרי בטל).
	ReplacePixels(src *image.RGBA) error
	// Buffer מחזיר את בופר ה־RGBA הפנימי אם קיים (אופציונלי; CPU תמיד).
	Buffer() *image.RGBA

	Destroy()
}
