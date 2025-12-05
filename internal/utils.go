package internal

import (
	"github.com/paulmach/orb"
)

// PointInPolygon checks if a point is inside a polygon using ray casting algorithm.
// Returns true if the point is inside, false if outside.
// Points on the boundary are considered inside.
func PointInPolygon(p orb.Point, poly orb.Polygon) bool {
	if len(poly) == 0 {
		return false
	}

	// Check outer ring
	if !pointInRing(p, orb.Ring(poly[0])) {
		return false
	}

	// Check holes - point must not be in any hole
	for i := 1; i < len(poly); i++ {
		if pointInRing(p, orb.Ring(poly[i])) {
			return false
		}
	}

	return true
}

// pointInRing checks if a point is inside a ring using ray casting.
func pointInRing(p orb.Point, ring orb.Ring) bool {
	if len(ring) < 3 {
		return false
	}

	inside := false
	j := len(ring) - 1

	for i := 0; i < len(ring); i++ {
		pi := ring[i]
		pj := ring[j]

		if ((pi[1] > p[1]) != (pj[1] > p[1])) &&
			(p[0] < (pj[0]-pi[0])*(p[1]-pi[1])/(pj[1]-pi[1])+pi[0]) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// LineSegmentIntersection finds the intersection point of two line segments.
// Returns the intersection point and true if segments intersect, false otherwise.
func LineSegmentIntersection(p1, p2, p3, p4 orb.Point) (orb.Point, bool) {
	// Check for shared endpoints first
	if PointsEqual(p1, p3, Epsilon) || PointsEqual(p1, p4, Epsilon) {
		return p1, true
	}
	if PointsEqual(p2, p3, Epsilon) || PointsEqual(p2, p4, Epsilon) {
		return p2, true
	}

	x1, y1 := p1[0], p1[1]
	x2, y2 := p2[0], p2[1]
	x3, y3 := p3[0], p3[1]
	x4, y4 := p4[0], p4[1]

	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if abs(denom) < Epsilon {
		return orb.Point{}, false // Parallel lines
	}

	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / denom
	u := -((x1-x2)*(y1-y3) - (y1-y2)*(x1-x3)) / denom

	// Use epsilon tolerance for endpoint detection
	if t >= -Epsilon && t <= 1+Epsilon && u >= -Epsilon && u <= 1+Epsilon {
		// Clamp t to [0, 1] for the intersection point
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		// Segments intersect
		x := x1 + t*(x2-x1)
		y := y1 + t*(y2-y1)
		return orb.Point{x, y}, true
	}

	return orb.Point{}, false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// PolygonOrientation determines if a polygon ring is clockwise (true) or counter-clockwise (false).
func PolygonOrientation(ring orb.Ring) bool {
	if len(ring) < 3 {
		return false
	}

	area := 0.0
	for i := 0; i < len(ring)-1; i++ {
		area += (ring[i+1][0] - ring[i][0]) * (ring[i+1][1] + ring[i][1])
	}

	// Positive area means clockwise, negative means counter-clockwise
	return area > 0
}

// NormalizeRing ensures a ring is counter-clockwise (outer) or clockwise (hole).
// For outer rings, makes it counter-clockwise.
// For holes, makes it clockwise.
func NormalizeRing(ring orb.Ring, isHole bool) orb.Ring {
	if len(ring) < 3 {
		return ring
	}

	clockwise := PolygonOrientation(ring)
	shouldBeClockwise := isHole

	if clockwise != shouldBeClockwise {
		// Reverse the ring
		reversed := make(orb.Ring, len(ring))
		for i := 0; i < len(ring); i++ {
			reversed[i] = ring[len(ring)-1-i]
		}
		return reversed
	}

	return ring
}

// PointsEqual checks if two points are equal within epsilon tolerance.
func PointsEqual(p1, p2 orb.Point, epsilon float64) bool {
	dx := p1[0] - p2[0]
	dy := p1[1] - p2[1]
	return dx*dx+dy*dy < epsilon*epsilon
}

// PointOnSegment checks if a point lies on a line segment.
func PointOnSegment(p, p1, p2 orb.Point, epsilon float64) bool {
	// Check if point is within bounding box of segment
	minX := p1[0]
	maxX := p2[0]
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY := p1[1]
	maxY := p2[1]
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	if p[0] < minX-epsilon || p[0] > maxX+epsilon ||
		p[1] < minY-epsilon || p[1] > maxY+epsilon {
		return false
	}

	// Check if point is collinear with segment
	cross := (p2[0]-p1[0])*(p[1]-p1[1]) - (p2[1]-p1[1])*(p[0]-p1[0])
	return cross*cross < epsilon*epsilon*((p2[0]-p1[0])*(p2[0]-p1[0])+(p2[1]-p1[1])*(p2[1]-p1[1]))
}

// Epsilon is the default tolerance for floating point comparisons.
const Epsilon = 1e-9

