package draw2d

import (
	"image/color"
	"testing"
)

func TestCPUBackendBasic(t *testing.T) {
	b := NewCPUBackend(100, 80, 96)
	defer b.Destroy()
	if b.Name() != "cpu" {
		t.Fatalf("name=%s", b.Name())
	}
	b.Clear(color.RGBA{10, 20, 30, 255})
	b.FillRect(10, 10, 20, 20, color.RGBA{255, 0, 0, 255})
	c := b.Buffer().RGBAAt(15, 15)
	if c.R < 200 {
		t.Fatalf("expected red fill, got %+v", c)
	}
	b.FillRoundedRect(40, 10, 30, 20, 5, color.RGBA{0, 255, 0, 255})
	b.StrokeLine(0, 0, 50, 50, 2, color.RGBA{0, 0, 255, 255})
	b.FillCircle(70, 40, 8, color.RGBA{255, 255, 0, 255})
	snap, err := b.SnapshotRGBA()
	if err != nil || snap == nil {
		t.Fatal(err)
	}
	if snap.Bounds().Dx() != 100 || snap.Bounds().Dy() != 80 {
		t.Fatalf("size %v", snap.Bounds())
	}
	if err := b.Resize(120, 90, 144); err != nil {
		t.Fatal(err)
	}
	if b.Buffer().Bounds().Dx() != 120 {
		t.Fatal("resize failed")
	}
}

func TestNewBackendPreferCPUFallback(t *testing.T) {
	b := NewBackend(64, 64, 96)
	defer b.Destroy()
	if b.Buffer() == nil {
		t.Fatal("nil buffer")
	}
	b.Clear(color.RGBA{255, 255, 255, 255})
	_ = b.Present()
}
