package orboperations

import "github.com/paulmach/orb"

// Union computes the union of two geometries.
// Returns a geometry representing the union of geom1 and geom2.
func Union(geom1, geom2 orb.Geometry) orb.Geometry {
	return booleanOperation(geom1, geom2, unionOp)
}

