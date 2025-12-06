package internal

import (
	"math"

	"github.com/paulmach/orb"
)

// DefaultQuadrantSegments is the default number of segments used to approximate a quarter circle.
const DefaultQuadrantSegments = 8

// CapStyle defines how line ends are handled in buffering.
type CapStyle int

const (
	// CapRound creates rounded ends (default).
	CapRound CapStyle = iota
	// CapFlat creates flat/butt ends.
	CapFlat
	// CapSquare creates square ends that extend past the endpoint.
	CapSquare
)

// JoinStyle defines how corners are handled in buffering.
type JoinStyle int

const (
	// JoinRound creates rounded corners (default).
	JoinRound JoinStyle = iota
	// JoinMiter creates pointed corners.
	JoinMiter
	// JoinBevel creates flat corners.
	JoinBevel
)

// BufferParams holds parameters for buffer operations.
type BufferParams struct {
	QuadrantSegments int       // Number of segments per quarter circle
	CapStyle         CapStyle  // Line end style
	JoinStyle        JoinStyle // Corner style
	MiterLimit       float64   // Miter limit for JoinMiter
}

// DefaultBufferParams returns default buffer parameters.
func DefaultBufferParams() BufferParams {
	return BufferParams{
		QuadrantSegments: DefaultQuadrantSegments,
		CapStyle:         CapRound,
		JoinStyle:        JoinRound,
		MiterLimit:       5.0,
	}
}

// BufferPoint creates a circular polygon around a point.
func BufferPoint(p orb.Point, distance float64, params BufferParams) orb.Polygon {
	if distance <= 0 {
		return orb.Polygon{}
	}

	segments := params.QuadrantSegments * 4
	if segments < 4 {
		segments = 4
	}

	ring := make(orb.Ring, segments+1)
	angleStep := 2 * math.Pi / float64(segments)

	for i := 0; i < segments; i++ {
		angle := float64(i) * angleStep
		ring[i] = orb.Point{
			p[0] + distance*math.Cos(angle),
			p[1] + distance*math.Sin(angle),
		}
	}
	ring[segments] = ring[0] // Close the ring

	return orb.Polygon{ring}
}

// BufferLineString creates a polygon around a linestring at the given distance.
func BufferLineString(ls orb.LineString, distance float64, params BufferParams) orb.Polygon {
	if len(ls) < 2 || distance <= 0 {
		return orb.Polygon{}
	}

	// For a single segment
	if len(ls) == 2 {
		return bufferSegment(ls[0], ls[1], distance, params)
	}

	// For multi-segment linestrings, create offset lines and join them
	return bufferMultiSegmentLine(ls, distance, params)
}

// BufferPolygon creates a buffered polygon at the given distance.
// Positive distance expands the polygon, negative distance shrinks it.
func BufferPolygon(poly orb.Polygon, distance float64, params BufferParams) orb.Polygon {
	if len(poly) == 0 || len(poly[0]) < 4 {
		return orb.Polygon{}
	}

	if distance == 0 {
		return poly
	}

	// Handle the outer ring
	outerRing := offsetRing(poly[0], distance, params, false)
	if len(outerRing) < 4 {
		return orb.Polygon{} // Polygon collapsed
	}

	result := orb.Polygon{outerRing}

	// Handle holes (offset in opposite direction)
	for i := 1; i < len(poly); i++ {
		holeRing := offsetRing(poly[i], -distance, params, true)
		if len(holeRing) >= 4 {
			result = append(result, holeRing)
		}
	}

	return result
}

// bufferSegment creates a buffer around a single line segment.
func bufferSegment(p1, p2 orb.Point, distance float64, params BufferParams) orb.Polygon {
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	length := math.Sqrt(dx*dx + dy*dy)

	if length < Epsilon {
		// Degenerate segment, treat as point
		return BufferPoint(p1, distance, params)
	}

	// Normal vector (perpendicular)
	nx := -dy / length
	ny := dx / length

	// Unit vector along the segment
	ux := dx / length
	uy := dy / length

	var ring orb.Ring

	// Right side offset points (going from p1 to p2)
	ring = append(ring,
		orb.Point{p1[0] + nx*distance, p1[1] + ny*distance},
		orb.Point{p2[0] + nx*distance, p2[1] + ny*distance},
	)

	// End cap at p2
	ring = append(ring, createCap(p2, ux, uy, distance, params, true)...)

	// Left side offset points (going from p2 to p1)
	ring = append(ring,
		orb.Point{p2[0] - nx*distance, p2[1] - ny*distance},
		orb.Point{p1[0] - nx*distance, p1[1] - ny*distance},
	)

	// End cap at p1
	ring = append(ring, createCap(p1, -ux, -uy, distance, params, true)...)

	// Close the ring
	ring = append(ring, ring[0])

	return orb.Polygon{ring}
}

// bufferMultiSegmentLine creates a buffer around a multi-segment linestring.
func bufferMultiSegmentLine(ls orb.LineString, distance float64, params BufferParams) orb.Polygon {
	if len(ls) < 2 {
		return orb.Polygon{}
	}

	// Build right side offset points
	var rightSide []orb.Point
	// Build left side offset points (in reverse order)
	var leftSide []orb.Point

	for i := 0; i < len(ls)-1; i++ {
		p1, p2 := ls[i], ls[i+1]
		dx := p2[0] - p1[0]
		dy := p2[1] - p1[1]
		length := math.Sqrt(dx*dx + dy*dy)

		if length < Epsilon {
			continue
		}

		// Normal vector
		nx := -dy / length
		ny := dx / length

		// Right side points for this segment
		r1 := orb.Point{p1[0] + nx*distance, p1[1] + ny*distance}
		r2 := orb.Point{p2[0] + nx*distance, p2[1] + ny*distance}

		// Left side points for this segment
		l1 := orb.Point{p1[0] - nx*distance, p1[1] - ny*distance}
		l2 := orb.Point{p2[0] - nx*distance, p2[1] - ny*distance}

		if i == 0 {
			// First segment - add start cap
			ux := dx / length
			uy := dy / length
			startCap := createCap(p1, -ux, -uy, distance, params, false)
			// Add left point, then cap (going counter-clockwise), then right point
			leftSide = append(leftSide, l1)
			rightSide = append(rightSide, r1)
			_ = startCap // We'll handle the cap differently
		}

		if i > 0 {
			// Add join at corner
			prevP1, prevP2 := ls[i-1], ls[i]
			rightJoin := createJoin(prevP1, prevP2, p2, distance, params, true)
			leftJoin := createJoin(prevP1, prevP2, p2, distance, params, false)
			rightSide = append(rightSide, rightJoin...)
			leftSide = append(leftSide, leftJoin...)
		}

		rightSide = append(rightSide, r2)
		leftSide = append(leftSide, l2)
	}

	// Build the final ring
	var ring orb.Ring

	// Start with right side
	ring = append(ring, rightSide...)

	// Add end cap
	lastIdx := len(ls) - 1
	dx := ls[lastIdx][0] - ls[lastIdx-1][0]
	dy := ls[lastIdx][1] - ls[lastIdx-1][1]
	length := math.Sqrt(dx*dx + dy*dy)
	if length >= Epsilon {
		ux := dx / length
		uy := dy / length
		endCap := createCap(ls[lastIdx], ux, uy, distance, params, true)
		ring = append(ring, endCap...)
	}

	// Add left side in reverse
	for i := len(leftSide) - 1; i >= 0; i-- {
		ring = append(ring, leftSide[i])
	}

	// Add start cap
	dx = ls[1][0] - ls[0][0]
	dy = ls[1][1] - ls[0][1]
	length = math.Sqrt(dx*dx + dy*dy)
	if length >= Epsilon {
		ux := dx / length
		uy := dy / length
		startCap := createCap(ls[0], -ux, -uy, distance, params, true)
		ring = append(ring, startCap...)
	}

	// Close the ring
	if len(ring) > 0 {
		ring = append(ring, ring[0])
	}

	return orb.Polygon{ring}
}

// createCap creates end cap points at a line end.
func createCap(p orb.Point, ux, uy, distance float64, params BufferParams, clockwise bool) []orb.Point {
	switch params.CapStyle {
	case CapFlat:
		return nil // No additional points needed

	case CapSquare:
		// Extend past the endpoint
		nx, ny := -uy, ux // Normal (perpendicular to direction)
		if !clockwise {
			nx, ny = -nx, -ny
		}
		return []orb.Point{
			{p[0] + ux*distance + nx*distance, p[1] + uy*distance + ny*distance},
			{p[0] + ux*distance - nx*distance, p[1] + uy*distance - ny*distance},
		}

	case CapRound:
		fallthrough
	default:
		// Create arc from one side to the other
		segments := params.QuadrantSegments * 2
		if segments < 2 {
			segments = 2
		}

		// Direction perpendicular to ux,uy
		nx, ny := -uy, ux

		var points []orb.Point
		startAngle := math.Atan2(ny, nx)
		angleStep := math.Pi / float64(segments)
		if clockwise {
			angleStep = -angleStep
		}

		for i := 0; i <= segments; i++ {
			angle := startAngle + float64(i)*angleStep
			points = append(points, orb.Point{
				p[0] + distance*math.Cos(angle),
				p[1] + distance*math.Sin(angle),
			})
		}
		return points
	}
}

// createJoin creates corner join points.
func createJoin(p1, p2, p3 orb.Point, distance float64, params BufferParams, rightSide bool) []orb.Point {
	// Vectors for the two segments
	dx1 := p2[0] - p1[0]
	dy1 := p2[1] - p1[1]
	len1 := math.Sqrt(dx1*dx1 + dy1*dy1)

	dx2 := p3[0] - p2[0]
	dy2 := p3[1] - p2[1]
	len2 := math.Sqrt(dx2*dx2 + dy2*dy2)

	if len1 < Epsilon || len2 < Epsilon {
		return nil
	}

	// Unit vectors
	ux1, uy1 := dx1/len1, dy1/len1
	ux2, uy2 := dx2/len2, dy2/len2

	// Normals (pointing right of direction)
	nx1, ny1 := -uy1, ux1
	nx2, ny2 := -uy2, ux2

	if !rightSide {
		nx1, ny1 = -nx1, -ny1
		nx2, ny2 = -nx2, -ny2
	}

	// Cross product to determine turn direction
	cross := ux1*uy2 - uy1*ux2
	if rightSide {
		cross = -cross
	}

	// Offset points at the corner
	offset1 := orb.Point{p2[0] + nx1*distance, p2[1] + ny1*distance}
	offset2 := orb.Point{p2[0] + nx2*distance, p2[1] + ny2*distance}

	if cross > Epsilon {
		// Outside of turn - need join
		switch params.JoinStyle {
		case JoinMiter:
			// Calculate miter point
			miterPt := calculateMiterPoint(p2, offset1, offset2, nx1, ny1, nx2, ny2, distance, params.MiterLimit)
			if miterPt != nil {
				return []orb.Point{*miterPt}
			}
			// Fall back to bevel
			return []orb.Point{offset1, offset2}

		case JoinBevel:
			return []orb.Point{offset1, offset2}

		case JoinRound:
			fallthrough
		default:
			// Create arc between the two offset points
			return createArcJoin(p2, offset1, offset2, distance, params.QuadrantSegments, rightSide)
		}
	}

	// Inside of turn - just return the intersection or closer point
	return nil
}

// calculateMiterPoint calculates the miter join point.
func calculateMiterPoint(corner orb.Point, offset1, offset2 orb.Point, nx1, ny1, nx2, ny2, distance, miterLimit float64) *orb.Point {
	// Direction vectors of the offset lines
	dx1, dy1 := -ny1, nx1 // Perpendicular to normal = along original line
	dx2, dy2 := -ny2, nx2

	// Solve for intersection
	denom := dx1*dy2 - dy1*dx2
	if math.Abs(denom) < Epsilon {
		return nil // Parallel lines
	}

	t := ((offset2[0]-offset1[0])*dy2 - (offset2[1]-offset1[1])*dx2) / denom
	miterX := offset1[0] + t*dx1
	miterY := offset1[1] + t*dy1

	// Check miter limit
	miterDist := math.Sqrt((miterX-corner[0])*(miterX-corner[0]) + (miterY-corner[1])*(miterY-corner[1]))
	if miterDist > distance*miterLimit {
		return nil // Exceeds miter limit
	}

	return &orb.Point{miterX, miterY}
}

// createArcJoin creates an arc between two offset points for round joins.
func createArcJoin(center, p1, p2 orb.Point, distance float64, quadrantSegments int, clockwise bool) []orb.Point {
	angle1 := math.Atan2(p1[1]-center[1], p1[0]-center[0])
	angle2 := math.Atan2(p2[1]-center[1], p2[0]-center[0])

	// Normalize angle difference
	var angleDiff float64
	if clockwise {
		angleDiff = angle2 - angle1
		if angleDiff < 0 {
			angleDiff += 2 * math.Pi
		}
	} else {
		angleDiff = angle1 - angle2
		if angleDiff < 0 {
			angleDiff += 2 * math.Pi
		}
		angleDiff = -angleDiff
	}

	// Number of segments based on angle
	segments := int(math.Ceil(math.Abs(angleDiff) / (math.Pi / 2) * float64(quadrantSegments)))
	if segments < 1 {
		segments = 1
	}

	var points []orb.Point
	points = append(points, p1)

	angleStep := angleDiff / float64(segments)
	for i := 1; i < segments; i++ {
		angle := angle1 + float64(i)*angleStep
		points = append(points, orb.Point{
			center[0] + distance*math.Cos(angle),
			center[1] + distance*math.Sin(angle),
		})
	}

	points = append(points, p2)
	return points
}

// offsetRing offsets a ring by the given distance.
func offsetRing(ring orb.Ring, distance float64, params BufferParams, isHole bool) orb.Ring {
	if len(ring) < 4 {
		return nil
	}

	// Remove the closing point for processing
	n := len(ring) - 1
	pts := ring[:n]

	var offsetPts []orb.Point

	for i := 0; i < n; i++ {
		prev := pts[(i-1+n)%n]
		curr := pts[i]
		next := pts[(i+1)%n]

		// Calculate offset point at this vertex
		offsetPt := offsetVertex(prev, curr, next, distance, params, isHole)
		if offsetPt != nil {
			offsetPts = append(offsetPts, *offsetPt)
		}
	}

	if len(offsetPts) < 3 {
		return nil
	}

	// Close the ring
	result := make(orb.Ring, len(offsetPts)+1)
	copy(result, offsetPts)
	result[len(offsetPts)] = offsetPts[0]

	// Ensure correct orientation
	if isHole {
		// Holes should be clockwise
		if !PolygonOrientation(result) {
			reverseRingInPlace(result)
		}
	} else {
		// Outer rings should be counter-clockwise
		if PolygonOrientation(result) {
			reverseRingInPlace(result)
		}
	}

	return result
}

// offsetVertex calculates the offset position for a vertex.
func offsetVertex(prev, curr, next orb.Point, distance float64, params BufferParams, isHole bool) *orb.Point {
	// Vectors
	dx1 := curr[0] - prev[0]
	dy1 := curr[1] - prev[1]
	len1 := math.Sqrt(dx1*dx1 + dy1*dy1)

	dx2 := next[0] - curr[0]
	dy2 := next[1] - curr[1]
	len2 := math.Sqrt(dx2*dx2 + dy2*dy2)

	if len1 < Epsilon || len2 < Epsilon {
		return &curr
	}

	// Unit vectors
	ux1, uy1 := dx1/len1, dy1/len1
	ux2, uy2 := dx2/len2, dy2/len2

	// Normals - RIGHT perpendicular (pointing outward for CCW ring, inward for CW ring)
	// For CCW ring, interior is on LEFT, exterior on RIGHT
	nx1, ny1 := uy1, -ux1
	nx2, ny2 := uy2, -ux2

	// For holes (CW rings), the outward direction is flipped
	if isHole {
		nx1, ny1 = -nx1, -ny1
		nx2, ny2 = -nx2, -ny2
	}

	// Bisector direction
	bx := nx1 + nx2
	by := ny1 + ny2
	bLen := math.Sqrt(bx*bx + by*by)

	if bLen < Epsilon {
		// Straight line - just offset perpendicular
		return &orb.Point{
			curr[0] + nx1*distance,
			curr[1] + ny1*distance,
		}
	}

	// Normalized bisector
	bx, by = bx/bLen, by/bLen

	// Calculate offset distance along bisector
	// cos(theta/2) where theta is the angle between normals
	cosHalfTheta := (nx1*bx + ny1*by)
	if math.Abs(cosHalfTheta) < Epsilon {
		cosHalfTheta = Epsilon
	}

	offsetDist := distance / cosHalfTheta

	// Limit the offset distance for sharp angles (miter limit)
	maxOffset := math.Abs(distance) * params.MiterLimit
	if math.Abs(offsetDist) > maxOffset {
		if offsetDist > 0 {
			offsetDist = maxOffset
		} else {
			offsetDist = -maxOffset
		}
	}

	return &orb.Point{
		curr[0] + bx*offsetDist,
		curr[1] + by*offsetDist,
	}
}

// reverseRingInPlace reverses the order of points in a ring in place.
func reverseRingInPlace(ring orb.Ring) {
	for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
		ring[i], ring[j] = ring[j], ring[i]
	}
}

// BufferMultiPoint creates a buffer around multiple points and unions overlapping buffers.
func BufferMultiPoint(mp orb.MultiPoint, distance float64, params BufferParams) orb.Geometry {
	if len(mp) == 0 || distance <= 0 {
		return orb.Polygon{}
	}

	if len(mp) == 1 {
		return BufferPoint(mp[0], distance, params)
	}

	// Buffer each point
	var buffers []orb.Polygon
	for _, p := range mp {
		buffered := BufferPoint(p, distance, params)
		if len(buffered) > 0 && len(buffered[0]) >= 4 {
			buffers = append(buffers, buffered)
		}
	}

	if len(buffers) == 0 {
		return orb.Polygon{}
	}
	if len(buffers) == 1 {
		return buffers[0]
	}

	// Group overlapping buffers and union each group
	var groups [][]orb.Polygon
	used := make([]bool, len(buffers))

	for i := 0; i < len(buffers); i++ {
		if used[i] {
			continue
		}

		// Start a new group with this buffer
		group := []orb.Polygon{buffers[i]}
		used[i] = true

		// Find all buffers that overlap with any buffer in this group
		changed := true
		for changed {
			changed = false
			for j := i + 1; j < len(buffers); j++ {
				if used[j] {
					continue
				}
				// Check if this buffer overlaps with any buffer in the current group
				for _, g := range group {
					if polygonsOverlap(buffers[j], g) {
						group = append(group, buffers[j])
						used[j] = true
						changed = true
						break
					}
				}
			}
		}

		groups = append(groups, group)
	}

	// Union each group and collect results
	var result orb.MultiPolygon
	for _, group := range groups {
		if len(group) == 1 {
			result = append(result, group[0])
		} else {
			// Union all polygons in the group
			unioned := group[0]
			for i := 1; i < len(group); i++ {
				unionedMulti := PolygonUnionMulti(unioned, group[i])
				if len(unionedMulti) == 1 {
					unioned = unionedMulti[0]
				} else if len(unionedMulti) > 1 {
					// If union produces multiple polygons, take the first
					unioned = unionedMulti[0]
				}
			}
			result = append(result, unioned)
		}
	}

	if len(result) == 1 {
		return result[0]
	}
	return result
}

// BufferMultiLineString creates a buffer around multiple linestrings and unions them.
func BufferMultiLineString(mls orb.MultiLineString, distance float64, params BufferParams) orb.Geometry {
	if len(mls) == 0 || distance <= 0 {
		return orb.Polygon{}
	}

	if len(mls) == 1 {
		return BufferLineString(mls[0], distance, params)
	}

	var result orb.MultiPolygon
	for _, ls := range mls {
		buffered := BufferLineString(ls, distance, params)
		if len(buffered) > 0 && len(buffered[0]) >= 4 {
			result = append(result, buffered)
		}
	}

	return unionMultiPolygon(result)
}

// BufferMultiPolygon creates a buffer around multiple polygons and unions them.
func BufferMultiPolygon(mp orb.MultiPolygon, distance float64, params BufferParams) orb.Geometry {
	if len(mp) == 0 {
		return orb.Polygon{}
	}

	if distance == 0 {
		if len(mp) == 1 {
			return mp[0]
		}
		return mp
	}

	var result orb.MultiPolygon
	for _, p := range mp {
		buffered := BufferPolygon(p, distance, params)
		if len(buffered) > 0 && len(buffered[0]) >= 4 {
			result = append(result, buffered)
		}
	}

	if distance > 0 {
		return unionMultiPolygon(result)
	}

	// For negative distance, don't union (polygons may become separate)
	if len(result) == 0 {
		return orb.Polygon{}
	}
	if len(result) == 1 {
		return result[0]
	}
	return result
}

// unionMultiPolygon unions all polygons in a MultiPolygon.
func unionMultiPolygon(mp orb.MultiPolygon) orb.Geometry {
	if len(mp) == 0 {
		return orb.Polygon{}
	}
	if len(mp) == 1 {
		return mp[0]
	}

	// Iteratively union polygons
	result := mp[0]
	for i := 1; i < len(mp); i++ {
		unioned := PolygonUnionMulti(result, mp[i])
		if len(unioned) == 1 {
			result = unioned[0]
		} else if len(unioned) > 1 {
			// Keep trying to union with remaining
			// For now, just take the first and continue
			result = unioned[0]
		}
	}

	return result
}

