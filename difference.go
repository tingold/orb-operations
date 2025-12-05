package orboperations

import "github.com/paulmach/orb"

// Difference computes the difference of two geometries (geom1 - geom2).
// Returns a geometry representing geom1 minus geom2.
func Difference(geom1, geom2 orb.Geometry) orb.Geometry {
	return booleanOperation(geom1, geom2, differenceOp)
}

