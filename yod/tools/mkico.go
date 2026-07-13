// כלי חד-פעמי: PNG → ICO עם כמה גדלים
//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func main() {
	srcPath := filepath.Join("assets", "yod-icon-source.png")
	dstPath := filepath.Join("assets", "yod.ico")
	if len(os.Args) >= 2 {
		srcPath = os.Args[1]
	}
	if len(os.Args) >= 3 {
		dstPath = os.Args[2]
	}

	f, err := os.Open(srcPath)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		fatal(err)
	}

	sizes := []int{16, 32, 48, 64, 128, 256}
	var pngs [][]byte
	for _, s := range sizes {
		img := resizeRGBA(src, s)
		var buf []byte
		buf, err = encodePNG(img)
		if err != nil {
			fatal(err)
		}
		pngs = append(pngs, buf)
	}

	if err := writeICO(dstPath, sizes, pngs); err != nil {
		fatal(err)
	}
	fmt.Println("נוצר:", dstPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func resizeRGBA(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var b []byte
	w := &byteWriter{buf: &b}
	if err := png.Encode(w, img); err != nil {
		return nil, err
	}
	return b, nil
}

type byteWriter struct{ buf *[]byte }

func (w *byteWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

func writeICO(path string, sizes []int, pngs [][]byte) error {
	n := len(sizes)
	headerSize := 6 + 16*n
	offset := headerSize
	out := make([]byte, headerSize)
	// ICONDIR
	binary.LittleEndian.PutUint16(out[0:], 0) // reserved
	binary.LittleEndian.PutUint16(out[2:], 1) // type icon
	binary.LittleEndian.PutUint16(out[4:], uint16(n))

	for i, s := range sizes {
		entry := out[6+i*16 : 6+(i+1)*16]
		w, h := s, s
		if w >= 256 {
			entry[0] = 0
		} else {
			entry[0] = byte(w)
		}
		if h >= 256 {
			entry[1] = 0
		} else {
			entry[1] = byte(h)
		}
		entry[2] = 0 // colors
		entry[3] = 0 // reserved
		binary.LittleEndian.PutUint16(entry[4:], 1)  // planes
		binary.LittleEndian.PutUint16(entry[6:], 32) // bit count
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(pngs[i])))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		out = append(out, p...)
	}
	return os.WriteFile(path, out, 0644)
}
