//go:build windows

package stdlib

import (
	"image"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"yod/internal/console"
	"yod/internal/object"
)

func showDrawing(img image.Image, title string) object.Object {
	console.HideIfOwned()

	rgba := imageToRGBA(img)
	pw, ph := rgba.Bounds().Dx(), rgba.Bounds().Dy()
	if pw < 1 || ph < 1 {
		return errObj("אין מה להציג")
	}

	dpi := screenDPI()
	// Bitmap ב־DPI של המסך → ImageView/CustomWidget מציירים 1:1 בלי מתיחה
	bmp, err := walk.NewBitmapFromImageForDPI(rgba, dpi)
	if err != nil {
		return errObj("הצגת ציור נכשלה: " + err.Error())
	}
	defer bmp.Dispose()

	// גודל ב־DIP כך שהשטח בפיקסלים = גודל הלוח
	dipW := pixelsToDIP(pw, dpi)
	dipH := pixelsToDIP(ph, dpi)

	_, err = (MainWindow{
		Title:   title,
		MinSize: Size{Width: dipW + 16, Height: dipH + 40},
		Size:    Size{Width: dipW + 16, Height: dipH + 40},
		Layout:  VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
		Children: []Widget{
			CustomWidget{
				MinSize:             Size{Width: dipW, Height: dipH},
				MaxSize:             Size{Width: dipW, Height: dipH},
				InvalidatesOnResize: true,
				PaintMode:           PaintBuffered,
				Paint: func(canvas *walk.Canvas, bounds walk.Rectangle) error {
					bg, err := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
					if err != nil {
						return err
					}
					defer bg.Dispose()
					if err := canvas.FillRectangle(bg, bounds); err != nil {
						return err
					}
					// ציור בגודל פיקסל מדויק של הלוח (זהה לקובץ השמור)
					dest := walk.Rectangle{
						X:      bounds.X,
						Y:      bounds.Y,
						Width:  pw,
						Height: ph,
					}
					return canvas.DrawImageStretchedPixels(bmp, dest)
				},
			},
		},
	}).Run()
	if err != nil {
		return errObj("הצגת ציור נכשלה: " + err.Error())
	}
	return object.Nil
}

func screenDPI() int {
	hdc := win.GetDC(0)
	if hdc == 0 {
		return 96
	}
	defer win.ReleaseDC(0, hdc)
	dpi := int(win.GetDeviceCaps(hdc, win.LOGPIXELSX))
	if dpi < 96 {
		return 96
	}
	return dpi
}

func pixelsToDIP(px, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return int(float64(px)*96.0/float64(dpi) + 0.5)
}
