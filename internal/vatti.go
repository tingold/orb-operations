package internal

import (
	"math"
	"sort"

	"github.com/paulmach/orb"
)

// VattiClipOperation represents the type of clipping operation
type VattiClipOperation int

const (
	VattiIntersection VattiClipOperation = iota
	VattiUnion
	VattiDifference
	VattiXor
)

// VattiClip performs polygon clipping.
func VattiClip(subject, clip orb.Polygon, op VattiClipOperation) orb.MultiPolygon {
	if len(subject) == 0 || len(subject[0]) < 3 {
		if op == VattiUnion {
			if len(clip) > 0 && len(clip[0]) >= 3 {
				return orb.MultiPolygon{clip}
			}
		}
		return orb.MultiPolygon{}
	}
	if len(clip) == 0 || len(clip[0]) < 3 {
		if op == VattiUnion || op == VattiDifference {
			return orb.MultiPolygon{subject}
		}
		return orb.MultiPolygon{}
	}

	// Normalize rings
	subjRing := prepRing(subject[0])
	clipRing := prepRing(clip[0])

	if len(subjRing) < 3 || len(clipRing) < 3 {
		if op == VattiUnion || op == VattiDifference {
			return orb.MultiPolygon{subject}
		}
		return orb.MultiPolygon{}
	}

	// Check for intersection
	hasIntersection := ringsHaveIntersection(subjRing, clipRing)

	if !hasIntersection {
		return handleNoOverlap(subject, clip, subjRing, clipRing, op)
	}

	switch op {
	case VattiIntersection:
		return computeIntersection(subject, clip, subjRing, clipRing)
	case VattiUnion:
		return computeUnion(subject, clip, subjRing, clipRing)
	case VattiDifference:
		return computeDifference(subject, clip, subjRing, clipRing)
	case VattiXor:
		return computeXor(subject, clip, subjRing, clipRing)
	}

	return orb.MultiPolygon{}
}

// prepRing prepares a ring for clipping (removes closing point, ensures CCW)
func prepRing(ring orb.Ring) orb.Ring {
	n := len(ring)
	if n < 3 {
		return nil
	}

	// Remove closing point if present
	if n > 1 && ptEq(ring[0], ring[n-1]) {
		ring = ring[:n-1]
		n--
	}
	if n < 3 {
		return nil
	}

	// Copy
	result := make(orb.Ring, n)
	copy(result, ring)

	// Ensure CCW
	if ringArea(result) < 0 {
		reverseRingV(result)
	}

	return result
}

// ringArea computes signed area (positive = CCW)
func ringArea(ring orb.Ring) float64 {
	n := len(ring)
	if n < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += ring[i][0] * ring[j][1]
		area -= ring[j][0] * ring[i][1]
	}
	return area / 2
}

// reverseRingV reverses a ring in place
func reverseRingV(ring orb.Ring) {
	for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
		ring[i], ring[j] = ring[j], ring[i]
	}
}

// ringsHaveIntersection checks if two rings have any edge intersection
func ringsHaveIntersection(r1, r2 orb.Ring) bool {
	n1, n2 := len(r1), len(r2)
	for i := 0; i < n1; i++ {
		for j := 0; j < n2; j++ {
			_, _, _, ok := segmentIntersection(r1[i], r1[(i+1)%n1], r2[j], r2[(j+1)%n2])
			if ok {
				return true
			}
		}
	}
	return false
}

// segmentIntersection finds the intersection of two line segments
func segmentIntersection(p1, p2, p3, p4 orb.Point) (orb.Point, float64, float64, bool) {
	d1x := p2[0] - p1[0]
	d1y := p2[1] - p1[1]
	d2x := p4[0] - p3[0]
	d2y := p4[1] - p3[1]

	cross := d1x*d2y - d1y*d2x
	if math.Abs(cross) < Epsilon {
		return orb.Point{}, 0, 0, false
	}

	dx := p3[0] - p1[0]
	dy := p3[1] - p1[1]

	t := (dx*d2y - dy*d2x) / cross
	u := (dx*d1y - dy*d1x) / cross

	if t > Epsilon && t < 1-Epsilon && u > Epsilon && u < 1-Epsilon {
		pt := orb.Point{p1[0] + t*d1x, p1[1] + t*d1y}
		return pt, t, u, true
	}
	return orb.Point{}, 0, 0, false
}

// handleNoOverlap handles the case when polygons don't have edge intersections
func handleNoOverlap(subject, clip orb.Polygon, subjRing, clipRing orb.Ring, op VattiClipOperation) orb.MultiPolygon {
	sInC := pointInRingV(subjRing[0], clipRing)
	cInS := pointInRingV(clipRing[0], subjRing)

	switch op {
	case VattiIntersection:
		if sInC {
			return orb.MultiPolygon{subject}
		}
		if cInS {
			return orb.MultiPolygon{clip}
		}
		return orb.MultiPolygon{}

	case VattiUnion:
		if sInC {
			return orb.MultiPolygon{clip}
		}
		if cInS {
			return orb.MultiPolygon{subject}
		}
		return orb.MultiPolygon{subject, clip}

	case VattiDifference:
		if sInC {
			return orb.MultiPolygon{}
		}
		if cInS {
			result := make(orb.Polygon, len(subject)+1)
			copy(result, subject)
			hole := makeCopy(clip[0])
			reverseRingV(hole)
			result = append(result, hole)
			return orb.MultiPolygon{result}
		}
		return orb.MultiPolygon{subject}

	case VattiXor:
		if sInC {
			result := make(orb.Polygon, len(clip)+1)
			copy(result, clip)
			hole := makeCopy(subject[0])
			reverseRingV(hole)
			result = append(result, hole)
			return orb.MultiPolygon{result}
		}
		if cInS {
			result := make(orb.Polygon, len(subject)+1)
			copy(result, subject)
			hole := makeCopy(clip[0])
			reverseRingV(hole)
			result = append(result, hole)
			return orb.MultiPolygon{result}
		}
		return orb.MultiPolygon{subject, clip}
	}
	return orb.MultiPolygon{}
}

// makeCopy creates a copy of a ring
func makeCopy(ring orb.Ring) orb.Ring {
	r := make(orb.Ring, len(ring))
	copy(r, ring)
	return r
}

// pointInRingV checks if a point is inside a ring
func pointInRingV(p orb.Point, ring orb.Ring) bool {
	if len(ring) < 3 {
		return false
	}
	inside := false
	n := len(ring)
	j := n - 1
	for i := 0; i < n; i++ {
		pi, pj := ring[i], ring[j]
		if ((pi[1] > p[1]) != (pj[1] > p[1])) &&
			(p[0] < (pj[0]-pi[0])*(p[1]-pi[1])/(pj[1]-pi[1])+pi[0]) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// computeIntersection computes polygon intersection using Sutherland-Hodgman
func computeIntersection(subject, clip orb.Polygon, subjRing, clipRing orb.Ring) orb.MultiPolygon {
	// Sutherland-Hodgman clipping
	result := sutherlandHodgmanClip(subjRing, clipRing)

	if len(result) < 3 {
		return orb.MultiPolygon{}
	}

	// Close the ring
	result = closeRing(result)

	// Ensure CCW
	if ringArea(result[:len(result)-1]) < 0 {
		reverseRingV(result)
	}

	poly := orb.Polygon{result}

	// Handle subject holes
	for i := 1; i < len(subject); i++ {
		hole := subject[i]
		if len(hole) >= 3 {
			holeResult := sutherlandHodgmanClip(prepRing(hole), clipRing)
			if len(holeResult) >= 3 {
				holeResult = closeRing(holeResult)
				// Ensure hole is CW
				if ringArea(holeResult[:len(holeResult)-1]) > 0 {
					reverseRingV(holeResult)
				}
				poly = append(poly, holeResult)
			}
		}
	}

	// Handle clip holes
	for i := 1; i < len(clip); i++ {
		hole := clip[i]
		if len(hole) >= 3 {
			holeResult := sutherlandHodgmanClip(result[:len(result)-1], prepRing(hole))
			if len(holeResult) >= 3 {
				holeResult = closeRing(holeResult)
				if ringArea(holeResult[:len(holeResult)-1]) > 0 {
					reverseRingV(holeResult)
				}
				poly = append(poly, holeResult)
			}
		}
	}

	return orb.MultiPolygon{poly}
}

// sutherlandHodgmanClip clips subject polygon against clip polygon
func sutherlandHodgmanClip(subject, clip orb.Ring) orb.Ring {
	output := make(orb.Ring, len(subject))
	copy(output, subject)

	for i := 0; i < len(clip); i++ {
		if len(output) == 0 {
			return nil
		}

		input := output
		output = nil

		edgeStart := clip[i]
		edgeEnd := clip[(i+1)%len(clip)]

		for j := 0; j < len(input); j++ {
			current := input[j]
			next := input[(j+1)%len(input)]

			currInside := isLeftOfEdge(current, edgeStart, edgeEnd)
			nextInside := isLeftOfEdge(next, edgeStart, edgeEnd)

			if currInside {
				output = appendUniqueV(output, current)
				if !nextInside {
					if inter, ok := lineIntersection(current, next, edgeStart, edgeEnd); ok {
						output = appendUniqueV(output, inter)
					}
				}
			} else if nextInside {
				if inter, ok := lineIntersection(current, next, edgeStart, edgeEnd); ok {
					output = appendUniqueV(output, inter)
				}
			}
		}
	}

	return output
}

// isLeftOfEdge checks if point is on the left side of directed edge
func isLeftOfEdge(p, edgeStart, edgeEnd orb.Point) bool {
	return (edgeEnd[0]-edgeStart[0])*(p[1]-edgeStart[1])-
		(edgeEnd[1]-edgeStart[1])*(p[0]-edgeStart[0]) >= -Epsilon
}

// lineIntersection finds intersection of two lines (not segments)
func lineIntersection(p1, p2, p3, p4 orb.Point) (orb.Point, bool) {
	d1x := p2[0] - p1[0]
	d1y := p2[1] - p1[1]
	d2x := p4[0] - p3[0]
	d2y := p4[1] - p3[1]

	cross := d1x*d2y - d1y*d2x
	if math.Abs(cross) < Epsilon {
		return orb.Point{}, false
	}

	dx := p3[0] - p1[0]
	dy := p3[1] - p1[1]
	t := (dx*d2y - dy*d2x) / cross

	return orb.Point{p1[0] + t*d1x, p1[1] + t*d1y}, true
}

// appendUniqueV appends point if not duplicate of last
func appendUniqueV(ring orb.Ring, pt orb.Point) orb.Ring {
	if len(ring) > 0 && ptEq(ring[len(ring)-1], pt) {
		return ring
	}
	return append(ring, pt)
}

// closeRing ensures ring is closed
func closeRing(ring orb.Ring) orb.Ring {
	if len(ring) < 3 {
		return ring
	}
	if !ptEq(ring[0], ring[len(ring)-1]) {
		ring = append(ring, ring[0])
	}
	return ring
}

// computeUnion computes polygon union
func computeUnion(subject, clip orb.Polygon, subjRing, clipRing orb.Ring) orb.MultiPolygon {
	// Use Greiner-Hormann for union
	result := greinerHormannOp(subjRing, clipRing, false) // false = union

	if len(result) == 0 {
		// Fallback
		if allPointsIn(subjRing, clipRing) {
			return orb.MultiPolygon{clip}
		}
		if allPointsIn(clipRing, subjRing) {
			return orb.MultiPolygon{subject}
		}
		return orb.MultiPolygon{subject, clip}
	}

	// Process holes
	return processUnionHoles(result, subject, clip)
}

// computeDifference computes polygon difference (subject - clip)
func computeDifference(subject, clip orb.Polygon, subjRing, clipRing orb.Ring) orb.MultiPolygon {
	// For difference A-B:
	// Result = parts of A that are outside B
	
	// First compute the intersection (the part to remove)
	intersection := sutherlandHodgmanClip(subjRing, clipRing)
	
	if len(intersection) < 3 {
		// No intersection - subject is entirely outside clip
		return orb.MultiPolygon{subject}
	}
	
	// Check if subject is entirely inside clip
	if allPointsIn(subjRing, clipRing) {
		return orb.MultiPolygon{}
	}
	
	// Use Greiner-Hormann but with correct difference logic
	result := greinerHormannDifference(subjRing, clipRing)
	
	if len(result) == 0 {
		// Fallback: if clip is inside subject, make it a hole
		if allPointsIn(clipRing, subjRing) {
			poly := make(orb.Polygon, 1, 2)
			poly[0] = closeRing(makeCopy(subjRing))
			hole := closeRing(makeCopy(clipRing))
			reverseRingV(hole)
			poly = append(poly, hole)
			return orb.MultiPolygon{poly}
		}
		return orb.MultiPolygon{subject}
	}

	return processDifferenceHoles(result, subject, clip)
}

// computeXor computes symmetric difference
func computeXor(subject, clip orb.Polygon, subjRing, clipRing orb.Ring) orb.MultiPolygon {
	diff1 := computeDifference(subject, clip, subjRing, clipRing)
	diff2 := computeDifference(clip, subject, clipRing, subjRing)

	result := make(orb.MultiPolygon, 0, len(diff1)+len(diff2))
	result = append(result, diff1...)
	result = append(result, diff2...)
	return result
}

// greinerHormannDifference performs difference operation specifically
func greinerHormannDifference(subjRing, clipRing orb.Ring) orb.MultiPolygon {
	// Build chains
	subjChain := buildLinkedList(subjRing)
	clipChain := buildLinkedList(clipRing)

	if subjChain == nil || clipChain == nil {
		return nil
	}

	// Insert intersections
	count := insertAllIntersections(subjChain, clipChain)
	if count == 0 {
		return nil
	}

	// Mark entry/exit
	markChain(subjChain, clipRing)
	markChain(clipChain, subjRing)

	// For difference: trace from exit points on subject, but at entry points
	// we need to follow the clip boundary in reverse direction
	return traceDifference(subjChain, clipChain)
}

// greinerHormannOp performs Greiner-Hormann polygon clipping
func greinerHormannOp(subjRing, clipRing orb.Ring, isDifference bool) orb.MultiPolygon {
	// Build chains
	subjChain := buildLinkedList(subjRing)
	clipChain := buildLinkedList(clipRing)

	if subjChain == nil || clipChain == nil {
		return nil
	}

	// Insert intersections
	count := insertAllIntersections(subjChain, clipChain)
	if count == 0 {
		return nil
	}

	// Mark entry/exit
	markChain(subjChain, clipRing)
	markChain(clipChain, subjRing)

	// Trace result
	return traceGH(subjChain, clipChain, isDifference)
}

// ghNode is a node in the linked list
type ghNode struct {
	pt       orb.Point
	next     *ghNode
	prev     *ghNode
	neighbor *ghNode
	alpha    float64
	isInter  bool
	isEntry  bool
	visited  bool
}

// buildLinkedList creates a circular linked list from a ring
func buildLinkedList(ring orb.Ring) *ghNode {
	if len(ring) < 3 {
		return nil
	}

	var head, tail *ghNode
	for _, pt := range ring {
		n := &ghNode{pt: pt}
		if head == nil {
			head = n
		}
		if tail != nil {
			tail.next = n
			n.prev = tail
		}
		tail = n
	}
	tail.next = head
	head.prev = tail
	return head
}

// insertAllIntersections finds and inserts all intersections
func insertAllIntersections(subjHead, clipHead *ghNode) int {
	type interInfo struct {
		pt     orb.Point
		sn, cn *ghNode
		sa, ca float64
	}
	var inters []interInfo

	sn := subjHead
	for {
		cn := clipHead
		for {
			pt, sa, ca, ok := segmentIntersection(sn.pt, sn.next.pt, cn.pt, cn.next.pt)
			if ok {
				inters = append(inters, interInfo{pt, sn, cn, sa, ca})
			}
			cn = cn.next
			if cn == clipHead {
				break
			}
		}
		sn = sn.next
		if sn == subjHead {
			break
		}
	}

	if len(inters) == 0 {
		return 0
	}

	// Group by subject edge
	sEdges := make(map[*ghNode][]interInfo)
	for _, i := range inters {
		sEdges[i.sn] = append(sEdges[i.sn], i)
	}
	for _, arr := range sEdges {
		sort.Slice(arr, func(i, j int) bool { return arr[i].sa < arr[j].sa })
	}

	// Insert and track
	interMap := make(map[int]*ghNode)
	for sn, arr := range sEdges {
		after := sn
		for i := range arr {
			nn := &ghNode{pt: arr[i].pt, alpha: arr[i].sa, isInter: true}
			nn.next = after.next
			nn.prev = after
			after.next.prev = nn
			after.next = nn
			// Find index in original inters
			for idx, orig := range inters {
				if ptEq(orig.pt, arr[i].pt) {
					interMap[idx] = nn
					break
				}
			}
			after = nn
		}
	}

	// Group by clip edge
	cEdges := make(map[*ghNode][]interInfo)
	for _, i := range inters {
		cEdges[i.cn] = append(cEdges[i.cn], i)
	}
	for _, arr := range cEdges {
		sort.Slice(arr, func(i, j int) bool { return arr[i].ca < arr[j].ca })
	}

	// Insert and link
	for cn, arr := range cEdges {
		after := cn
		for i := range arr {
			nn := &ghNode{pt: arr[i].pt, alpha: arr[i].ca, isInter: true}
			nn.next = after.next
			nn.prev = after
			after.next.prev = nn
			after.next = nn
			// Link neighbor
			for idx, orig := range inters {
				if ptEq(orig.pt, arr[i].pt) {
					if sNode, ok := interMap[idx]; ok {
						nn.neighbor = sNode
						sNode.neighbor = nn
					}
					break
				}
			}
			after = nn
		}
	}

	return len(inters)
}

// markChain marks entry/exit status
func markChain(head *ghNode, otherRing orb.Ring) {
	if head == nil {
		return
	}

	cur := head
	for cur.isInter {
		cur = cur.next
		if cur == head {
			return
		}
	}

	inside := pointInRingV(cur.pt, otherRing)

	start := cur
	for {
		if cur.isInter {
			cur.isEntry = !inside
			inside = !inside
		}
		cur = cur.next
		if cur == start {
			break
		}
	}
}

// traceDifference traces difference result
func traceDifference(subjHead, clipHead *ghNode) orb.MultiPolygon {
	resetVisited(subjHead)
	resetVisited(clipHead)

	var result orb.MultiPolygon

	// For difference A-B: start at exit points (leaving B) on subject
	// At entry points (entering B), switch to clip and traverse in REVERSE
	cur := subjHead
	for {
		if cur.isInter && !cur.visited && !cur.isEntry {
			ring := traceDifferenceRing(cur)
			if len(ring) >= 3 {
				ring = closeRing(ring)
				if ringArea(ring[:len(ring)-1]) < 0 {
					reverseRingV(ring)
				}
				result = append(result, orb.Polygon{ring})
			}
		}
		cur = cur.next
		if cur == subjHead {
			break
		}
	}

	return result
}

// traceDifferenceRing traces a single difference ring
func traceDifferenceRing(start *ghNode) orb.Ring {
	var ring orb.Ring
	cur := start
	onSubject := true

	for i := 0; i < 10000; i++ {
		ring = appendUniqueV(ring, cur.pt)
		cur.visited = true

		if cur.isInter && cur.neighbor != nil {
			cur.neighbor.visited = true

			if onSubject {
				if cur.isEntry {
					// Entering clip - switch to clip and go BACKWARDS
					cur = cur.neighbor.prev
					onSubject = false
				} else {
					// Exiting clip - continue on subject
					cur = cur.next
				}
			} else {
				// On clip (going backwards)
				if cur.neighbor.isEntry {
					// This is an entry point on subject - switch back to subject
					cur = cur.neighbor.next
					onSubject = true
				} else {
					// Continue backwards on clip
					cur = cur.prev
				}
			}
		} else {
			if onSubject {
				cur = cur.next
			} else {
				cur = cur.prev
			}
		}

		if cur == start {
			break
		}
		if cur.isInter && cur.visited && len(ring) > 2 {
			break
		}
	}

	return ring
}

// traceGH traces result polygons
func traceGH(subjHead, clipHead *ghNode, isDifference bool) orb.MultiPolygon {
	resetVisited(subjHead)
	resetVisited(clipHead)

	var result orb.MultiPolygon

	// For union: trace from exit points (leaving other polygon) to stay on outer boundary
	// For difference: trace from exit points on subject (leaving clip)
	
	cur := subjHead
	for {
		if cur.isInter && !cur.visited {
			// Exit point = we were inside and are leaving
			// For both union and difference we start at exit points (!isEntry)
			if !cur.isEntry {
				ring := traceRingGH(cur, isDifference)
				if len(ring) >= 3 {
					ring = closeRing(ring)
					if ringArea(ring[:len(ring)-1]) < 0 {
						reverseRingV(ring)
					}
					result = append(result, orb.Polygon{ring})
				}
			}
		}
		cur = cur.next
		if cur == subjHead {
			break
		}
	}

	return result
}

// resetVisited resets visited flags
func resetVisited(head *ghNode) {
	if head == nil {
		return
	}
	cur := head
	for {
		cur.visited = false
		cur = cur.next
		if cur == head {
			break
		}
	}
}

// traceRingGH traces a single ring
func traceRingGH(start *ghNode, isDifference bool) orb.Ring {
	var ring orb.Ring
	cur := start

	for i := 0; i < 10000; i++ {
		ring = appendUniqueV(ring, cur.pt)
		cur.visited = true

		if cur.isInter && cur.neighbor != nil {
			cur.neighbor.visited = true
			// At intersection:
			// isEntry=true means we're entering other polygon (going from outside to inside)
			// isEntry=false means we're exiting other polygon (going from inside to outside)
			//
			// For union: we want to stay OUTSIDE both polygons
			// - At entry point (about to go inside): switch to other polygon to avoid going in
			// - At exit point (coming out): continue on same polygon
			//
			// For difference (A-B): we want parts of A that are OUTSIDE B
			// - At entry point into B: switch to B and traverse B boundary
			// - At exit point from B: continue on A
			if cur.isEntry {
				cur = cur.neighbor.next
			} else {
				cur = cur.next
			}
		} else {
			cur = cur.next
		}

		if cur == start {
			break
		}
		if cur.isInter && cur.visited && len(ring) > 2 {
			break
		}
	}

	return ring
}

// allPointsIn checks if all points of inner are inside outer
func allPointsIn(inner, outer orb.Ring) bool {
	for _, pt := range inner {
		if !pointInRingV(pt, outer) {
			return false
		}
	}
	return true
}

// processUnionHoles handles holes for union operation
func processUnionHoles(result orb.MultiPolygon, subject, clip orb.Polygon) orb.MultiPolygon {
	for i := 1; i < len(subject); i++ {
		hole := subject[i]
		if len(hole) >= 3 && !ringsHaveIntersection(prepRing(hole), prepRing(clip[0])) {
			for j := range result {
				if allPointsIn(prepRing(hole), result[j][0][:len(result[j][0])-1]) {
					result[j] = append(result[j], hole)
					break
				}
			}
		}
	}

	for i := 1; i < len(clip); i++ {
		hole := clip[i]
		if len(hole) >= 3 && !ringsHaveIntersection(prepRing(hole), prepRing(subject[0])) {
			for j := range result {
				if allPointsIn(prepRing(hole), result[j][0][:len(result[j][0])-1]) {
					result[j] = append(result[j], hole)
					break
				}
			}
		}
	}

	return result
}

// processDifferenceHoles handles holes for difference operation
func processDifferenceHoles(result orb.MultiPolygon, subject, clip orb.Polygon) orb.MultiPolygon {
	for i := 1; i < len(subject); i++ {
		hole := subject[i]
		if len(hole) >= 3 && !ringsHaveIntersection(prepRing(hole), prepRing(clip[0])) {
			for j := range result {
				if allPointsIn(prepRing(hole), result[j][0][:len(result[j][0])-1]) {
					result[j] = append(result[j], hole)
					break
				}
			}
		}
	}

	return result
}

// ptEq checks if two points are equal
func ptEq(p1, p2 orb.Point) bool {
	return math.Abs(p1[0]-p2[0]) < Epsilon && math.Abs(p1[1]-p2[1]) < Epsilon
}
