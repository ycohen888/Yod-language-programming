package gpu

import "math"

// Vec3 — וקטור תלת־ממדי.
type Vec3 struct {
	X, Y, Z float32
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(s float32) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }

func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l < 1e-8 {
		return Vec3{0, 1, 0}
	}
	return v.Scale(1 / l)
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Dot(b Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// Mat4 — מטריצה עמודה־ראשית (OpenGL).
type Mat4 [16]float32

func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func (a Mat4) Mul(b Mat4) Mat4 {
	var out Mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+row] * b[col*4+k]
			}
			out[col*4+row] = s
		}
	}
	return out
}

func Translation(x, y, z float32) Mat4 {
	m := Identity()
	m[12], m[13], m[14] = x, y, z
	return m
}

func ScaleMat(x, y, z float32) Mat4 {
	m := Identity()
	m[0], m[5], m[10] = x, y, z
	return m
}

func RotationY(rad float32) Mat4 {
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	m := Identity()
	m[0], m[2] = c, s
	m[8], m[10] = -s, c
	return m
}

func RotationX(rad float32) Mat4 {
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	m := Identity()
	m[5], m[6] = c, s
	m[9], m[10] = -s, c
	return m
}

func Perspective(fovDeg, aspect, near, far float32) Mat4 {
	if aspect < 1e-6 {
		aspect = 1
	}
	f := 1 / float32(math.Tan(float64(fovDeg)*math.Pi/180/2))
	nf := 1 / (near - far)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) * nf, -1,
		0, 0, 2 * far * near * nf, 0,
	}
}

func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)
	m := Identity()
	m[0], m[4], m[8] = s.X, s.Y, s.Z
	m[1], m[5], m[9] = u.X, u.Y, u.Z
	m[2], m[6], m[10] = -f.X, -f.Y, -f.Z
	m[12] = -s.Dot(eye)
	m[13] = -u.Dot(eye)
	m[14] = f.Dot(eye)
	return m
}
