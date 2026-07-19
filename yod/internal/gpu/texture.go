package gpu

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// LoadImageRGBA טוען PNG/JPEG לקבצי RGBA (שורה עליונה = v גבוה ל־OpenGL אם flip=true).
func LoadImageRGBA(path string, flipY bool) (pix []byte, w, h int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("קריאת תמונה: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("פענוח תמונה: %w", err)
	}
	b := img.Bounds()
	w, h = b.Dx(), b.Dy()
	pix = make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		srcY := y
		if flipY {
			srcY = h - 1 - y
		}
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+srcY).RGBA()
			i := (y*w + x) * 4
			pix[i] = byte(r >> 8)
			pix[i+1] = byte(g >> 8)
			pix[i+2] = byte(bl >> 8)
			pix[i+3] = byte(a >> 8)
		}
	}
	return pix, w, h, nil
}
