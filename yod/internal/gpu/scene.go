package gpu

import "math"

// MeshData — נתוני רשת גולמיים (לפני טרנספורם עולם).
type MeshData struct {
	Name      string
	Positions []float32
	Normals   []float32
	UVs       []float32
	Indices   []uint32
}

// Node — אובייקט בסצנה.
type Node struct {
	Name     string
	Mesh     *MeshData
	Position Vec3
	Rotation Vec3 // רדיאנים: X,Y,Z
	Scale    Vec3
	Color    [3]float32
	Texture  TextureID
}

func (n *Node) ModelMatrix() Mat4 {
	sx, sy, sz := n.Scale.X, n.Scale.Y, n.Scale.Z
	if sx == 0 && sy == 0 && sz == 0 {
		sx, sy, sz = 1, 1, 1
	}
	m := Translation(n.Position.X, n.Position.Y, n.Position.Z)
	m = m.Mul(RotationY(n.Rotation.Y))
	m = m.Mul(RotationX(n.Rotation.X))
	m = m.Mul(ScaleMat(sx, sy, sz))
	return m
}

// Scene — מצלמה + אור + צמתים.
type Scene struct {
	Camera Camera
	Light  Light
	Nodes  []*Node
}

func NewScene() *Scene {
	return &Scene{
		Camera: DefaultCamera(),
		Light:  DefaultLight(),
		Nodes:  nil,
	}
}

func (s *Scene) AddNode(n *Node) {
	if n == nil {
		return
	}
	if n.Scale.X == 0 && n.Scale.Y == 0 && n.Scale.Z == 0 {
		n.Scale = Vec3{1, 1, 1}
	}
	if n.Color[0] == 0 && n.Color[1] == 0 && n.Color[2] == 0 {
		n.Color = [3]float32{0.85, 0.85, 0.9}
	}
	s.Nodes = append(s.Nodes, n)
}

func (s *Scene) Find(name string) *Node {
	for _, n := range s.Nodes {
		if n.Name == name {
			return n
		}
	}
	return nil
}

func (s *Scene) BuildDrawMeshes() []DrawMesh {
	out := make([]DrawMesh, 0, len(s.Nodes))
	for _, n := range s.Nodes {
		if n == nil || n.Mesh == nil {
			continue
		}
		out = append(out, DrawMesh{
			Positions: n.Mesh.Positions,
			Normals:   n.Mesh.Normals,
			UVs:       n.Mesh.UVs,
			Indices:   n.Mesh.Indices,
			Color:     n.Color,
			Texture:   n.Texture,
			Model:     n.ModelMatrix(),
		})
	}
	return out
}

// NewCubeMesh — קוביית יחידה סביב המרכז.
func NewCubeMesh() *MeshData {
	// 6 פאות × 2 משולשים × 3 קודקודים (expanded) עם נורמלים ו־UV
	type v struct{ x, y, z, nx, ny, nz, u, v float32 }
	faces := [][]v{
		{{0.5, -0.5, 0.5, 1, 0, 0, 0, 0}, {0.5, 0.5, 0.5, 1, 0, 0, 0, 1}, {0.5, 0.5, -0.5, 1, 0, 0, 1, 1}, {0.5, -0.5, -0.5, 1, 0, 0, 1, 0}},
		{{-0.5, -0.5, -0.5, -1, 0, 0, 0, 0}, {-0.5, 0.5, -0.5, -1, 0, 0, 0, 1}, {-0.5, 0.5, 0.5, -1, 0, 0, 1, 1}, {-0.5, -0.5, 0.5, -1, 0, 0, 1, 0}},
		{{-0.5, 0.5, 0.5, 0, 1, 0, 0, 0}, {-0.5, 0.5, -0.5, 0, 1, 0, 0, 1}, {0.5, 0.5, -0.5, 0, 1, 0, 1, 1}, {0.5, 0.5, 0.5, 0, 1, 0, 1, 0}},
		{{-0.5, -0.5, -0.5, 0, -1, 0, 0, 0}, {-0.5, -0.5, 0.5, 0, -1, 0, 0, 1}, {0.5, -0.5, 0.5, 0, -1, 0, 1, 1}, {0.5, -0.5, -0.5, 0, -1, 0, 1, 0}},
		{{-0.5, -0.5, 0.5, 0, 0, 1, 0, 0}, {-0.5, 0.5, 0.5, 0, 0, 1, 0, 1}, {0.5, 0.5, 0.5, 0, 0, 1, 1, 1}, {0.5, -0.5, 0.5, 0, 0, 1, 1, 0}},
		{{0.5, -0.5, -0.5, 0, 0, -1, 0, 0}, {0.5, 0.5, -0.5, 0, 0, -1, 0, 1}, {-0.5, 0.5, -0.5, 0, 0, -1, 1, 1}, {-0.5, -0.5, -0.5, 0, 0, -1, 1, 0}},
	}
	m := &MeshData{Name: "קוביה"}
	var idx uint32
	for _, f := range faces {
		for _, tri := range [][3]int{{0, 1, 2}, {0, 2, 3}} {
			for _, ti := range tri {
				p := f[ti]
				m.Positions = append(m.Positions, p.x, p.y, p.z)
				m.Normals = append(m.Normals, p.nx, p.ny, p.nz)
				m.UVs = append(m.UVs, p.u, p.v)
				m.Indices = append(m.Indices, idx)
				idx++
			}
		}
	}
	return m
}

// NewPlaneMesh — מישור XZ.
func NewPlaneMesh(size float32) *MeshData {
	if size <= 0 {
		size = 4
	}
	h := size / 2
	m := &MeshData{
		Name: "מישור",
		Positions: []float32{
			-h, 0, -h, h, 0, -h, h, 0, h,
			-h, 0, -h, h, 0, h, -h, 0, h,
		},
		Normals: []float32{
			0, 1, 0, 0, 1, 0, 0, 1, 0,
			0, 1, 0, 0, 1, 0, 0, 1, 0,
		},
		UVs: []float32{
			0, 0, 1, 0, 1, 1,
			0, 0, 1, 1, 0, 1,
		},
	}
	return m
}

// OrbitCamera מעדכן מיקום מצלמה במסלול סביב Target.
func OrbitCamera(cam *Camera, yawDeg, pitchDeg, dist float64) {
	yaw := yawDeg * math.Pi / 180
	pitch := pitchDeg * math.Pi / 180
	if pitch > 1.4 {
		pitch = 1.4
	}
	if pitch < -1.4 {
		pitch = -1.4
	}
	if dist < 0.5 {
		dist = 0.5
	}
	cp := math.Cos(pitch)
	cam.Eye = Vec3{
		X: cam.Target.X + float32(dist*cp*math.Sin(yaw)),
		Y: cam.Target.Y + float32(dist*math.Sin(pitch)),
		Z: cam.Target.Z + float32(dist*cp*math.Cos(yaw)),
	}
}
