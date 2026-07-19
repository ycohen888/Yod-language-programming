//go:build windows

package draw2d

func tryNewD2D(physW, physH, dpi int) (Backend2D, error) {
	return NewD2DBackend(physW, physH, dpi)
}
