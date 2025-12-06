package orboperations

import (
	"github.com/paulmach/orb"
	"github.com/tingold/orb-operations/internal"
)

// BufferParams holds parameters for buffer operations.
// Use DefaultBufferParams() to get sensible defaults.
type BufferParams = internal.BufferParams

// Cap styles for line end handling.
const (
	CapRound  = internal.CapRound  // Rounded ends (default)
	CapFlat   = internal.CapFlat   // Flat/butt ends
	CapSquare = internal.CapSquare // Square ends extending past endpoint
)

// Join styles for corner handling.
const (
	JoinRound = internal.JoinRound // Rounded corners (default)
	JoinMiter = internal.JoinMiter // Pointed corners
	JoinBevel = internal.JoinBevel // Flat corners
)

// DefaultBufferParams returns default buffer parameters.
// - QuadrantSegments: 8 (32 segments for a full circle)
// - CapStyle: CapRound
// - JoinStyle: JoinRound
// - MiterLimit: 5.0
func DefaultBufferParams() BufferParams {
	return internal.DefaultBufferParams()
}

// Buffer computes a buffer around a geometry at the specified distance.
// Positive distance expands the geometry, negative distance shrinks polygons.
// Returns a Polygon or MultiPolygon representing the buffered area.
//
// For Points: creates a circular polygon approximation.
// For LineStrings: creates a polygon around the line at the given distance.
// For Polygons: expands (positive) or contracts (negative) the polygon.
func Buffer(geom orb.Geometry, distance float64) orb.Geometry {
	return BufferWithParams(geom, distance, DefaultBufferParams())
}

// BufferWithParams computes a buffer with custom parameters.
// See BufferParams for available options.
func BufferWithParams(geom orb.Geometry, distance float64, params BufferParams) orb.Geometry {
	if geom == nil {
		return nil
	}

	switch g := geom.(type) {
	case orb.Point:
		return internal.BufferPoint(g, distance, params)

	case orb.MultiPoint:
		return internal.BufferMultiPoint(g, distance, params)

	case orb.LineString:
		return internal.BufferLineString(g, distance, params)

	case orb.MultiLineString:
		return internal.BufferMultiLineString(g, distance, params)

	case orb.Polygon:
		return internal.BufferPolygon(g, distance, params)

	case orb.MultiPolygon:
		return internal.BufferMultiPolygon(g, distance, params)

	case orb.Ring:
		// Treat ring as a closed linestring
		ls := orb.LineString(g)
		return internal.BufferLineString(ls, distance, params)

	case orb.Collection:
		return bufferCollection(g, distance, params)

	default:
		return nil
	}
}

// bufferCollection buffers each element of a collection and unions the results.
func bufferCollection(coll orb.Collection, distance float64, params BufferParams) orb.Geometry {
	if len(coll) == 0 {
		return nil
	}

	var results []orb.Polygon
	for _, g := range coll {
		buffered := BufferWithParams(g, distance, params)
		if buffered != nil {
			switch b := buffered.(type) {
			case orb.Polygon:
				if len(b) > 0 && len(b[0]) >= 4 {
					results = append(results, b)
				}
			case orb.MultiPolygon:
				for _, p := range b {
					if len(p) > 0 && len(p[0]) >= 4 {
						results = append(results, p)
					}
				}
			}
		}
	}

	if len(results) == 0 {
		return orb.Polygon{}
	}

	if len(results) == 1 {
		return results[0]
	}

	// Union all results
	result := results[0]
	for i := 1; i < len(results); i++ {
		unioned := internal.PolygonUnionMulti(result, results[i])
		if len(unioned) == 1 {
			result = unioned[0]
		} else if len(unioned) > 1 {
			// If union produces multiple polygons, continue with first
			result = unioned[0]
		}
	}

	return result
}
