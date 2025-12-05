package internal

import (
	"math"
	"sort"

	"github.com/paulmach/orb"
)

// LineStringUnion computes the union of two line strings.
// Returns all segments from both lines, split at intersection points.
// Each unique portion of the combined geometry appears exactly once.
func LineStringUnion(ls1, ls2 orb.LineString) orb.MultiLineString {
	// Clean up line strings by removing consecutive duplicate points
	ls1 = removeConsecutiveDuplicates(ls1)
	ls2 = removeConsecutiveDuplicates(ls2)

	if len(ls1) < 2 {
		if len(ls2) < 2 {
			return orb.MultiLineString{}
		}
		return orb.MultiLineString{ls2}
	}
	if len(ls2) < 2 {
		return orb.MultiLineString{ls1}
	}

	// Check if lines are identical
	if lineStringsEqual(ls1, ls2) {
		return orb.MultiLineString{ls1}
	}

	// Find all intersection/split points from both lines
	splitPoints := findAllSplitPoints(ls1, ls2)

	// Split both lines at all split points
	segments1 := splitLineAtPoints(ls1, splitPoints)
	segments2 := splitLineAtPoints(ls2, splitPoints)

	// For union, we want:
	// 1. All segments from ls1 (some may overlap with ls2)
	// 2. Segments from ls2 that are NOT on ls1
	var result orb.MultiLineString

	// Add all segments from ls1
	result = append(result, segments1...)

	// Add segments from ls2 that are not duplicates of ls1 segments
	for _, seg2 := range segments2 {
		isDuplicate := false
		for _, seg1 := range segments1 {
			if segmentsAreSamePath(seg1, seg2) {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			result = append(result, seg2)
		}
	}

	// Merge consecutive segments from the same source (ls2's non-overlapping portions)
	return mergeNonOverlappingSegments(result, ls1, ls2)
}

// mergeNonOverlappingSegments merges consecutive segments that were originally
// consecutive edges in their source line AND are not on the other line.
// A connection point must be a vertex in the original source line to allow merging.
func mergeNonOverlappingSegments(segments orb.MultiLineString, ls1, ls2 orb.LineString) orb.MultiLineString {
	if len(segments) <= 1 {
		return segments
	}

	// Build set of original vertices from both lines
	originalVertices := make(map[orb.Point]bool)
	for _, pt := range ls1 {
		originalVertices[roundPoint(pt)] = true
	}
	for _, pt := range ls2 {
		originalVertices[roundPoint(pt)] = true
	}

	// Categorize each segment: is it on ls1 only, ls2 only, or both?
	type segmentInfo struct {
		seg    orb.LineString
		onLs1  bool
		onLs2  bool
		merged bool
	}

	infos := make([]segmentInfo, len(segments))
	for i, seg := range segments {
		if len(seg) < 2 {
			continue
		}
		mid := orb.Point{(seg[0][0] + seg[len(seg)-1][0]) / 2, (seg[0][1] + seg[len(seg)-1][1]) / 2}
		infos[i] = segmentInfo{
			seg:   seg,
			onLs1: pointOnLineString(mid, ls1),
			onLs2: pointOnLineString(mid, ls2),
		}
	}

	// isOriginalVertex checks if a point was a vertex in the original lines
	isOriginalVertex := func(pt orb.Point) bool {
		return originalVertices[roundPoint(pt)]
	}

	// canMergeAt checks if two segments can be merged at a shared point.
	// They can only merge if the point is an original vertex AND both segments
	// have the same overlap category AND the category is "on one line only".
	canMergeAt := func(pt orb.Point, info1, info2 segmentInfo) bool {
		// Must share the same category
		if info1.onLs1 != info2.onLs1 || info1.onLs2 != info2.onLs2 {
			return false
		}
		// The connection point must be an original vertex
		if !isOriginalVertex(pt) {
			return false
		}
		// Only merge segments that are NOT on the other line (non-overlapping portions)
		// If a segment is on both lines (shared), don't merge it
		if info1.onLs1 && info1.onLs2 {
			return false
		}
		return true
	}

	// Merge consecutive segments
	var result orb.MultiLineString

	for i := 0; i < len(infos); i++ {
		if infos[i].merged || len(infos[i].seg) < 2 {
			continue
		}
		infos[i].merged = true

		chain := make(orb.LineString, len(infos[i].seg))
		copy(chain, infos[i].seg)
		chainInfo := infos[i]

		// Try to extend by finding matching segments
		changed := true
		for changed {
			changed = false
			for j := 0; j < len(infos); j++ {
				if infos[j].merged || len(infos[j].seg) < 2 {
					continue
				}

				seg := infos[j].seg
				// Check if seg connects to chain's end
				if PointsEqual(seg[0], chain[len(chain)-1], Epsilon) && canMergeAt(seg[0], chainInfo, infos[j]) {
					chain = append(chain, seg[1:]...)
					infos[j].merged = true
					changed = true
				} else if PointsEqual(seg[len(seg)-1], chain[len(chain)-1], Epsilon) && canMergeAt(seg[len(seg)-1], chainInfo, infos[j]) {
					// Append reversed
					for k := len(seg) - 2; k >= 0; k-- {
						chain = append(chain, seg[k])
					}
					infos[j].merged = true
					changed = true
				} else if PointsEqual(seg[len(seg)-1], chain[0], Epsilon) && canMergeAt(seg[len(seg)-1], chainInfo, infos[j]) {
					// Prepend
					newChain := make(orb.LineString, len(seg)+len(chain)-1)
					copy(newChain, seg)
					copy(newChain[len(seg):], chain[1:])
					chain = newChain
					infos[j].merged = true
					changed = true
				} else if PointsEqual(seg[0], chain[0], Epsilon) && canMergeAt(seg[0], chainInfo, infos[j]) {
					// Prepend reversed
					newChain := make(orb.LineString, len(seg)+len(chain)-1)
					for k := 0; k < len(seg); k++ {
						newChain[k] = seg[len(seg)-1-k]
					}
					copy(newChain[len(seg):], chain[1:])
					chain = newChain
					infos[j].merged = true
					changed = true
				}
			}
		}

		result = append(result, chain)
	}

	return result
}

// findAllSplitPoints finds all points where two line strings should be split.
// This includes: intersection points, endpoints touching the other line,
// and vertices of one line that lie on the other.
func findAllSplitPoints(ls1, ls2 orb.LineString) []orb.Point {
	var splitPoints []orb.Point
	seen := make(map[orb.Point]bool)

	addPoint := func(pt orb.Point) {
		key := roundPoint(pt)
		if !seen[key] {
			seen[key] = true
			splitPoints = append(splitPoints, pt)
		}
	}

	// Find segment-segment intersections (crossing points)
	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			pt, alpha1, alpha2, ok := segmentIntersectionFull(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
			if ok {
				// Add if it's an interior crossing for either segment
				isInterior1 := alpha1 > Epsilon && alpha1 < 1-Epsilon
				isInterior2 := alpha2 > Epsilon && alpha2 < 1-Epsilon
				if isInterior1 || isInterior2 {
					addPoint(pt)
				}
			}
		}
	}

	// Add all vertices of ls1 that lie on ls2 (interior of any segment)
	for _, pt := range ls1 {
		for j := 0; j < len(ls2)-1; j++ {
			if PointOnSegment(pt, ls2[j], ls2[j+1], Epsilon) {
				addPoint(pt)
				break
			}
		}
	}

	// Add all vertices of ls2 that lie on ls1 (interior of any segment)
	for _, pt := range ls2 {
		for i := 0; i < len(ls1)-1; i++ {
			if PointOnSegment(pt, ls1[i], ls1[i+1], Epsilon) {
				addPoint(pt)
				break
			}
		}
	}

	// Find overlap boundaries - endpoints of collinear overlapping segments
	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			if segmentsCollinear(ls1[i], ls1[i+1], ls2[j], ls2[j+1]) {
				overlap := findSegmentOverlap(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
				if len(overlap) >= 2 {
					addPoint(overlap[0])
					addPoint(overlap[len(overlap)-1])
				}
			}
		}
	}

	return splitPoints
}

// linesCollinear checks if all points of both lines are collinear.
func linesCollinear(ls1, ls2 orb.LineString) bool {
	if len(ls1) < 2 || len(ls2) < 2 {
		return false
	}

	// Use the first segment of ls1 to define the line
	p1, p2 := ls1[0], ls1[len(ls1)-1]
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	length := math.Sqrt(dx*dx + dy*dy)
	if length < Epsilon {
		return false
	}

	// Check if all points of ls2 are on the line defined by ls1
	for _, pt := range ls2 {
		cross := (pt[0]-p1[0])*dy - (pt[1]-p1[1])*dx
		if math.Abs(cross) > Epsilon*length {
			return false
		}
	}

	return true
}

// unionCollinearLines computes the union of two collinear line strings.
func unionCollinearLines(ls1, ls2 orb.LineString) orb.MultiLineString {
	// Project points onto a common line to find ranges
	p1, p2 := ls1[0], ls1[len(ls1)-1]
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	length := math.Sqrt(dx*dx + dy*dy)
	if length < Epsilon {
		return orb.MultiLineString{ls1}
	}

	// Get parametric ranges for both lines
	t1Start := 0.0
	t1End := 1.0

	t2Start := ((ls2[0][0]-p1[0])*dx + (ls2[0][1]-p1[1])*dy) / (length * length)
	t2End := ((ls2[len(ls2)-1][0]-p1[0])*dx + (ls2[len(ls2)-1][1]-p1[1])*dy) / (length * length)

	if t2Start > t2End {
		t2Start, t2End = t2End, t2Start
	}

	// Check if they overlap
	overlapStart := math.Max(t1Start, t2Start)
	overlapEnd := math.Min(t1End, t2End)

	if overlapEnd <= overlapStart+Epsilon {
		// No overlap - return both as separate segments
		return orb.MultiLineString{ls1, ls2}
	}

	// There is overlap - merge into a single continuous line
	// Collect all unique projected positions
	type projPoint struct {
		t  float64
		pt orb.Point
	}
	var projPoints []projPoint

	// Add ls1 points
	for _, pt := range ls1 {
		t := ((pt[0]-p1[0])*dx + (pt[1]-p1[1])*dy) / (length * length)
		projPoints = append(projPoints, projPoint{t, pt})
	}

	// Add ls2 points (only unique ones)
	for _, pt := range ls2 {
		t := ((pt[0]-p1[0])*dx + (pt[1]-p1[1])*dy) / (length * length)
		found := false
		for _, pp := range projPoints {
			if math.Abs(pp.t-t) < Epsilon {
				found = true
				break
			}
		}
		if !found {
			projPoints = append(projPoints, projPoint{t, pt})
		}
	}

	// Sort by parametric position
	sort.Slice(projPoints, func(i, j int) bool {
		return projPoints[i].t < projPoints[j].t
	})

	// Build the line string from sorted points
	result := make(orb.LineString, len(projPoints))
	for i, pp := range projPoints {
		result[i] = pp.pt
	}

	// Split into segments
	var segments orb.MultiLineString
	for i := 0; i < len(result)-1; i++ {
		seg := orb.LineString{result[i], result[i+1]}
		segments = append(segments, seg)
	}

	return segments
}

// LineStringIntersectionGeom computes the intersection of two line strings.
// Returns the overlapping portions as LineStrings, crossing points, shared endpoints,
// or a Collection if there are mixed geometry types.
func LineStringIntersectionGeom(ls1, ls2 orb.LineString) orb.Geometry {
	// Clean up line strings by removing consecutive duplicate points
	ls1 = removeConsecutiveDuplicates(ls1)
	ls2 = removeConsecutiveDuplicates(ls2)

	if len(ls1) < 2 || len(ls2) < 2 {
		return nil
	}

	// Check if lines are identical
	if lineStringsEqual(ls1, ls2) {
		return ls1
	}

	// Find overlapping segments
	overlaps := findOverlappingSegments(ls1, ls2)

	// Find all intersection points including crossing points and shared vertices
	var intersectionPoints orb.MultiPoint
	seen := make(map[orb.Point]bool)

	// Find crossing points (non-collinear intersections)
	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			// Skip collinear segments - they contribute to overlaps, not points
			if segmentsCollinear(ls1[i], ls1[i+1], ls2[j], ls2[j+1]) {
				continue
			}
			pt, _, _, ok := segmentIntersectionFull(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
			if ok {
				key := roundPoint(pt)
				if !seen[key] && !pointOnAnyOverlap(pt, overlaps) {
					seen[key] = true
					intersectionPoints = append(intersectionPoints, pt)
				}
			}
		}
	}

	// Check for shared endpoints (touching lines)
	for _, p1 := range []orb.Point{ls1[0], ls1[len(ls1)-1]} {
		for _, p2 := range []orb.Point{ls2[0], ls2[len(ls2)-1]} {
			if PointsEqual(p1, p2, Epsilon) {
				key := roundPoint(p1)
				if !seen[key] && !pointOnAnyOverlap(p1, overlaps) {
					seen[key] = true
					intersectionPoints = append(intersectionPoints, p1)
				}
			}
		}
	}

	// Build result based on what we found
	hasOverlaps := len(overlaps) > 0
	hasPoints := len(intersectionPoints) > 0

	if hasOverlaps && hasPoints {
		// Return Collection with both line segments and points
		var coll orb.Collection
		for _, seg := range overlaps {
			coll = append(coll, seg)
		}
		for _, pt := range intersectionPoints {
			coll = append(coll, pt)
		}
		return coll
	}

	if hasOverlaps {
		if len(overlaps) == 1 {
			return overlaps[0]
		}
		return overlaps
	}

	if hasPoints {
		if len(intersectionPoints) == 1 {
			return intersectionPoints[0]
		}
		return intersectionPoints
	}

	return nil
}

// pointOnAnyOverlap checks if a point lies on any of the overlap segments.
func pointOnAnyOverlap(pt orb.Point, overlaps orb.MultiLineString) bool {
	for _, overlap := range overlaps {
		if pointOnLineString(pt, overlap) {
			return true
		}
	}
	return false
}

// LineStringIntersection computes the intersection points of two line strings.
// Returns points where the line strings intersect (for backward compatibility).
func LineStringIntersection(ls1, ls2 orb.LineString) orb.MultiPoint {
	if len(ls1) < 2 || len(ls2) < 2 {
		return orb.MultiPoint{}
	}

	return findLineIntersectionPoints(ls1, ls2)
}

// LineStringDifference computes ls1 - ls2.
// Returns parts of ls1 that don't overlap with ls2.
func LineStringDifference(ls1, ls2 orb.LineString) orb.MultiLineString {
	// Clean up line strings by removing consecutive duplicate points
	ls1 = removeConsecutiveDuplicates(ls1)
	ls2 = removeConsecutiveDuplicates(ls2)

	if len(ls1) < 2 {
		return orb.MultiLineString{}
	}
	if len(ls2) < 2 {
		return orb.MultiLineString{ls1}
	}

	// Check if lines are identical
	if lineStringsEqual(ls1, ls2) {
		return orb.MultiLineString{}
	}

	// Find all split points
	splitPoints := findAllSplitPoints(ls1, ls2)

	// If no split points, check if ls1 is entirely on or off ls2
	if len(splitPoints) == 0 {
		// Check if ls1 is contained in ls2
		if lineStringContained(ls1, ls2) {
			return orb.MultiLineString{}
		}
		return orb.MultiLineString{ls1}
	}

	// Split ls1 at all split points
	segments := splitLineAtPoints(ls1, splitPoints)

	// Filter out segments that are on ls2
	var result orb.MultiLineString
	for _, seg := range segments {
		if len(seg) < 2 {
			continue
		}
		// Check if the segment is on ls2 by checking its midpoint
		mid := orb.Point{
			(seg[0][0] + seg[len(seg)-1][0]) / 2,
			(seg[0][1] + seg[len(seg)-1][1]) / 2,
		}
		if !pointOnLineString(mid, ls2) {
			result = append(result, seg)
		}
	}

	// Merge consecutive segments, but only at original vertices of ls1
	return mergeAtOriginalVertices(result, ls1)
}

// mergeAtOriginalVertices merges consecutive segments but only if they connect
// at a vertex that was in the original line.
func mergeAtOriginalVertices(segments orb.MultiLineString, originalLine orb.LineString) orb.MultiLineString {
	if len(segments) <= 1 {
		return segments
	}

	// Build set of original vertices
	originalVertices := make(map[orb.Point]bool)
	for _, pt := range originalLine {
		originalVertices[roundPoint(pt)] = true
	}

	isOriginalVertex := func(pt orb.Point) bool {
		return originalVertices[roundPoint(pt)]
	}

	merged := make([]bool, len(segments))
	var result orb.MultiLineString

	for i := 0; i < len(segments); i++ {
		if merged[i] || len(segments[i]) < 2 {
			continue
		}
		merged[i] = true

		chain := make(orb.LineString, len(segments[i]))
		copy(chain, segments[i])

		// Keep extending the chain, but only at original vertices
		changed := true
		for changed {
			changed = false
			for j := 0; j < len(segments); j++ {
				if merged[j] || len(segments[j]) < 2 {
					continue
				}

				seg := segments[j]
				endPt := chain[len(chain)-1]
				startPt := chain[0]

				// Check if seg connects to chain's end at an original vertex
				if PointsEqual(seg[0], endPt, Epsilon) && isOriginalVertex(endPt) {
					chain = append(chain, seg[1:]...)
					merged[j] = true
					changed = true
				} else if PointsEqual(seg[len(seg)-1], endPt, Epsilon) && isOriginalVertex(endPt) {
					// Append reversed
					for k := len(seg) - 2; k >= 0; k-- {
						chain = append(chain, seg[k])
					}
					merged[j] = true
					changed = true
				} else if PointsEqual(seg[len(seg)-1], startPt, Epsilon) && isOriginalVertex(startPt) {
					// Prepend
					newChain := make(orb.LineString, len(seg)+len(chain)-1)
					copy(newChain, seg)
					copy(newChain[len(seg):], chain[1:])
					chain = newChain
					merged[j] = true
					changed = true
				} else if PointsEqual(seg[0], startPt, Epsilon) && isOriginalVertex(startPt) {
					// Prepend reversed
					newChain := make(orb.LineString, len(seg)+len(chain)-1)
					for k := 0; k < len(seg); k++ {
						newChain[k] = seg[len(seg)-1-k]
					}
					copy(newChain[len(seg):], chain[1:])
					chain = newChain
					merged[j] = true
					changed = true
				}
			}
		}

		result = append(result, chain)
	}

	return result
}

// differenceCollinearLines computes the difference of two collinear lines.
func differenceCollinearLines(ls1, ls2 orb.LineString) orb.MultiLineString {
	// Project ls1 onto itself to get range [0, len1]
	p1, p2 := ls1[0], ls1[len(ls1)-1]
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	len1 := math.Sqrt(dx*dx + dy*dy)
	if len1 < Epsilon {
		return orb.MultiLineString{}
	}

	// Find ls2's range on the same line
	t2Start := ((ls2[0][0]-p1[0])*dx + (ls2[0][1]-p1[1])*dy) / (len1 * len1)
	t2End := ((ls2[len(ls2)-1][0]-p1[0])*dx + (ls2[len(ls2)-1][1]-p1[1])*dy) / (len1 * len1)

	// Ensure t2Start < t2End
	if t2Start > t2End {
		t2Start, t2End = t2End, t2Start
	}

	// ls1 range is [0, 1]
	// We want parts of [0, 1] that are not in [t2Start, t2End]
	var result orb.MultiLineString

	// Part before ls2 starts (if any)
	if t2Start > Epsilon {
		end := math.Min(t2Start, 1.0)
		if end > Epsilon {
			startPt := ls1[0]
			endPt := orb.Point{
				p1[0] + end*dx,
				p1[1] + end*dy,
			}
			result = append(result, orb.LineString{startPt, endPt})
		}
	}

	// Part after ls2 ends (if any)
	if t2End < 1-Epsilon {
		start := math.Max(t2End, 0.0)
		if start < 1-Epsilon {
			startPt := orb.Point{
				p1[0] + start*dx,
				p1[1] + start*dy,
			}
			endPt := ls1[len(ls1)-1]
			result = append(result, orb.LineString{startPt, endPt})
		}
	}

	// If ls2 doesn't overlap with ls1 at all, return entire ls1
	if (t2End < 0) || (t2Start > 1) {
		return orb.MultiLineString{ls1}
	}

	return result
}

// lineStringsEqual checks if two line strings are equal (same points in same order).
func lineStringsEqual(ls1, ls2 orb.LineString) bool {
	if len(ls1) != len(ls2) {
		return false
	}
	for i := range ls1 {
		if !PointsEqual(ls1[i], ls2[i], Epsilon) {
			return false
		}
	}
	return true
}

// findLineIntersections finds all intersection points between two line strings.
func findLineIntersections(ls1, ls2 orb.LineString) []orb.Point {
	var intersections []orb.Point
	seen := make(map[orb.Point]bool)

	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			pt, _, _, ok := segmentIntersectionFull(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
			if ok {
				key := roundPoint(pt)
				if !seen[key] {
					intersections = append(intersections, pt)
					seen[key] = true
				}
			}
		}
	}

	// Also add any endpoints that touch
	for _, p1 := range ls1 {
		for j := 0; j < len(ls2)-1; j++ {
			if PointOnSegment(p1, ls2[j], ls2[j+1], Epsilon) {
				key := roundPoint(p1)
				if !seen[key] {
					intersections = append(intersections, p1)
					seen[key] = true
				}
			}
		}
	}

	for _, p2 := range ls2 {
		for i := 0; i < len(ls1)-1; i++ {
			if PointOnSegment(p2, ls1[i], ls1[i+1], Epsilon) {
				key := roundPoint(p2)
				if !seen[key] {
					intersections = append(intersections, p2)
					seen[key] = true
				}
			}
		}
	}

	return intersections
}

// findLineIntersectionPoints finds intersection points (not overlapping segments).
func findLineIntersectionPoints(ls1, ls2 orb.LineString) orb.MultiPoint {
	var intersections orb.MultiPoint
	seen := make(map[orb.Point]bool)

	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			pt, _, _, ok := segmentIntersectionFull(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
			if ok {
				// Only add if not collinear (proper intersection)
				if !segmentsCollinear(ls1[i], ls1[i+1], ls2[j], ls2[j+1]) {
					key := roundPoint(pt)
					if !seen[key] {
						intersections = append(intersections, pt)
						seen[key] = true
					}
				}
			}
		}
	}

	// Also check for endpoint touching (when segments share a common endpoint)
	// Check ls1 endpoints against ls2
	for _, p1 := range []orb.Point{ls1[0], ls1[len(ls1)-1]} {
		for _, p2 := range []orb.Point{ls2[0], ls2[len(ls2)-1]} {
			if PointsEqual(p1, p2, Epsilon) {
				key := roundPoint(p1)
				if !seen[key] {
					intersections = append(intersections, p1)
					seen[key] = true
				}
			}
		}
	}

	// Check for endpoint of one line on the interior of another
	for _, p := range []orb.Point{ls1[0], ls1[len(ls1)-1]} {
		for j := 0; j < len(ls2)-1; j++ {
			if PointOnSegment(p, ls2[j], ls2[j+1], Epsilon) {
				key := roundPoint(p)
				if !seen[key] {
					intersections = append(intersections, p)
					seen[key] = true
				}
			}
		}
	}

	for _, p := range []orb.Point{ls2[0], ls2[len(ls2)-1]} {
		for i := 0; i < len(ls1)-1; i++ {
			if PointOnSegment(p, ls1[i], ls1[i+1], Epsilon) {
				key := roundPoint(p)
				if !seen[key] {
					intersections = append(intersections, p)
					seen[key] = true
				}
			}
		}
	}

	return intersections
}

// findOverlappingSegments finds segments where two line strings overlap.
func findOverlappingSegments(ls1, ls2 orb.LineString) orb.MultiLineString {
	var overlaps orb.MultiLineString

	for i := 0; i < len(ls1)-1; i++ {
		for j := 0; j < len(ls2)-1; j++ {
			if segmentsCollinear(ls1[i], ls1[i+1], ls2[j], ls2[j+1]) {
				overlap := findSegmentOverlap(ls1[i], ls1[i+1], ls2[j], ls2[j+1])
				if len(overlap) >= 2 {
					overlaps = append(overlaps, overlap)
				}
			}
		}
	}

	return mergeOverlappingSegments(overlaps)
}

// segmentsCollinear checks if two segments are collinear.
func segmentsCollinear(p1, p2, p3, p4 orb.Point) bool {
	// Check if all four points are collinear
	cross1 := (p2[0]-p1[0])*(p3[1]-p1[1]) - (p2[1]-p1[1])*(p3[0]-p1[0])
	cross2 := (p2[0]-p1[0])*(p4[1]-p1[1]) - (p2[1]-p1[1])*(p4[0]-p1[0])
	return math.Abs(cross1) < Epsilon && math.Abs(cross2) < Epsilon
}

// findSegmentOverlap finds the overlapping portion of two collinear segments.
func findSegmentOverlap(p1, p2, p3, p4 orb.Point) orb.LineString {
	// Project all points onto the line
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	len1 := math.Sqrt(dx*dx + dy*dy)
	if len1 < Epsilon {
		return nil
	}

	// Normalize direction
	dx /= len1
	dy /= len1

	// Project points onto line
	t1 := 0.0
	t2 := len1
	t3 := (p3[0]-p1[0])*dx + (p3[1]-p1[1])*dy
	t4 := (p4[0]-p1[0])*dx + (p4[1]-p1[1])*dy

	// Ensure t3 < t4
	if t3 > t4 {
		t3, t4 = t4, t3
	}

	// Find overlap
	overlapStart := math.Max(t1, t3)
	overlapEnd := math.Min(t2, t4)

	if overlapEnd <= overlapStart+Epsilon {
		return nil // No overlap
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

// mergeOverlappingSegments merges consecutive overlapping segments.
func mergeOverlappingSegments(segments orb.MultiLineString) orb.MultiLineString {
	if len(segments) <= 1 {
		return segments
	}

	// Sort by starting point
	sort.Slice(segments, func(i, j int) bool {
		if segments[i][0][0] != segments[j][0][0] {
			return segments[i][0][0] < segments[j][0][0]
		}
		return segments[i][0][1] < segments[j][0][1]
	})

	var result orb.MultiLineString
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		// Check if segments can be merged (share an endpoint)
		if PointsEqual(current[len(current)-1], segments[i][0], Epsilon) {
			// Extend current
			current = append(current, segments[i][1:]...)
		} else {
			result = append(result, current)
			current = segments[i]
		}
	}
	result = append(result, current)

	return result
}

// splitLineAtPoints splits a line string at the given points.
// This first breaks the line into individual edge segments, then
// splits any edges that have interior split points.
func splitLineAtPoints(ls orb.LineString, points []orb.Point) orb.MultiLineString {
	if len(ls) < 2 {
		return orb.MultiLineString{}
	}

	// First, break line into individual edges
	var edges orb.MultiLineString
	for i := 0; i < len(ls)-1; i++ {
		edges = append(edges, orb.LineString{ls[i], ls[i+1]})
	}

	if len(points) == 0 {
		return edges
	}

	// For each edge, check if any split points are in its interior
	var result orb.MultiLineString
	for _, edge := range edges {
		result = append(result, splitEdgeAtPoints(edge, points)...)
	}

	return result
}

// splitEdgeAtPoints splits a single edge (2-point LineString) at interior points.
func splitEdgeAtPoints(edge orb.LineString, points []orb.Point) orb.MultiLineString {
	if len(edge) != 2 {
		return orb.MultiLineString{edge}
	}

	p1, p2 := edge[0], edge[1]
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	segLen := math.Sqrt(dx*dx + dy*dy)
	if segLen < Epsilon {
		return orb.MultiLineString{edge}
	}

	// Find all interior split points on this edge
	type alphaPoint struct {
		alpha float64
		point orb.Point
	}
	var splitEvents []alphaPoint

	for _, pt := range points {
		if PointOnSegment(pt, p1, p2, Epsilon) {
			// Calculate alpha
			ptDx := pt[0] - p1[0]
			ptDy := pt[1] - p1[1]
			alpha := (ptDx*dx + ptDy*dy) / (segLen * segLen)
			// Only split at interior points
			if alpha > Epsilon && alpha < 1-Epsilon {
				splitEvents = append(splitEvents, alphaPoint{alpha, pt})
			}
		}
	}

	if len(splitEvents) == 0 {
		return orb.MultiLineString{edge}
	}

	// Sort by alpha
	sort.Slice(splitEvents, func(i, j int) bool {
		return splitEvents[i].alpha < splitEvents[j].alpha
	})

	// Build split edges
	var result orb.MultiLineString
	current := p1
	for _, ev := range splitEvents {
		result = append(result, orb.LineString{current, ev.point})
		current = ev.point
	}
	result = append(result, orb.LineString{current, p2})

	return result
}

// pointOnLineString checks if a point lies on any segment of a line string.
func pointOnLineString(pt orb.Point, ls orb.LineString) bool {
	for i := 0; i < len(ls)-1; i++ {
		if PointOnSegment(pt, ls[i], ls[i+1], Epsilon) {
			return true
		}
	}
	return false
}

// deduplicateSegments removes duplicate segments from a multi-line string.
// Two segments are considered duplicates only if they have the same path
// (same sequence of points, possibly reversed).
func deduplicateSegments(segments orb.MultiLineString) orb.MultiLineString {
	if len(segments) <= 1 {
		return segments
	}

	var result orb.MultiLineString

	for _, seg := range segments {
		if len(seg) < 2 {
			continue
		}
		// Check if this segment is already in result
		isDuplicate := false
		for _, existing := range result {
			if segmentsAreSamePath(seg, existing) {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			result = append(result, seg)
		}
	}

	return result
}

// segmentsAreSamePath checks if two segments represent the same path
// (same points in same or reversed order).
func segmentsAreSamePath(s1, s2 orb.LineString) bool {
	if len(s1) != len(s2) {
		return false
	}

	// Check forward
	forward := true
	for i := range s1 {
		if !PointsEqual(s1[i], s2[i], Epsilon) {
			forward = false
			break
		}
	}
	if forward {
		return true
	}

	// Check reversed
	n := len(s1)
	for i := range s1 {
		if !PointsEqual(s1[i], s2[n-1-i], Epsilon) {
			return false
		}
	}
	return true
}

// combineLineStrings combines two line strings if they share endpoints.
func combineLineStrings(ls1, ls2 orb.LineString) orb.MultiLineString {
	if len(ls1) == 0 && len(ls2) == 0 {
		return orb.MultiLineString{}
	}
	if len(ls1) < 2 {
		return orb.MultiLineString{ls2}
	}
	if len(ls2) < 2 {
		return orb.MultiLineString{ls1}
	}

	// Check if they share endpoints
	ls1Start := ls1[0]
	ls1End := ls1[len(ls1)-1]
	ls2Start := ls2[0]
	ls2End := ls2[len(ls2)-1]

	// Try to combine if endpoints match
	if PointsEqual(ls1End, ls2Start, Epsilon) {
		combined := make(orb.LineString, 0, len(ls1)+len(ls2)-1)
		combined = append(combined, ls1...)
		combined = append(combined, ls2[1:]...)
		return orb.MultiLineString{combined}
	}

	if PointsEqual(ls1End, ls2End, Epsilon) {
		// Reverse ls2 and combine
		ls2Reversed := reverseLineString(ls2)
		combined := make(orb.LineString, 0, len(ls1)+len(ls2Reversed)-1)
		combined = append(combined, ls1...)
		combined = append(combined, ls2Reversed[1:]...)
		return orb.MultiLineString{combined}
	}

	if PointsEqual(ls1Start, ls2End, Epsilon) {
		combined := make(orb.LineString, 0, len(ls2)+len(ls1)-1)
		combined = append(combined, ls2...)
		combined = append(combined, ls1[1:]...)
		return orb.MultiLineString{combined}
	}

	if PointsEqual(ls1Start, ls2Start, Epsilon) {
		// Reverse ls2 and combine
		ls2Reversed := reverseLineString(ls2)
		combined := make(orb.LineString, 0, len(ls2Reversed)+len(ls1)-1)
		combined = append(combined, ls2Reversed...)
		combined = append(combined, ls1[1:]...)
		return orb.MultiLineString{combined}
	}

	// No shared endpoints, return as separate line strings
	return orb.MultiLineString{ls1, ls2}
}

// reverseLineString returns a reversed copy of the line string.
func reverseLineString(ls orb.LineString) orb.LineString {
	n := len(ls)
	result := make(orb.LineString, n)
	for i := 0; i < n; i++ {
		result[i] = ls[n-1-i]
	}
	return result
}

// lineStringContained checks if ls1 is completely contained within ls2.
func lineStringContained(ls1, ls2 orb.LineString) bool {
	if len(ls1) < 2 || len(ls2) < 2 {
		return false
	}

	// Check if all points of ls1 are on segments of ls2
	for _, pt := range ls1 {
		if !pointOnLineString(pt, ls2) {
			return false
		}
	}

	return true
}

// segmentIntersectionFull finds the intersection of two segments with full alpha values.
// For collinear segments, returns the overlap endpoints rather than a midpoint.
func segmentIntersectionFull(p1, p2, p3, p4 orb.Point) (orb.Point, float64, float64, bool) {
	d1x := p2[0] - p1[0]
	d1y := p2[1] - p1[1]
	d2x := p4[0] - p3[0]
	d2y := p4[1] - p3[1]

	cross := d1x*d2y - d1y*d2x
	if math.Abs(cross) < Epsilon {
		// Segments are parallel or collinear
		// For collinear segments, don't return a midpoint - this causes extra splits
		return orb.Point{}, 0, 0, false
	}

	dx := p3[0] - p1[0]
	dy := p3[1] - p1[1]

	alpha1 := (dx*d2y - dy*d2x) / cross
	alpha2 := (dx*d1y - dy*d1x) / cross

	if alpha1 >= -Epsilon && alpha1 <= 1+Epsilon && alpha2 >= -Epsilon && alpha2 <= 1+Epsilon {
		pt := orb.Point{
			p1[0] + alpha1*d1x,
			p1[1] + alpha1*d1y,
		}
		return pt, alpha1, alpha2, true
	}

	return orb.Point{}, 0, 0, false
}

// roundPoint rounds a point to reduce floating point precision issues.
func roundPoint(p orb.Point) orb.Point {
	scale := 1e9
	return orb.Point{
		float64(int64(p[0]*scale+0.5)) / scale,
		float64(int64(p[1]*scale+0.5)) / scale,
	}
}

// LineStringSymmetricDifference computes the symmetric difference of two line strings.
func LineStringSymmetricDifference(ls1, ls2 orb.LineString) orb.MultiLineString {
	diff1 := LineStringDifference(ls1, ls2)
	diff2 := LineStringDifference(ls2, ls1)
	combined := append(diff1, diff2...)
	// Deduplicate (segments from diff1 and diff2 shouldn't overlap, but just in case)
	return deduplicateSegments(combined)
}

// removeConsecutiveDuplicates removes consecutive duplicate points from a line string.
func removeConsecutiveDuplicates(ls orb.LineString) orb.LineString {
	if len(ls) < 2 {
		return ls
	}

	result := orb.LineString{ls[0]}
	for i := 1; i < len(ls); i++ {
		if !PointsEqual(ls[i], result[len(result)-1], Epsilon) {
			result = append(result, ls[i])
		}
	}

	return result
}
