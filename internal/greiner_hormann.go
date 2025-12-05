package internal

import (
	"math"

	"github.com/paulmach/orb"
)

// ClipOperation represents the type of clipping operation.
type ClipOperation int

const (
	ClipIntersection ClipOperation = iota
	ClipUnion
	ClipDifference
	ClipXor
)

// MartinezRuedaClip performs polygon clipping using our custom Vatti-based implementation.
// This is a robust implementation for polygon boolean operations.
func MartinezRuedaClip(subject, clip orb.Polygon, op ClipOperation) orb.MultiPolygon {
	if len(subject) == 0 || len(subject[0]) < 3 {
		if op == ClipUnion {
			return orb.MultiPolygon{clip}
		}
		return orb.MultiPolygon{}
	}
	if len(clip) == 0 || len(clip[0]) < 3 {
		if op == ClipUnion || op == ClipDifference {
			return orb.MultiPolygon{subject}
		}
		return orb.MultiPolygon{}
	}

	// Convert ClipOperation to VattiClipOperation
	var vattiOp VattiClipOperation
	switch op {
	case ClipIntersection:
		vattiOp = VattiIntersection
	case ClipUnion:
		vattiOp = VattiUnion
	case ClipDifference:
		vattiOp = VattiDifference
	case ClipXor:
		vattiOp = VattiXor
	default:
		vattiOp = VattiIntersection
	}

	// Use our custom Vatti implementation
	return VattiClip(subject, clip, vattiOp)
}

// Legacy types and functions for compatibility with other code in this package

// Vertex represents a vertex in the polygon clipping algorithm.
type Vertex struct {
	Point       orb.Point
	Next        *Vertex
	Prev        *Vertex
	Neighbor    *Vertex
	Alpha       float64
	Intersect   bool
	Entry       bool
	Visited     bool
	Processed   bool
}

// PolygonList represents a polygon as a circular doubly-linked list.
type PolygonList struct {
	First *Vertex
}

// Edge represents a clipping edge for Sutherland-Hodgman.
type Edge struct {
	P1, P2 orb.Point
}

// IsInside checks if a point is on the "inside" side of the edge.
func (e Edge) IsInside(p orb.Point) bool {
	cross := (e.P2[0]-e.P1[0])*(p[1]-e.P1[1]) - (e.P2[1]-e.P1[1])*(p[0]-e.P1[0])
	return cross >= -Epsilon
}

// Intersect finds the intersection of a line segment with this edge.
func (e Edge) Intersect(p1, p2 orb.Point) (orb.Point, bool) {
	d1x := p2[0] - p1[0]
	d1y := p2[1] - p1[1]
	d2x := e.P2[0] - e.P1[0]
	d2y := e.P2[1] - e.P1[1]

	cross := d1x*d2y - d1y*d2x
	if math.Abs(cross) < Epsilon {
		return orb.Point{}, false
	}

	dx := e.P1[0] - p1[0]
	dy := e.P1[1] - p1[1]

	t := (dx*d2y - dy*d2x) / cross

	pt := orb.Point{
		p1[0] + t*d1x,
		p1[1] + t*d1y,
	}
	return pt, true
}

// SutherlandHodgman implements Sutherland-Hodgman polygon clipping algorithm.
func SutherlandHodgman(subject orb.Ring, clipEdges []Edge) orb.Ring {
	output := make(orb.Ring, len(subject))
	copy(output, subject)

	for _, edge := range clipEdges {
		if len(output) == 0 {
			return nil
		}
		input := output
		output = nil

		for i := 0; i < len(input); i++ {
			current := input[i]
			next := input[(i+1)%len(input)]

			currentInside := edge.IsInside(current)
			nextInside := edge.IsInside(next)

			if currentInside {
				output = appendUnique(output, current)
				if !nextInside {
					if pt, ok := edge.Intersect(current, next); ok {
						output = appendUnique(output, pt)
					}
				}
			} else if nextInside {
				if pt, ok := edge.Intersect(current, next); ok {
					output = appendUnique(output, pt)
				}
			}
		}
	}

	return output
}

// appendUnique appends a point to a ring only if it's not a duplicate of the last point.
func appendUnique(ring orb.Ring, pt orb.Point) orb.Ring {
	if len(ring) > 0 {
		last := ring[len(ring)-1]
		if math.Abs(last[0]-pt[0]) < Epsilon && math.Abs(last[1]-pt[1]) < Epsilon {
			return ring
		}
	}
	return append(ring, pt)
}

// pointInRingWinding uses winding number to check if point is inside ring.
func pointInRingWinding(p orb.Point, ring orb.Ring) bool {
	if len(ring) < 3 {
		return false
	}

	winding := 0
	n := len(ring)

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		if ring[i][1] <= p[1] {
			if ring[j][1] > p[1] {
				if isLeft(ring[i], ring[j], p) > 0 {
					winding++
				}
			}
		} else {
			if ring[j][1] <= p[1] {
				if isLeft(ring[i], ring[j], p) < 0 {
					winding--
				}
			}
		}
	}

	return winding != 0
}

// isLeft returns >0 if point is left of line, <0 if right, 0 if on line.
func isLeft(p0, p1, p2 orb.Point) float64 {
	return (p1[0]-p0[0])*(p2[1]-p0[1]) - (p2[0]-p0[0])*(p1[1]-p0[1])
}

// reverseRing returns a reversed copy of the ring.
func reverseRing(ring orb.Ring) orb.Ring {
	n := len(ring)
	result := make(orb.Ring, n)
	for i := 0; i < n; i++ {
		result[i] = ring[n-1-i]
	}
	return result
}

// ClipPolygons performs polygon clipping - legacy interface.
func ClipPolygons(subject, clip orb.Polygon, op ClipOperation) orb.MultiPolygon {
	return MartinezRuedaClip(subject, clip, op)
}

// WeilerAtherton is now just a wrapper around MartinezRuedaClip.
func WeilerAtherton(subject, clip orb.Polygon, op ClipOperation) orb.MultiPolygon {
	return MartinezRuedaClip(subject, clip, op)
}
