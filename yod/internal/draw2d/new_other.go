//go:build !windows

package draw2d

import "fmt"

func tryNewD2D(physW, physH, dpi int) (Backend2D, error) {
	return nil, fmt.Errorf("Direct2D זמין רק ב־Windows")
}
