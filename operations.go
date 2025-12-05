package orboperations

import (
	"github.com/paulmach/orb"
	"github.com/tingold/orb-operations/internal"
)

type booleanOpType int

const (
	unionOp booleanOpType = iota
	intersectionOp
	differenceOp
)

// booleanOperation performs a boolean operation on two geometries.
func booleanOperation(geom1, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	// Handle nil geometries
	if geom1 == nil {
		if op == unionOp {
			return geom2
		}
		return nil
	}
	if geom2 == nil {
		if op == unionOp || op == differenceOp {
			return geom1
		}
		return nil
	}

	// Type switch on first geometry
	switch g1 := geom1.(type) {
	case orb.Point:
		return handlePointOp(g1, geom2, op)
	case orb.MultiPoint:
		return handleMultiPointOp(g1, geom2, op)
	case orb.LineString:
		return handleLineStringOp(g1, geom2, op)
	case orb.MultiLineString:
		return handleMultiLineStringOp(g1, geom2, op)
	case orb.Polygon:
		return handlePolygonOp(g1, geom2, op)
	case orb.MultiPolygon:
		return handleMultiPolygonOp(g1, geom2, op)
	default:
		return nil
	}
}

func handlePointOp(p1 orb.Point, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		switch op {
		case unionOp:
			return normalizePointResult(internal.PointUnion(p1, g2))
		case intersectionOp:
			return normalizePointResult(internal.PointIntersection(p1, g2))
		case differenceOp:
			return normalizePointResult(internal.PointDifference(p1, g2))
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			return normalizePointResult(internal.MultiPointUnion(orb.MultiPoint{p1}, g2))
		case intersectionOp:
			if internal.PointInMultiPoint(p1, g2) {
				return p1
			}
			return orb.MultiPoint{}
		case differenceOp:
			if internal.PointInMultiPoint(p1, g2) {
				return orb.MultiPoint{}
			}
			return p1
		}
	case orb.LineString:
		return handlePointLineStringOp(p1, g2, op)
	case orb.MultiLineString:
		return handlePointMultiLineStringOp(p1, g2, op)
	case orb.Polygon:
		return handlePointPolygonOp(p1, g2, op)
	case orb.MultiPolygon:
		return handlePointMultiPolygonOp(p1, g2, op)
	}
	return nil
}

// handlePointLineStringOp handles Point-LineString operations.
func handlePointLineStringOp(p orb.Point, ls orb.LineString, op booleanOpType) orb.Geometry {
	onLine := pointOnLineString(p, ls)

	switch op {
	case unionOp:
		if onLine {
			// Point is on line, absorbed by line
			return ls
		}
		// Point is disjoint from line, return both
		return orb.Collection{ls, p}
	case intersectionOp:
		if onLine {
			return p
		}
		return orb.MultiPoint{}
	case differenceOp:
		if onLine {
			return orb.MultiPoint{}
		}
		return p
	}
	return nil
}

// handlePointMultiLineStringOp handles Point-MultiLineString operations.
func handlePointMultiLineStringOp(p orb.Point, mls orb.MultiLineString, op booleanOpType) orb.Geometry {
	onLine := false
	for _, ls := range mls {
		if pointOnLineString(p, ls) {
			onLine = true
			break
		}
	}

	switch op {
	case unionOp:
		if onLine {
			return mls
		}
		// Point is disjoint from all lines, return Collection with flattened lines
		var coll orb.Collection
		for _, ls := range mls {
			coll = append(coll, ls)
		}
		coll = append(coll, p)
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if onLine {
			return p
		}
		return orb.MultiPoint{}
	case differenceOp:
		if onLine {
			return orb.MultiPoint{}
		}
		return p
	}
	return nil
}

// handlePointPolygonOp handles Point-Polygon operations.
func handlePointPolygonOp(p orb.Point, poly orb.Polygon, op booleanOpType) orb.Geometry {
	inPoly := internal.PointInPolygon(p, poly)
	onBoundary := pointOnPolygonBoundary(p, poly)

	switch op {
	case unionOp:
		if inPoly || onBoundary {
			// Point is inside or on boundary, absorbed by polygon
			return poly
		}
		// Point is outside polygon, return both
		return orb.Collection{poly, p}
	case intersectionOp:
		if inPoly || onBoundary {
			return p
		}
		return orb.MultiPoint{}
	case differenceOp:
		if inPoly || onBoundary {
			return orb.MultiPoint{}
		}
		return p
	}
	return nil
}

// handlePointMultiPolygonOp handles Point-MultiPolygon operations.
func handlePointMultiPolygonOp(p orb.Point, mp orb.MultiPolygon, op booleanOpType) orb.Geometry {
	inAny := false
	for _, poly := range mp {
		if internal.PointInPolygon(p, poly) || pointOnPolygonBoundary(p, poly) {
			inAny = true
			break
		}
	}

	switch op {
	case unionOp:
		if inAny {
			return mp
		}
		// Point is outside all polygons, return Collection with flattened polygons
		var coll orb.Collection
		for _, poly := range mp {
			coll = append(coll, poly)
		}
		coll = append(coll, p)
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if inAny {
			return p
		}
		return orb.MultiPoint{}
	case differenceOp:
		if inAny {
			return orb.MultiPoint{}
		}
		return p
	}
	return nil
}

func handleMultiPointOp(mp1 orb.MultiPoint, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		// Delegate to point handler but swap order for some operations
		switch op {
		case unionOp:
			return normalizePointResult(internal.MultiPointUnion(mp1, orb.MultiPoint{g2}))
		case intersectionOp:
			if internal.PointInMultiPoint(g2, mp1) {
				return g2
			}
			return orb.MultiPoint{}
		case differenceOp:
			return normalizePointResult(internal.MultiPointDifference(mp1, orb.MultiPoint{g2}))
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			return normalizePointResult(internal.MultiPointUnion(mp1, g2))
		case intersectionOp:
			return normalizePointResult(internal.MultiPointIntersection(mp1, g2))
		case differenceOp:
			return normalizePointResult(internal.MultiPointDifference(mp1, g2))
		}
	case orb.LineString:
		return handleMultiPointLineStringOp(mp1, g2, op)
	case orb.MultiLineString:
		return handleMultiPointMultiLineStringOp(mp1, g2, op)
	case orb.Polygon:
		return handleMultiPointPolygonOp(mp1, g2, op)
	case orb.MultiPolygon:
		return handleMultiPointMultiPolygonOp(mp1, g2, op)
	}
	return nil
}

// handleMultiPointLineStringOp handles MultiPoint-LineString operations.
func handleMultiPointLineStringOp(mp orb.MultiPoint, ls orb.LineString, op booleanOpType) orb.Geometry {
	var onLine, notOnLine orb.MultiPoint
	for _, p := range mp {
		if pointOnLineString(p, ls) {
			onLine = append(onLine, p)
		} else {
			notOnLine = append(notOnLine, p)
		}
	}

	switch op {
	case unionOp:
		if len(notOnLine) == 0 {
			// All points are on the line
			return ls
		}
		// Some points are not on the line, return Collection
		var coll orb.Collection
		coll = append(coll, ls)
		for _, p := range notOnLine {
			coll = append(coll, p)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if len(onLine) == 0 {
			return orb.MultiPoint{}
		}
		if len(onLine) == 1 {
			return onLine[0]
		}
		return onLine
	case differenceOp:
		if len(notOnLine) == 0 {
			return orb.MultiPoint{}
		}
		if len(notOnLine) == 1 {
			return notOnLine[0]
		}
		return notOnLine
	}
	return nil
}

// handleMultiPointMultiLineStringOp handles MultiPoint-MultiLineString operations.
func handleMultiPointMultiLineStringOp(mp orb.MultiPoint, mls orb.MultiLineString, op booleanOpType) orb.Geometry {
	var onLine, notOnLine orb.MultiPoint
	for _, p := range mp {
		found := false
		for _, ls := range mls {
			if pointOnLineString(p, ls) {
				found = true
				break
			}
		}
		if found {
			onLine = append(onLine, p)
		} else {
			notOnLine = append(notOnLine, p)
		}
	}

	switch op {
	case unionOp:
		if len(notOnLine) == 0 {
			return mls
		}
		// Some points are not on any line, return Collection with flattened lines
		var coll orb.Collection
		for _, ls := range mls {
			coll = append(coll, ls)
		}
		for _, p := range notOnLine {
			coll = append(coll, p)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if len(onLine) == 0 {
			return orb.MultiPoint{}
		}
		if len(onLine) == 1 {
			return onLine[0]
		}
		return onLine
	case differenceOp:
		if len(notOnLine) == 0 {
			return orb.MultiPoint{}
		}
		if len(notOnLine) == 1 {
			return notOnLine[0]
		}
		return notOnLine
	}
	return nil
}

// handleMultiPointPolygonOp handles MultiPoint-Polygon operations.
func handleMultiPointPolygonOp(mp orb.MultiPoint, poly orb.Polygon, op booleanOpType) orb.Geometry {
	var inside, outside orb.MultiPoint
	for _, p := range mp {
		if internal.PointInPolygon(p, poly) || pointOnPolygonBoundary(p, poly) {
			inside = append(inside, p)
		} else {
			outside = append(outside, p)
		}
	}

	switch op {
	case unionOp:
		if len(outside) == 0 {
			return poly
		}
		// Some points are outside the polygon, return Collection
		var coll orb.Collection
		coll = append(coll, poly)
		for _, p := range outside {
			coll = append(coll, p)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if len(inside) == 0 {
			return orb.MultiPoint{}
		}
		if len(inside) == 1 {
			return inside[0]
		}
		return inside
	case differenceOp:
		if len(outside) == 0 {
			return orb.MultiPoint{}
		}
		if len(outside) == 1 {
			return outside[0]
		}
		return outside
	}
	return nil
}

// handleMultiPointMultiPolygonOp handles MultiPoint-MultiPolygon operations.
func handleMultiPointMultiPolygonOp(mp orb.MultiPoint, mpoly orb.MultiPolygon, op booleanOpType) orb.Geometry {
	var inside, outside orb.MultiPoint
	for _, p := range mp {
		found := false
		for _, poly := range mpoly {
			if internal.PointInPolygon(p, poly) || pointOnPolygonBoundary(p, poly) {
				found = true
				break
			}
		}
		if found {
			inside = append(inside, p)
		} else {
			outside = append(outside, p)
		}
	}

	switch op {
	case unionOp:
		if len(outside) == 0 {
			return mpoly
		}
		// Some points are outside all polygons, return Collection with flattened polygons
		var coll orb.Collection
		for _, poly := range mpoly {
			coll = append(coll, poly)
		}
		for _, p := range outside {
			coll = append(coll, p)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if len(inside) == 0 {
			return orb.MultiPoint{}
		}
		if len(inside) == 1 {
			return inside[0]
		}
		return inside
	case differenceOp:
		if len(outside) == 0 {
			return orb.MultiPoint{}
		}
		if len(outside) == 1 {
			return outside[0]
		}
		return outside
	}
	return nil
}

func handleLineStringOp(ls1 orb.LineString, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		// Point-Line with swapped operands for difference
		switch op {
		case unionOp:
			if pointOnLineString(g2, ls1) {
				return ls1
			}
			// Point is not on line, return Collection
			return orb.Collection{ls1, g2}
		case intersectionOp:
			if pointOnLineString(g2, ls1) {
				return g2
			}
			return orb.MultiPoint{}
		case differenceOp:
			// Line - Point = Line (point doesn't affect line)
			return ls1
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			var notOnLine orb.MultiPoint
			for _, p := range g2 {
				if !pointOnLineString(p, ls1) {
					notOnLine = append(notOnLine, p)
				}
			}
			if len(notOnLine) == 0 {
				return ls1
			}
			// Some points are not on line, return Collection
			var coll orb.Collection
			coll = append(coll, ls1)
			for _, p := range notOnLine {
				coll = append(coll, p)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var onLine orb.MultiPoint
			for _, p := range g2 {
				if pointOnLineString(p, ls1) {
					onLine = append(onLine, p)
				}
			}
			if len(onLine) == 0 {
				return orb.MultiPoint{}
			}
			if len(onLine) == 1 {
				return onLine[0]
			}
			return onLine
		case differenceOp:
			return ls1
		}
	case orb.LineString:
		switch op {
		case unionOp:
			return normalizeLineStringResult(internal.LineStringUnion(ls1, g2))
		case intersectionOp:
			return internal.LineStringIntersectionGeom(ls1, g2)
		case differenceOp:
			return normalizeLineStringResult(internal.LineStringDifference(ls1, g2))
		}
	case orb.MultiLineString:
		return handleLineStringMultiLineStringOp(ls1, g2, op)
	case orb.Polygon:
		return handleLineStringPolygonOp(ls1, g2, op)
	case orb.MultiPolygon:
		return handleLineStringMultiPolygonOp(ls1, g2, op)
	}
	return nil
}

// handleLineStringMultiLineStringOp handles LineString-MultiLineString operations.
func handleLineStringMultiLineStringOp(ls orb.LineString, mls orb.MultiLineString, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		result := orb.MultiLineString{ls}
		for _, ls2 := range mls {
			union := internal.LineStringUnion(ls, ls2)
			result = append(result, union...)
		}
		return normalizeLineStringResult(result)
	case intersectionOp:
		var result orb.MultiLineString
		for _, ls2 := range mls {
			inter := internal.LineStringIntersectionGeom(ls, ls2)
			if inter != nil {
				switch g := inter.(type) {
				case orb.LineString:
					result = append(result, g)
				case orb.MultiLineString:
					result = append(result, g...)
				}
			}
		}
		if len(result) == 0 {
			return nil
		}
		return normalizeLineStringResult(result)
	case differenceOp:
		current := orb.MultiLineString{ls}
		for _, ls2 := range mls {
			var newCurrent orb.MultiLineString
			for _, seg := range current {
				diff := internal.LineStringDifference(seg, ls2)
				newCurrent = append(newCurrent, diff...)
			}
			current = newCurrent
		}
		if len(current) == 0 {
			return orb.MultiLineString{}
		}
		return normalizeLineStringResult(current)
	}
	return nil
}

// handleLineStringPolygonOp handles LineString-Polygon operations.
func handleLineStringPolygonOp(ls orb.LineString, poly orb.Polygon, op booleanOpType) orb.Geometry {
	if len(poly) == 0 || len(poly[0]) < 3 {
		if op == unionOp || op == differenceOp {
			return ls
		}
		return nil
	}

	// Clip line string to polygon
	inside, outside := clipLineToPolygon(ls, poly)

	switch op {
	case unionOp:
		if len(outside) == 0 {
			// Line is entirely inside or on boundary of polygon
			return poly
		}
		// Line has parts outside the polygon, return Collection
		var coll orb.Collection
		coll = append(coll, poly)
		for _, l := range outside {
			coll = append(coll, l)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		if len(inside) == 0 {
			return orb.MultiLineString{}
		}
		if len(inside) == 1 {
			return inside[0]
		}
		return inside
	case differenceOp:
		if len(outside) == 0 {
			return orb.MultiLineString{}
		}
		if len(outside) == 1 {
			return outside[0]
		}
		return outside
	}
	return nil
}

// handleLineStringMultiPolygonOp handles LineString-MultiPolygon operations.
func handleLineStringMultiPolygonOp(ls orb.LineString, mp orb.MultiPolygon, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		// Calculate line parts outside all polygons
		current := orb.MultiLineString{ls}
		for _, poly := range mp {
			var newCurrent orb.MultiLineString
			for _, seg := range current {
				_, outside := clipLineToPolygon(seg, poly)
				newCurrent = append(newCurrent, outside...)
			}
			current = newCurrent
		}
		if len(current) == 0 {
			// Line is entirely inside the polygons
			return mp
		}
		// Line has parts outside all polygons, return Collection with flattened polygons
		var coll orb.Collection
		for _, poly := range mp {
			coll = append(coll, poly)
		}
		for _, l := range current {
			coll = append(coll, l)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		var result orb.MultiLineString
		for _, poly := range mp {
			inside, _ := clipLineToPolygon(ls, poly)
			result = append(result, inside...)
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case differenceOp:
		current := orb.MultiLineString{ls}
		for _, poly := range mp {
			var newCurrent orb.MultiLineString
			for _, seg := range current {
				_, outside := clipLineToPolygon(seg, poly)
				newCurrent = append(newCurrent, outside...)
			}
			current = newCurrent
		}
		if len(current) == 0 {
			return orb.MultiLineString{}
		}
		return current
	}
	return nil
}

func handleMultiLineStringOp(mls1 orb.MultiLineString, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		switch op {
		case unionOp:
			for _, ls := range mls1 {
				if pointOnLineString(g2, ls) {
					return mls1
				}
			}
			// Point is not on any line, return Collection with flattened lines
			var coll orb.Collection
			for _, ls := range mls1 {
				coll = append(coll, ls)
			}
			coll = append(coll, g2)
			return normalizeCollectionResult(coll)
		case intersectionOp:
			for _, ls := range mls1 {
				if pointOnLineString(g2, ls) {
					return g2
				}
			}
			return orb.MultiPoint{}
		case differenceOp:
			return mls1
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			var notOnLine orb.MultiPoint
			for _, p := range g2 {
				found := false
				for _, ls := range mls1 {
					if pointOnLineString(p, ls) {
						found = true
						break
					}
				}
				if !found {
					notOnLine = append(notOnLine, p)
				}
			}
			if len(notOnLine) == 0 {
				return mls1
			}
			// Some points are not on any line, return Collection with flattened lines
			var coll orb.Collection
			for _, ls := range mls1 {
				coll = append(coll, ls)
			}
			for _, p := range notOnLine {
				coll = append(coll, p)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var onLine orb.MultiPoint
			for _, p := range g2 {
				for _, ls := range mls1 {
					if pointOnLineString(p, ls) {
						onLine = append(onLine, p)
						break
					}
				}
			}
			if len(onLine) == 0 {
				return orb.MultiPoint{}
			}
			return onLine
		case differenceOp:
			return mls1
		}
	case orb.LineString:
		// Swap operands for some operations
		return handleLineStringMultiLineStringOp(g2, mls1, swapOp(op))
	case orb.MultiLineString:
		return handleMultiLineStringMultiLineStringOp(mls1, g2, op)
	case orb.Polygon:
		return handleMultiLineStringPolygonOp(mls1, g2, op)
	case orb.MultiPolygon:
		return handleMultiLineStringMultiPolygonOp(mls1, g2, op)
	}
	return nil
}

// handleMultiLineStringMultiLineStringOp handles MultiLineString-MultiLineString operations.
func handleMultiLineStringMultiLineStringOp(mls1, mls2 orb.MultiLineString, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		result := make(orb.MultiLineString, 0, len(mls1)+len(mls2))
		result = append(result, mls1...)
		result = append(result, mls2...)
		return normalizeLineStringResult(result)
	case intersectionOp:
		var result orb.MultiLineString
		for _, ls1 := range mls1 {
			for _, ls2 := range mls2 {
				inter := internal.LineStringIntersectionGeom(ls1, ls2)
				if inter != nil {
					switch g := inter.(type) {
					case orb.LineString:
						result = append(result, g)
					case orb.MultiLineString:
						result = append(result, g...)
					}
				}
			}
		}
		if len(result) == 0 {
			return nil
		}
		return normalizeLineStringResult(result)
	case differenceOp:
		var result orb.MultiLineString
		for _, ls1 := range mls1 {
			current := orb.MultiLineString{ls1}
			for _, ls2 := range mls2 {
				var newCurrent orb.MultiLineString
				for _, seg := range current {
					diff := internal.LineStringDifference(seg, ls2)
					newCurrent = append(newCurrent, diff...)
				}
				current = newCurrent
			}
			result = append(result, current...)
		}
		if len(result) == 0 {
			return orb.MultiLineString{}
		}
		return normalizeLineStringResult(result)
	}
	return nil
}

// handleMultiLineStringPolygonOp handles MultiLineString-Polygon operations.
func handleMultiLineStringPolygonOp(mls orb.MultiLineString, poly orb.Polygon, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		// Calculate line parts outside the polygon
		var outsideLines orb.MultiLineString
		for _, ls := range mls {
			_, outside := clipLineToPolygon(ls, poly)
			outsideLines = append(outsideLines, outside...)
		}
		if len(outsideLines) == 0 {
			// All lines are inside the polygon
			return poly
		}
		// Some line parts are outside, return Collection
		var coll orb.Collection
		coll = append(coll, poly)
		for _, l := range outsideLines {
			coll = append(coll, l)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		var result orb.MultiLineString
		for _, ls := range mls {
			inside, _ := clipLineToPolygon(ls, poly)
			result = append(result, inside...)
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case differenceOp:
		var result orb.MultiLineString
		for _, ls := range mls {
			_, outside := clipLineToPolygon(ls, poly)
			result = append(result, outside...)
		}
		if len(result) == 0 {
			return orb.MultiLineString{}
		}
		return result
	}
	return nil
}

// handleMultiLineStringMultiPolygonOp handles MultiLineString-MultiPolygon operations.
func handleMultiLineStringMultiPolygonOp(mls orb.MultiLineString, mp orb.MultiPolygon, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		// Calculate line parts outside all polygons
		var outsideLines orb.MultiLineString
		for _, ls := range mls {
			current := orb.MultiLineString{ls}
			for _, poly := range mp {
				var newCurrent orb.MultiLineString
				for _, seg := range current {
					_, outside := clipLineToPolygon(seg, poly)
					newCurrent = append(newCurrent, outside...)
				}
				current = newCurrent
			}
			outsideLines = append(outsideLines, current...)
		}
		if len(outsideLines) == 0 {
			// All lines are inside the polygons
			return mp
		}
		// Some line parts are outside, return Collection with flattened polygons
		var coll orb.Collection
		for _, poly := range mp {
			coll = append(coll, poly)
		}
		for _, l := range outsideLines {
			coll = append(coll, l)
		}
		return normalizeCollectionResult(coll)
	case intersectionOp:
		var result orb.MultiLineString
		for _, ls := range mls {
			for _, poly := range mp {
				inside, _ := clipLineToPolygon(ls, poly)
				result = append(result, inside...)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case differenceOp:
		var result orb.MultiLineString
		for _, ls := range mls {
			current := orb.MultiLineString{ls}
			for _, poly := range mp {
				var newCurrent orb.MultiLineString
				for _, seg := range current {
					_, outside := clipLineToPolygon(seg, poly)
					newCurrent = append(newCurrent, outside...)
				}
				current = newCurrent
			}
			result = append(result, current...)
		}
		if len(result) == 0 {
			return orb.MultiLineString{}
		}
		return result
	}
	return nil
}

func handlePolygonOp(p1 orb.Polygon, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		// Polygon - Point operations
		switch op {
		case unionOp:
			if internal.PointInPolygon(g2, p1) || pointOnPolygonBoundary(g2, p1) {
				return p1
			}
			// Point is outside polygon, return Collection
			return orb.Collection{p1, g2}
		case intersectionOp:
			if internal.PointInPolygon(g2, p1) || pointOnPolygonBoundary(g2, p1) {
				return g2
			}
			return orb.MultiPoint{}
		case differenceOp:
			return p1
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			var outside orb.MultiPoint
			for _, p := range g2 {
				if !internal.PointInPolygon(p, p1) && !pointOnPolygonBoundary(p, p1) {
					outside = append(outside, p)
				}
			}
			if len(outside) == 0 {
				return p1
			}
			// Some points are outside, return Collection
			var coll orb.Collection
			coll = append(coll, p1)
			for _, p := range outside {
				coll = append(coll, p)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var inside orb.MultiPoint
			for _, p := range g2 {
				if internal.PointInPolygon(p, p1) || pointOnPolygonBoundary(p, p1) {
					inside = append(inside, p)
				}
			}
			if len(inside) == 0 {
				return orb.MultiPoint{}
			}
			return inside
		case differenceOp:
			return p1
		}
	case orb.LineString:
		// Polygon - Line
		switch op {
		case unionOp:
			_, outside := clipLineToPolygon(g2, p1)
			if len(outside) == 0 {
				return p1
			}
			// Line has parts outside, return Collection
			var coll orb.Collection
			coll = append(coll, p1)
			for _, l := range outside {
				coll = append(coll, l)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			inside, _ := clipLineToPolygon(g2, p1)
			if len(inside) == 0 {
				return nil
			}
			if len(inside) == 1 {
				return inside[0]
			}
			return inside
		case differenceOp:
			return p1
		}
	case orb.MultiLineString:
		switch op {
		case unionOp:
			var outsideLines orb.MultiLineString
			for _, ls := range g2 {
				_, outside := clipLineToPolygon(ls, p1)
				outsideLines = append(outsideLines, outside...)
			}
			if len(outsideLines) == 0 {
				return p1
			}
			// Some lines are outside, return Collection
			var coll orb.Collection
			coll = append(coll, p1)
			for _, l := range outsideLines {
				coll = append(coll, l)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var result orb.MultiLineString
			for _, ls := range g2 {
				inside, _ := clipLineToPolygon(ls, p1)
				result = append(result, inside...)
			}
			if len(result) == 0 {
				return nil
			}
			return result
		case differenceOp:
			return p1
		}
	case orb.Polygon:
		switch op {
		case unionOp:
			return normalizePolygonResult(internal.PolygonUnionMulti(p1, g2))
		case intersectionOp:
			// Use PolygonIntersectionGeom to handle dimension collapse
			result := internal.PolygonIntersectionGeom(p1, g2)
			if result == nil {
				return orb.Polygon{}
			}
			return result
		case differenceOp:
			return normalizePolygonResult(internal.PolygonDifferenceMulti(p1, g2))
		}
	case orb.MultiPolygon:
		return handlePolygonMultiPolygonOp(p1, g2, op)
	}
	return nil
}

// handlePolygonMultiPolygonOp handles Polygon-MultiPolygon operations.
func handlePolygonMultiPolygonOp(p orb.Polygon, mp orb.MultiPolygon, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		// Collect all polygons and merge overlapping ones
		allPolygons := make(orb.MultiPolygon, 0, 1+len(mp))
		allPolygons = append(allPolygons, p)
		allPolygons = append(allPolygons, mp...)
		return normalizePolygonResult(mergeOverlappingPolygons(allPolygons))
	case intersectionOp:
		var results orb.MultiPolygon
		for _, p2 := range mp {
			inter := internal.PolygonIntersectionMulti(p, p2)
			results = append(results, inter...)
		}
		return normalizePolygonResult(results)
	case differenceOp:
		var result orb.MultiPolygon = orb.MultiPolygon{p}
		for _, p2 := range mp {
			var newResult orb.MultiPolygon
			for _, r := range result {
				diff := internal.PolygonDifferenceMulti(r, p2)
				newResult = append(newResult, diff...)
			}
			result = newResult
			if len(result) == 0 {
				return orb.Polygon{}
			}
		}
		return normalizePolygonResult(result)
	}
	return nil
}

func handleMultiPolygonOp(mp1 orb.MultiPolygon, geom2 orb.Geometry, op booleanOpType) orb.Geometry {
	switch g2 := geom2.(type) {
	case orb.Point:
		switch op {
		case unionOp:
			for _, poly := range mp1 {
				if internal.PointInPolygon(g2, poly) || pointOnPolygonBoundary(g2, poly) {
					return mp1
				}
			}
			// Point is outside all polygons, return Collection with flattened polygons
			var coll orb.Collection
			for _, poly := range mp1 {
				coll = append(coll, poly)
			}
			coll = append(coll, g2)
			return normalizeCollectionResult(coll)
		case intersectionOp:
			for _, poly := range mp1 {
				if internal.PointInPolygon(g2, poly) || pointOnPolygonBoundary(g2, poly) {
					return g2
				}
			}
			return orb.MultiPoint{}
		case differenceOp:
			return mp1
		}
	case orb.MultiPoint:
		switch op {
		case unionOp:
			var outside orb.MultiPoint
			for _, p := range g2 {
				inAny := false
				for _, poly := range mp1 {
					if internal.PointInPolygon(p, poly) || pointOnPolygonBoundary(p, poly) {
						inAny = true
						break
					}
				}
				if !inAny {
					outside = append(outside, p)
				}
			}
			if len(outside) == 0 {
				return mp1
			}
			// Some points are outside, return Collection with flattened polygons
			var coll orb.Collection
			for _, poly := range mp1 {
				coll = append(coll, poly)
			}
			for _, p := range outside {
				coll = append(coll, p)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var inside orb.MultiPoint
			for _, p := range g2 {
				for _, poly := range mp1 {
					if internal.PointInPolygon(p, poly) || pointOnPolygonBoundary(p, poly) {
						inside = append(inside, p)
						break
					}
				}
			}
			if len(inside) == 0 {
				return orb.MultiPoint{}
			}
			return inside
		case differenceOp:
			return mp1
		}
	case orb.LineString:
		switch op {
		case unionOp:
			// Calculate line parts outside all polygons
			current := orb.MultiLineString{g2}
			for _, poly := range mp1 {
				var newCurrent orb.MultiLineString
				for _, seg := range current {
					_, outside := clipLineToPolygon(seg, poly)
					newCurrent = append(newCurrent, outside...)
				}
				current = newCurrent
			}
			if len(current) == 0 {
				return mp1
			}
			// Line has parts outside all polygons, return Collection with flattened polygons
			var coll orb.Collection
			for _, poly := range mp1 {
				coll = append(coll, poly)
			}
			for _, l := range current {
				coll = append(coll, l)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var result orb.MultiLineString
			for _, poly := range mp1 {
				inside, _ := clipLineToPolygon(g2, poly)
				result = append(result, inside...)
			}
			if len(result) == 0 {
				return nil
			}
			return result
		case differenceOp:
			return mp1
		}
	case orb.MultiLineString:
		switch op {
		case unionOp:
			// Calculate line parts outside all polygons
			var outsideLines orb.MultiLineString
			for _, ls := range g2 {
				current := orb.MultiLineString{ls}
				for _, poly := range mp1 {
					var newCurrent orb.MultiLineString
					for _, seg := range current {
						_, outside := clipLineToPolygon(seg, poly)
						newCurrent = append(newCurrent, outside...)
					}
					current = newCurrent
				}
				outsideLines = append(outsideLines, current...)
			}
			if len(outsideLines) == 0 {
				return mp1
			}
			// Some lines are outside, return Collection with flattened polygons
			var coll orb.Collection
			for _, poly := range mp1 {
				coll = append(coll, poly)
			}
			for _, l := range outsideLines {
				coll = append(coll, l)
			}
			return normalizeCollectionResult(coll)
		case intersectionOp:
			var result orb.MultiLineString
			for _, ls := range g2 {
				for _, poly := range mp1 {
					inside, _ := clipLineToPolygon(ls, poly)
					result = append(result, inside...)
				}
			}
			if len(result) == 0 {
				return nil
			}
			return result
		case differenceOp:
			return mp1
		}
	case orb.Polygon:
		return handleMultiPolygonPolygonOp(mp1, g2, op)
	case orb.MultiPolygon:
		return handleMultiPolygonMultiPolygonOp(mp1, g2, op)
	}
	return nil
}

// handleMultiPolygonPolygonOp handles MultiPolygon-Polygon operations.
func handleMultiPolygonPolygonOp(mp orb.MultiPolygon, p orb.Polygon, op booleanOpType) orb.Geometry {
	switch op {
	case unionOp:
		// Collect all polygons and merge overlapping ones
		allPolygons := make(orb.MultiPolygon, 0, len(mp)+1)
		allPolygons = append(allPolygons, mp...)
		allPolygons = append(allPolygons, p)
		return normalizePolygonResult(mergeOverlappingPolygons(allPolygons))
	case intersectionOp:
		var results orb.MultiPolygon
		for _, p2 := range mp {
			inter := internal.PolygonIntersectionMulti(p2, p)
			results = append(results, inter...)
		}
		return normalizePolygonResult(results)
	case differenceOp:
		var results orb.MultiPolygon
		for _, p2 := range mp {
			diff := internal.PolygonDifferenceMulti(p2, p)
			results = append(results, diff...)
		}
		return normalizePolygonResult(results)
	}
	return nil
}

// handleMultiPolygonMultiPolygonOp handles MultiPolygon-MultiPolygon operations.
func handleMultiPolygonMultiPolygonOp(mp1, mp2 orb.MultiPolygon, op booleanOpType) orb.Geometry {
	if len(mp1) == 0 {
		if op == unionOp {
			return normalizePolygonResult(mp2)
		}
		return orb.Polygon{}
	}

	switch op {
	case unionOp:
		// Collect all polygons and merge overlapping ones
		allPolygons := make(orb.MultiPolygon, 0, len(mp1)+len(mp2))
		allPolygons = append(allPolygons, mp1...)
		allPolygons = append(allPolygons, mp2...)
		return normalizePolygonResult(mergeOverlappingPolygons(allPolygons))
	case intersectionOp:
		// Intersect each pair
		var results orb.MultiPolygon
		for _, p1 := range mp1 {
			for _, p2 := range mp2 {
				inter := internal.PolygonIntersectionMulti(p1, p2)
				results = append(results, inter...)
			}
		}
		return normalizePolygonResult(results)
	case differenceOp:
		// Subtract all of mp2 from each polygon in mp1
		var results orb.MultiPolygon
		for _, p1 := range mp1 {
			current := orb.MultiPolygon{p1}
			for _, p2 := range mp2 {
				var newCurrent orb.MultiPolygon
				for _, c := range current {
					diff := internal.PolygonDifferenceMulti(c, p2)
					newCurrent = append(newCurrent, diff...)
				}
				current = newCurrent
				if len(current) == 0 {
					break
				}
			}
			results = append(results, current...)
		}
		return normalizePolygonResult(results)
	}
	return nil
}

// mergeOverlappingPolygons merges overlapping polygons into a single result.
// It iteratively unions overlapping polygons until no more merges are possible.
func mergeOverlappingPolygons(polygons orb.MultiPolygon) orb.MultiPolygon {
	if len(polygons) <= 1 {
		return polygons
	}

	// Keep merging until no changes, with a safety limit
	maxIterations := len(polygons) * len(polygons)
	for iteration := 0; iteration < maxIterations; iteration++ {
		merged := false
		var result orb.MultiPolygon
		used := make([]bool, len(polygons))

		for i := 0; i < len(polygons); i++ {
			if used[i] {
				continue
			}

			current := polygons[i]
			used[i] = true

			// Try to merge with other polygons
			for j := i + 1; j < len(polygons); j++ {
				if used[j] {
					continue
				}

				// Check if polygons overlap or touch
				if polygonsOverlapOrTouch(current, polygons[j]) {
					// Union them
					union := internal.PolygonUnionMulti(current, polygons[j])
					if len(union) == 1 {
						// Successfully merged into one polygon
						current = union[0]
						used[j] = true
						merged = true
					}
					// If union returns > 1 polygon, they don't actually merge
					// Just mark as used and add the other polygon separately
					// Don't set merged=true as this isn't a real merge
				}
			}

			result = append(result, current)
		}

		// Add any polygons that weren't used (shouldn't happen, but safety)
		for i, u := range used {
			if !u {
				result = append(result, polygons[i])
			}
		}

		polygons = result
		if !merged || len(polygons) <= 1 {
			break
		}
	}

	return polygons
}

// polygonsOverlapOrTouch checks if two polygons overlap or touch.
func polygonsOverlapOrTouch(p1, p2 orb.Polygon) bool {
	if len(p1) == 0 || len(p2) == 0 {
		return false
	}

	b1 := p1.Bound()
	b2 := p2.Bound()

	// Quick bounding box check (with tolerance for touching)
	if b1.Max[0] < b2.Min[0]-internal.Epsilon || b2.Max[0] < b1.Min[0]-internal.Epsilon ||
		b1.Max[1] < b2.Min[1]-internal.Epsilon || b2.Max[1] < b1.Min[1]-internal.Epsilon {
		return false
	}

	// Check if any vertex of p1 is in or on boundary of p2
	for _, ring := range p1 {
		for _, pt := range ring {
			if internal.PointInPolygon(pt, p2) || pointOnPolygonBoundary(pt, p2) {
				return true
			}
		}
	}

	// Check if any vertex of p2 is in or on boundary of p1
	for _, ring := range p2 {
		for _, pt := range ring {
			if internal.PointInPolygon(pt, p1) || pointOnPolygonBoundary(pt, p1) {
				return true
			}
		}
	}

	// Check for edge intersections
	for _, ring1 := range p1 {
		for i := 0; i < len(ring1)-1; i++ {
			for _, ring2 := range p2 {
				for j := 0; j < len(ring2)-1; j++ {
					_, intersects := internal.LineSegmentIntersection(ring1[i], ring1[i+1], ring2[j], ring2[j+1])
					if intersects {
						return true
					}
				}
			}
		}
	}

	return false
}

// swapOp swaps the operands for asymmetric operations.
func swapOp(op booleanOpType) booleanOpType {
	// For difference, swapping operands changes the result
	// For union and intersection, the result is the same
	return op
}

// Helper functions

// pointOnLineString checks if a point lies on any segment of a line string.
func pointOnLineString(pt orb.Point, ls orb.LineString) bool {
	for i := 0; i < len(ls)-1; i++ {
		if internal.PointOnSegment(pt, ls[i], ls[i+1], internal.Epsilon) {
			return true
		}
	}
	return false
}

// pointOnPolygonBoundary checks if a point lies on the boundary of a polygon.
func pointOnPolygonBoundary(pt orb.Point, poly orb.Polygon) bool {
	for _, ring := range poly {
		for i := 0; i < len(ring)-1; i++ {
			if internal.PointOnSegment(pt, ring[i], ring[i+1], internal.Epsilon) {
				return true
			}
		}
	}
	return false
}

// clipLineToPolygon clips a line string to a polygon.
// Returns two MultiLineStrings: parts inside the polygon and parts outside.
func clipLineToPolygon(ls orb.LineString, poly orb.Polygon) (inside, outside orb.MultiLineString) {
	if len(ls) < 2 || len(poly) == 0 || len(poly[0]) < 3 {
		return nil, orb.MultiLineString{ls}
	}

	// Process each segment
	var currentInside, currentOutside orb.LineString

	for i := 0; i < len(ls)-1; i++ {
		p1, p2 := ls[i], ls[i+1]

		// Find intersection points with polygon boundary
		var intersections []struct {
			pt    orb.Point
			alpha float64
		}

		for _, ring := range poly {
			for j := 0; j < len(ring)-1; j++ {
				pt, alpha1, _, ok := segmentIntersection(p1, p2, ring[j], ring[j+1])
				if ok && alpha1 > internal.Epsilon && alpha1 < 1-internal.Epsilon {
					intersections = append(intersections, struct {
						pt    orb.Point
						alpha float64
					}{pt, alpha1})
				}
			}
		}

		if len(intersections) == 0 {
			// No intersections - entire segment is either inside or outside
			mid := orb.Point{(p1[0] + p2[0]) / 2, (p1[1] + p2[1]) / 2}
			if internal.PointInPolygon(mid, poly) {
				if len(currentOutside) > 0 {
					outside = append(outside, currentOutside)
					currentOutside = nil
				}
				currentInside = append(currentInside, p1)
			} else {
				if len(currentInside) > 0 {
					inside = append(inside, currentInside)
					currentInside = nil
				}
				currentOutside = append(currentOutside, p1)
			}
		} else {
			// Sort intersections by alpha
			for k := 0; k < len(intersections)-1; k++ {
				for l := k + 1; l < len(intersections); l++ {
					if intersections[l].alpha < intersections[k].alpha {
						intersections[k], intersections[l] = intersections[l], intersections[k]
					}
				}
			}

			// Process segments between intersection points
			current := p1
			currentAlpha := 0.0

			for _, inter := range intersections {
				// Segment from current to intersection
				mid := orb.Point{
					(current[0] + inter.pt[0]) / 2,
					(current[1] + inter.pt[1]) / 2,
				}
				if internal.PointInPolygon(mid, poly) {
					currentInside = append(currentInside, current, inter.pt)
					inside = append(inside, currentInside)
					currentInside = nil
				} else {
					currentOutside = append(currentOutside, current, inter.pt)
					outside = append(outside, currentOutside)
					currentOutside = nil
				}
				current = inter.pt
				currentAlpha = inter.alpha
			}

			// Final segment
			if currentAlpha < 1 {
				mid := orb.Point{
					(current[0] + p2[0]) / 2,
					(current[1] + p2[1]) / 2,
				}
				if internal.PointInPolygon(mid, poly) {
					currentInside = append(currentInside, current)
				} else {
					currentOutside = append(currentOutside, current)
				}
			}
		}
	}

	// Add final point and close any open segments
	lastPt := ls[len(ls)-1]
	if len(currentInside) > 0 {
		currentInside = append(currentInside, lastPt)
		if len(currentInside) >= 2 {
			inside = append(inside, currentInside)
		}
	}
	if len(currentOutside) > 0 {
		currentOutside = append(currentOutside, lastPt)
		if len(currentOutside) >= 2 {
			outside = append(outside, currentOutside)
		}
	}

	return inside, outside
}

// segmentIntersection finds the intersection of two line segments.
func segmentIntersection(p1, p2, p3, p4 orb.Point) (orb.Point, float64, float64, bool) {
	d1x := p2[0] - p1[0]
	d1y := p2[1] - p1[1]
	d2x := p4[0] - p3[0]
	d2y := p4[1] - p3[1]

	cross := d1x*d2y - d1y*d2x
	if cross > -internal.Epsilon && cross < internal.Epsilon {
		return orb.Point{}, 0, 0, false
	}

	dx := p3[0] - p1[0]
	dy := p3[1] - p1[1]

	alpha1 := (dx*d2y - dy*d2x) / cross
	alpha2 := (dx*d1y - dy*d1x) / cross

	if alpha1 >= -internal.Epsilon && alpha1 <= 1+internal.Epsilon &&
		alpha2 >= -internal.Epsilon && alpha2 <= 1+internal.Epsilon {
		pt := orb.Point{
			p1[0] + alpha1*d1x,
			p1[1] + alpha1*d1y,
		}
		return pt, alpha1, alpha2, true
	}

	return orb.Point{}, 0, 0, false
}

// Normalization helper functions

// normalizePointResult converts MultiPoint with 1 element to Point.
// Also handles empty point results by returning empty Point.
func normalizePointResult(geom orb.Geometry) orb.Geometry {
	if geom == nil {
		return nil
	}
	switch g := geom.(type) {
	case orb.MultiPoint:
		if len(g) == 0 {
			return orb.MultiPoint{}
		}
		if len(g) == 1 {
			return g[0]
		}
		return g
	default:
		return geom
	}
}

// normalizeLineStringResult converts MultiLineString with 1 element to LineString.
func normalizeLineStringResult(geom orb.Geometry) orb.Geometry {
	if geom == nil {
		return nil
	}
	switch g := geom.(type) {
	case orb.MultiLineString:
		if len(g) == 0 {
			return orb.MultiLineString{}
		}
		if len(g) == 1 {
			return g[0]
		}
		return g
	default:
		return geom
	}
}

// normalizePolygonResult handles Polygon vs MultiPolygon results properly.
func normalizePolygonResult(geom orb.Geometry) orb.Geometry {
	if geom == nil {
		return nil
	}
	switch g := geom.(type) {
	case orb.MultiPolygon:
		if len(g) == 0 {
			return orb.Polygon{}
		}
		if len(g) == 1 {
			return g[0]
		}
		return g
	default:
		return geom
	}
}

// normalizeCollectionResult normalizes a Collection result.
// Returns single geometry if collection has only one element.
// Returns appropriate Multi* type if all elements are same dimension.
// Returns Collection only when truly mixed dimensions.
func normalizeCollectionResult(coll orb.Collection) orb.Geometry {
	if len(coll) == 0 {
		return orb.Collection{}
	}
	if len(coll) == 1 {
		return coll[0]
	}

	// Check if all elements are the same type/dimension
	var points orb.MultiPoint
	var lines orb.MultiLineString
	var polygons orb.MultiPolygon
	hasPoints, hasLines, hasPolygons := false, false, false

	for _, g := range coll {
		switch geom := g.(type) {
		case orb.Point:
			points = append(points, geom)
			hasPoints = true
		case orb.MultiPoint:
			points = append(points, geom...)
			hasPoints = true
		case orb.LineString:
			lines = append(lines, geom)
			hasLines = true
		case orb.MultiLineString:
			lines = append(lines, geom...)
			hasLines = true
		case orb.Polygon:
			polygons = append(polygons, geom)
			hasPolygons = true
		case orb.MultiPolygon:
			polygons = append(polygons, geom...)
			hasPolygons = true
		case orb.Collection:
			// Recursively handle nested collections
			for _, nested := range geom {
				switch n := nested.(type) {
				case orb.Point:
					points = append(points, n)
					hasPoints = true
				case orb.MultiPoint:
					points = append(points, n...)
					hasPoints = true
				case orb.LineString:
					lines = append(lines, n)
					hasLines = true
				case orb.MultiLineString:
					lines = append(lines, n...)
					hasLines = true
				case orb.Polygon:
					polygons = append(polygons, n)
					hasPolygons = true
				case orb.MultiPolygon:
					polygons = append(polygons, n...)
					hasPolygons = true
				}
			}
		}
	}

	// Count how many different dimensions we have
	dimCount := 0
	if hasPoints {
		dimCount++
	}
	if hasLines {
		dimCount++
	}
	if hasPolygons {
		dimCount++
	}

	// If only one dimension, return the appropriate Multi* type
	if dimCount == 1 {
		if hasPolygons {
			return normalizePolygonResult(polygons)
		}
		if hasLines {
			return normalizeLineStringResult(lines)
		}
		if hasPoints {
			return normalizePointResult(points)
		}
	}

	// Mixed dimensions - build a proper Collection
	var result orb.Collection
	// Add polygons first (highest dimension), then lines, then points
	for _, p := range polygons {
		result = append(result, p)
	}
	for _, l := range lines {
		result = append(result, l)
	}
	for _, pt := range points {
		result = append(result, pt)
	}

	return result
}
