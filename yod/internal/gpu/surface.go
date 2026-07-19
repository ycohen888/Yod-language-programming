package gpu

import (
	"fmt"
	"sync"
	"time"
)

// Surface — משטח GPU על HWND (מארח Composite ב־חלונות).
type Surface struct {
	mu      sync.Mutex
	backend Backend
	width   int
	height  int
	clearR  float32
	clearG  float32
	clearB  float32
	mode    string // "משולש" | "סצנה" | ""
	scene   *Scene
	angle   float64
	fps     float64
	frames  int
	lastFPS time.Time
	ready   bool
}

// NewSurface יוצר משטח בלי הקשר GL עדיין (Attach אחרי Create של החלון).
func NewSurface(w, h int) *Surface {
	if w < 40 {
		w = 40
	}
	if h < 40 {
		h = 40
	}
	return &Surface{
		width:   w,
		height:  h,
		clearR:  0.08,
		clearG:  0.10,
		clearB:  0.14,
		mode:    "משולש",
		lastFPS: time.Now(),
	}
}

func (s *Surface) Width() int  { return s.width }
func (s *Surface) Height() int { return s.height }
func (s *Surface) Ready() bool { return s.ready }
func (s *Surface) FPS() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fps
}
func (s *Surface) BackendName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil {
		return ""
	}
	return s.backend.Name()
}

// Attach מחבר הקשר OpenGL ל־HWND של המארח.
func (s *Surface) Attach(hwnd uintptr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return nil
	}
	b := newOpenGLBackend()
	if err := b.Init(hwnd); err != nil {
		return err
	}
	s.backend = b
	s.ready = true
	_ = b.MakeCurrent()
	b.Viewport(s.width, s.height)
	return nil
}

func (s *Surface) Resize(w, h int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	s.width, s.height = w, h
	if s.ready && s.backend != nil {
		_ = s.backend.MakeCurrent()
		s.backend.Viewport(w, h)
	}
}

func (s *Surface) SetClear(r, g, b float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearR, s.clearG, s.clearB = r, g, b
}

func (s *Surface) SetMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

func (s *Surface) SetScene(sc *Scene) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scene = sc
	if sc != nil {
		s.mode = "סצנה"
	}
}

func (s *Surface) Scene() *Scene {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scene
}

func (s *Surface) LoadTextureRGBA(pix []byte, w, h int) (TextureID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready || s.backend == nil {
		return 0, fmt.Errorf("משטח GPU לא מחובר")
	}
	if err := s.backend.MakeCurrent(); err != nil {
		return 0, err
	}
	return s.backend.LoadTextureRGBA(pix, w, h)
}

// Present מצייר פריים ומחליף באפרים.
func (s *Surface) Present() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready || s.backend == nil {
		return fmt.Errorf("משטח GPU לא מחובר עדיין")
	}
	if err := s.backend.MakeCurrent(); err != nil {
		return err
	}
	s.backend.Viewport(s.width, s.height)
	s.backend.Clear(s.clearR, s.clearG, s.clearB, 1)
	switch s.mode {
	case "סצנה":
		if s.scene != nil {
			aspect := float32(s.width) / float32(s.height)
			if aspect < 1e-6 {
				aspect = 1
			}
			meshes := s.scene.BuildDrawMeshes()
			s.backend.DrawScene(s.scene.Camera, s.scene.Light, meshes, aspect)
		}
	default:
		s.angle += 0.025
		s.backend.DrawColoredTriangle(s.angle)
	}
	if err := s.backend.Swap(); err != nil {
		return err
	}
	s.frames++
	if time.Since(s.lastFPS) >= time.Second {
		s.fps = float64(s.frames) / time.Since(s.lastFPS).Seconds()
		s.frames = 0
		s.lastFPS = time.Now()
	}
	return nil
}

func (s *Surface) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend != nil {
		s.backend.Destroy()
		s.backend = nil
	}
	s.ready = false
}
