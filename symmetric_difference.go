package orboperations

import "github.com/paulmach/orb"

// SymmetricDifference computes the symmetric difference of two geometries.
// Returns a geometry representing (geom1 - geom2) ∪ (geom2 - geom1).
func SymmetricDifference(geom1, geom2 orb.Geometry) orb.Geometry {
	diff1 := Difference(geom1, geom2)
	diff2 := Difference(geom2, geom1)
	return Union(diff1, diff2)
}

