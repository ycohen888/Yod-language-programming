package draw2d

// NewBackend יוצר באקאנד מועדף (Direct2D ב־Windows, אחרת CPU).
func NewBackend(physW, physH, dpi int) Backend2D {
	if b, err := tryNewD2D(physW, physH, dpi); err == nil && b != nil {
		return b
	}
	return NewCPUBackend(physW, physH, dpi)
}
