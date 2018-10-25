package geom

import "math"

const (
	DNF = 100.0
	DMF = 16.0
)

type Vector struct {
	x, y, z float64
}

func NewVector(x, y, z float64) *Vector {
	return &Vector{x, y, z}
}

func (v *Vector) X() float64 { return v.x }
func (v *Vector) Y() float64 { return v.y }
func (v *Vector) Z() float64 { return v.z }

func (v *Vector) IsZero() bool { return v.x == 0 && v.y == 0 && v.z == 0 }

func (v *Vector) Magnitude() float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z)
}

func (v *Vector) Sub(o *Vector) *Vector {
	v.x -= o.x
	v.y -= o.y
	v.z -= o.z
	return v
}

func (v *Vector) Mul(k float64) *Vector {
	v.x *= k
	v.y *= k
	v.z *= k
	return v
}

func (v *Vector) Scale(k float64) *Vector {
	if mag := v.Magnitude(); mag > 1e-6 {
		v.Mul(k / mag)
	}
	return v
}

func Distance(from, to *Vector) float64 {
	return from.Sub(to).Magnitude()
}
