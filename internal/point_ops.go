package internal

import (
	"github.com/paulmach/orb"
)

// PointUnion computes the union of two points (returns MultiPoint with both).
func PointUnion(p1, p2 orb.Point) orb.MultiPoint {
	if PointsEqual(p1, p2, Epsilon) {
		return orb.MultiPoint{p1}
	}
	return orb.MultiPoint{p1, p2}
}

// PointIntersection computes the intersection of two points.
// Returns the point if they're equal, empty otherwise.
func PointIntersection(p1, p2 orb.Point) orb.MultiPoint {
	if PointsEqual(p1, p2, Epsilon) {
		return orb.MultiPoint{p1}
	}
	return orb.MultiPoint{}
}

// PointDifference computes p1 - p2.
// Returns p1 if they're different, empty otherwise.
func PointDifference(p1, p2 orb.Point) orb.MultiPoint {
	if PointsEqual(p1, p2, Epsilon) {
		return orb.MultiPoint{}
	}
	return orb.MultiPoint{p1}
}

// MultiPointUnion computes the union of two multi-points.
func MultiPointUnion(mp1, mp2 orb.MultiPoint) orb.MultiPoint {
	seen := make(map[orb.Point]bool)
	result := make(orb.MultiPoint, 0)

	// Add points from mp1
	for _, pt := range mp1 {
		key := roundPoint(pt)
		if !seen[key] {
			result = append(result, pt)
			seen[key] = true
		}
	}

	// Add points from mp2
	for _, pt := range mp2 {
		key := roundPoint(pt)
		if !seen[key] {
			result = append(result, pt)
			seen[key] = true
		}
	}

	return result
}

// MultiPointIntersection computes the intersection of two multi-points.
func MultiPointIntersection(mp1, mp2 orb.MultiPoint) orb.MultiPoint {
	seen := make(map[orb.Point]bool)
	result := make(orb.MultiPoint, 0)

	// Build set of points from mp2
	mp2Set := make(map[orb.Point]bool)
	for _, pt := range mp2 {
		key := roundPoint(pt)
		mp2Set[key] = true
	}

	// Find common points
	for _, pt := range mp1 {
		key := roundPoint(pt)
		if mp2Set[key] && !seen[key] {
			result = append(result, pt)
			seen[key] = true
		}
	}

	return result
}

// MultiPointDifference computes mp1 - mp2.
func MultiPointDifference(mp1, mp2 orb.MultiPoint) orb.MultiPoint {
	// Build set of points to exclude
	exclude := make(map[orb.Point]bool)
	for _, pt := range mp2 {
		key := roundPoint(pt)
		exclude[key] = true
	}

	result := make(orb.MultiPoint, 0)
	for _, pt := range mp1 {
		key := roundPoint(pt)
		if !exclude[key] {
			result = append(result, pt)
		}
	}

	return result
}

// PointInMultiPoint checks if a point is in a multi-point.
func PointInMultiPoint(p orb.Point, mp orb.MultiPoint) bool {
	for _, pt := range mp {
		if PointsEqual(p, pt, Epsilon) {
			return true
		}
	}
	return false
}

