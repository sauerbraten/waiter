package geom

import "math"

const (
	DNF = 100.0
	DMF = 16.0
)

type Vector [3]float64

func (v *Vector) Magnitude() float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func (v *Vector) Sub(o *Vector) *Vector {
	v[0] -= o[0]
	v[1] -= o[1]
	v[2] -= o[2]
	return v
}

func (v *Vector) Mul(k float64) *Vector {
	v[0] *= k
	v[1] *= k
	v[2] *= k
	return v
}

func Distance(from, to *Vector) float64 {
	return from.Sub(to).Magnitude()
}

func (v *Vector) Scale(k float64) *Vector {
	if mag := v.Magnitude(); mag > 1e-6 {
		v.Mul(k / mag)
	}
	return v
}
