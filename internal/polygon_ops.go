package internal

import (
	"math"

	"github.com/paulmach/orb"
)

// PolygonUnion computes the union of two polygons.
// Returns a single polygon (possibly with multiple outer rings represented as a single polygon).
func PolygonUnion(p1, p2 orb.Polygon) orb.Polygon {
	if len(p1) == 0 || (len(p1) > 0 && len(p1[0]) < 3) {
		return p2
	}
	if len(p2) == 0 || (len(p2) > 0 && len(p2[0]) < 3) {
		return p1
	}

	result := PolygonUnionMulti(p1, p2)
	if len(result) == 0 {
		return orb.Polygon{}
	}
	if len(result) == 1 {
		return result[0]
	}
	// Multiple polygons - combine all rings into one polygon
	combined := make(orb.Polygon, 0)
	for _, poly := range result {
		combined = append(combined, poly...)
	}
	return combined
}

// PolygonUnionMulti computes the union of two polygons, returning a MultiPolygon.
func PolygonUnionMulti(p1, p2 orb.Polygon) orb.MultiPolygon {
	if len(p1) == 0 || (len(p1) > 0 && len(p1[0]) < 3) {
		return orb.MultiPolygon{p2}
	}
	if len(p2) == 0 || (len(p2) > 0 && len(p2[0]) < 3) {
		return orb.MultiPolygon{p1}
	}

	return MartinezRuedaClip(p1, p2, ClipUnion)
}

// PolygonIntersection computes the intersection of two polygons.
func PolygonIntersection(p1, p2 orb.Polygon) orb.Polygon {
	if len(p1) == 0 || len(p2) == 0 {
		return orb.Polygon{}
	}
	if len(p1[0]) < 3 || len(p2[0]) < 3 {
		return orb.Polygon{}
	}

	result := PolygonIntersectionMulti(p1, p2)
	if len(result) == 0 {
		return orb.Polygon{}
	}
	if len(result) == 1 {
		return result[0]
	}
	// Multiple polygons - combine all rings into one polygon
	combined := make(orb.Polygon, 0)
	for _, poly := range result {
		combined = append(combined, poly...)
	}
	return combined
}

// PolygonIntersectionMulti computes the intersection of two polygons, returning a MultiPolygon.
func PolygonIntersectionMulti(p1, p2 orb.Polygon) orb.MultiPolygon {
	if len(p1) == 0 || len(p2) == 0 {
		return orb.MultiPolygon{}
	}
	if len(p1[0]) < 3 || len(p2[0]) < 3 {
		return orb.MultiPolygon{}
	}

	return MartinezRuedaClip(p1, p2, ClipIntersection)
}

// PolygonIntersectionGeom computes the intersection of two polygons.
// Returns the appropriate geometry type based on the intersection:
// - Polygon/MultiPolygon for area intersection
// - LineString/MultiLineString for shared boundary edges
// - Point/MultiPoint for shared boundary vertices
// - nil if no intersection
func PolygonIntersectionGeom(p1, p2 orb.Polygon) orb.Geometry {
	if len(p1) == 0 || len(p2) == 0 {
		return nil
	}
	if len(p1[0]) < 3 || len(p2[0]) < 3 {
		return nil
	}

	// First try polygon intersection
	result := MartinezRuedaClip(p1, p2, ClipIntersection)
	if len(result) > 0 {
		if len(result) == 1 {
			return result[0]
		}
		return result
	}

	// No area intersection - check for boundary intersection
	return findBoundaryIntersection(p1, p2)
}

// findBoundaryIntersection finds shared boundary elements between two polygons.
func findBoundaryIntersection(p1, p2 orb.Polygon) orb.Geometry {
	var sharedLines orb.MultiLineString
	var sharedPoints orb.MultiPoint

	// Check each edge of p1's rings against each edge of p2's rings
	for _, ring1 := range p1 {
		for i := 0; i < len(ring1)-1; i++ {
			seg1Start, seg1End := ring1[i], ring1[i+1]

			for _, ring2 := range p2 {
				for j := 0; j < len(ring2)-1; j++ {
					seg2Start, seg2End := ring2[j], ring2[j+1]

					// Check for collinear overlapping segments
					overlap := findCollinearOverlap(seg1Start, seg1End, seg2Start, seg2End)
					if len(overlap) >= 2 {
						sharedLines = append(sharedLines, overlap)
						continue
					}

					// Check for point intersection
					pt, ok := LineSegmentIntersection(seg1Start, seg1End, seg2Start, seg2End)
					if ok {
						// Add point if not already in list
						found := false
						for _, p := range sharedPoints {
							if PointsEqual(p, pt, Epsilon) {
								found = true
								break
							}
						}
						if !found {
							sharedPoints = append(sharedPoints, pt)
						}
					}
				}
			}
		}
	}

	// Merge overlapping/connected line segments
	sharedLines = mergeConnectedLines(sharedLines)

	// Return the most appropriate geometry type
	if len(sharedLines) > 0 {
		// Remove points that are on the lines
		var isolatedPoints orb.MultiPoint
		for _, pt := range sharedPoints {
			onLine := false
			for _, line := range sharedLines {
				if pointLiesOnLineString(pt, line) {
					onLine = true
					break
				}
			}
			if !onLine {
				isolatedPoints = append(isolatedPoints, pt)
			}
		}

		if len(isolatedPoints) > 0 {
			// Return collection of lines and points
			var coll orb.Collection
			for _, line := range sharedLines {
				coll = append(coll, line)
			}
			for _, pt := range isolatedPoints {
				coll = append(coll, pt)
			}
			return coll
		}

		if len(sharedLines) == 1 {
			return sharedLines[0]
		}
		return sharedLines
	}

	if len(sharedPoints) > 0 {
		if len(sharedPoints) == 1 {
			return sharedPoints[0]
		}
		return sharedPoints
	}

	return nil
}

// pointLiesOnLineString checks if a point is on a line string.
func pointLiesOnLineString(pt orb.Point, ls orb.LineString) bool {
	for i := 0; i < len(ls)-1; i++ {
		if PointOnSegment(pt, ls[i], ls[i+1], Epsilon) {
			return true
		}
	}
	return false
}

// findCollinearOverlap finds the overlapping portion of two collinear segments.
func findCollinearOverlap(p1, p2, p3, p4 orb.Point) orb.LineString {
	// Check if segments are collinear
	cross1 := (p2[0]-p1[0])*(p3[1]-p1[1]) - (p2[1]-p1[1])*(p3[0]-p1[0])
	cross2 := (p2[0]-p1[0])*(p4[1]-p1[1]) - (p2[1]-p1[1])*(p4[0]-p1[0])

	if math.Abs(cross1) > Epsilon || math.Abs(cross2) > Epsilon {
		return nil // Not collinear
	}

	// Project points onto the line and find overlap
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	length := math.Sqrt(dx*dx + dy*dy)
	if length < Epsilon {
		return nil
	}

	// Parametric positions on segment p1-p2
	t1 := 0.0
	t2 := 1.0

	// Project p3 and p4 onto segment p1-p2
	t3 := ((p3[0]-p1[0])*dx + (p3[1]-p1[1])*dy) / (length * length)
	t4 := ((p4[0]-p1[0])*dx + (p4[1]-p1[1])*dy) / (length * length)

	// Ensure t3 < t4
	if t3 > t4 {
		t3, t4 = t4, t3
	}

	// Find overlap
	overlapStart := math.Max(math.Min(t1, t2), math.Min(t3, t4))
	overlapEnd := math.Min(math.Max(t1, t2), math.Max(t3, t4))

	// Check if there's a valid overlap
	if overlapEnd-overlapStart < Epsilon {
		return nil
	}

	// Convert back to points
	startPt := orb.Point{
		p1[0] + overlapStart*dx,
		p1[1] + overlapStart*dy,
	}
	endPt := orb.Point{
		p1[0] + overlapEnd*dx,
		p1[1] + overlapEnd*dy,
	}

	return orb.LineString{startPt, endPt}
}

// mergeConnectedLines merges connected line segments.
func mergeConnectedLines(lines orb.MultiLineString) orb.MultiLineString {
	if len(lines) <= 1 {
		return lines
	}

	merged := true
	for merged {
		merged = false
		var newLines orb.MultiLineString

		for i, line := range lines {
			if len(line) == 0 {
				continue
			}

			// Try to merge with other lines
			mergedLine := line
			for j := i + 1; j < len(lines); j++ {
				if len(lines[j]) == 0 {
					continue
				}

				// Check if lines connect
				if PointsEqual(mergedLine[len(mergedLine)-1], lines[j][0], Epsilon) {
					// Connect end of mergedLine to start of lines[j]
					for k := 1; k < len(lines[j]); k++ {
						mergedLine = append(mergedLine, lines[j][k])
					}
					lines[j] = nil
					merged = true
				} else if PointsEqual(mergedLine[0], lines[j][len(lines[j])-1], Epsilon) {
					// Connect end of lines[j] to start of mergedLine
					newLine := make(orb.LineString, len(lines[j])+len(mergedLine)-1)
					copy(newLine, lines[j])
					copy(newLine[len(lines[j]):], mergedLine[1:])
					mergedLine = newLine
					lines[j] = nil
					merged = true
				} else if PointsEqual(mergedLine[len(mergedLine)-1], lines[j][len(lines[j])-1], Epsilon) {
					// Reverse lines[j] and connect
					for k := len(lines[j]) - 2; k >= 0; k-- {
						mergedLine = append(mergedLine, lines[j][k])
					}
					lines[j] = nil
					merged = true
				} else if PointsEqual(mergedLine[0], lines[j][0], Epsilon) {
					// Reverse mergedLine and connect
					newLine := make(orb.LineString, 0, len(mergedLine)+len(lines[j])-1)
					for k := len(mergedLine) - 1; k >= 0; k-- {
						newLine = append(newLine, mergedLine[k])
					}
					for k := 1; k < len(lines[j]); k++ {
						newLine = append(newLine, lines[j][k])
					}
					mergedLine = newLine
					lines[j] = nil
					merged = true
				}
			}

			newLines = append(newLines, mergedLine)
		}

		lines = newLines
	}

	// Filter out empty lines
	var result orb.MultiLineString
	for _, line := range lines {
		if len(line) >= 2 {
			result = append(result, line)
		}
	}

	return result
}

// PolygonDifference computes p1 - p2 (p1 minus p2).
func PolygonDifference(p1, p2 orb.Polygon) orb.Polygon {
	if len(p1) == 0 || (len(p1) > 0 && len(p1[0]) < 3) {
		return orb.Polygon{}
	}
	if len(p2) == 0 || (len(p2) > 0 && len(p2[0]) < 3) {
		return p1
	}

	result := PolygonDifferenceMulti(p1, p2)
	if len(result) == 0 {
		return orb.Polygon{}
	}
	if len(result) == 1 {
		return result[0]
	}
	// Multiple polygons - combine all rings into one polygon
	combined := make(orb.Polygon, 0)
	for _, poly := range result {
		combined = append(combined, poly...)
	}
	return combined
}

// PolygonDifferenceMulti computes p1 - p2, returning a MultiPolygon.
func PolygonDifferenceMulti(p1, p2 orb.Polygon) orb.MultiPolygon {
	if len(p1) == 0 || (len(p1) > 0 && len(p1[0]) < 3) {
		return orb.MultiPolygon{}
	}
	if len(p2) == 0 || (len(p2) > 0 && len(p2[0]) < 3) {
		return orb.MultiPolygon{p1}
	}

	return MartinezRuedaClip(p1, p2, ClipDifference)
}

// PolygonSymmetricDifference computes the symmetric difference of two polygons.
func PolygonSymmetricDifference(p1, p2 orb.Polygon) orb.MultiPolygon {
	if len(p1) == 0 || (len(p1) > 0 && len(p1[0]) < 3) {
		return orb.MultiPolygon{p2}
	}
	if len(p2) == 0 || (len(p2) > 0 && len(p2[0]) < 3) {
		return orb.MultiPolygon{p1}
	}

	return MartinezRuedaClip(p1, p2, ClipXor)
}

// polygonsOverlap checks if two polygons overlap.
func polygonsOverlap(p1, p2 orb.Polygon) bool {
	b1 := p1.Bound()
	b2 := p2.Bound()

	// Quick bounding box check
	if b1.Max[0] < b2.Min[0] || b2.Max[0] < b1.Min[0] ||
		b1.Max[1] < b2.Min[1] || b2.Max[1] < b1.Min[1] {
		return false
	}

	// Check if any vertex of p1 is in p2 or vice versa
	for _, ring := range p1 {
		for _, pt := range ring {
			if PointInPolygon(pt, p2) {
				return true
			}
		}
	}

	for _, ring := range p2 {
		for _, pt := range ring {
			if PointInPolygon(pt, p1) {
				return true
			}
		}
	}

	// Check for edge intersections
	return edgesIntersect(p1, p2)
}

// edgesIntersect checks if any edges of the polygons intersect.
func edgesIntersect(p1, p2 orb.Polygon) bool {
	for _, ring1 := range p1 {
		for i := 0; i < len(ring1)-1; i++ {
			for _, ring2 := range p2 {
				for j := 0; j < len(ring2)-1; j++ {
					_, intersects := LineSegmentIntersection(ring1[i], ring1[i+1], ring2[j], ring2[j+1])
					if intersects {
						return true
					}
				}
			}
		}
	}
	return false
}

// polygonInPolygon checks if polygon p1 is completely inside polygon p2.
func polygonInPolygon(p1, p2 orb.Polygon) bool {
	if len(p1) == 0 || len(p2) == 0 {
		return false
	}

	// Check if all vertices of p1's outer ring are inside p2
	outerRing := p1[0]
	for _, pt := range outerRing {
		if !PointInPolygon(pt, p2) {
			return false
		}
	}

	return true
}

// combinePolygons combines two non-overlapping polygons.
func combinePolygons(p1, p2 orb.Polygon) orb.Polygon {
	result := make(orb.Polygon, 0, len(p1)+len(p2))
	result = append(result, p1...)
	result = append(result, p2...)
	return result
}

// addHole adds a hole to a polygon.
func addHole(poly orb.Polygon, hole orb.Ring) orb.Polygon {
	result := make(orb.Polygon, len(poly), len(poly)+1)
	copy(result, poly)
	result = append(result, hole)
	return result
}
