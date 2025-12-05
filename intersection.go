package orboperations

import "github.com/paulmach/orb"

// Intersection computes the intersection of two geometries.
// Returns a geometry representing the intersection of geom1 and geom2.
func Intersection(geom1, geom2 orb.Geometry) orb.Geometry {
	return booleanOperation(geom1, geom2, intersectionOp)
}

