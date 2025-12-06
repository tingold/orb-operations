package orboperations

import (
	"math"
	"testing"

	"github.com/paulmach/orb"
)

func TestBufferPoint(t *testing.T) {
	tests := []struct {
		name     string
		point    orb.Point
		distance float64
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			name:     "buffer point with positive distance",
			point:    orb.Point{0, 0},
			distance: 1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) != 1 {
					t.Errorf("expected 1 ring, got %d", len(poly))
					return
				}
				// Check that all points are approximately distance 1 from origin
				for _, pt := range poly[0] {
					dist := math.Sqrt(pt[0]*pt[0] + pt[1]*pt[1])
					if math.Abs(dist-1.0) > 0.01 {
						t.Errorf("point %v has distance %f from origin, expected ~1.0", pt, dist)
					}
				}
			},
		},
		{
			name:     "buffer point with zero distance",
			point:    orb.Point{5, 5},
			distance: 0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) != 0 {
					t.Errorf("expected empty polygon for zero distance, got %d rings", len(poly))
				}
			},
		},
		{
			name:     "buffer point with negative distance",
			point:    orb.Point{0, 0},
			distance: -1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) != 0 {
					t.Errorf("expected empty polygon for negative distance, got %d rings", len(poly))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Buffer(tt.point, tt.distance)
			tt.validate(t, result)
		})
	}
}

func TestBufferLineString(t *testing.T) {
	tests := []struct {
		name     string
		line     orb.LineString
		distance float64
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			name:     "buffer horizontal line",
			line:     orb.LineString{{0, 0}, {10, 0}},
			distance: 1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
					return
				}
				// Check bounds - should extend 1 unit in all directions from the line
				bound := poly.Bound()
				if bound.Min[0] > -0.5 || bound.Max[0] < 10.5 {
					t.Errorf("unexpected X bounds: [%f, %f]", bound.Min[0], bound.Max[0])
				}
				if bound.Min[1] > -0.5 || bound.Max[1] < 0.5 {
					t.Errorf("unexpected Y bounds: [%f, %f]", bound.Min[1], bound.Max[1])
				}
			},
		},
		{
			name:     "buffer vertical line",
			line:     orb.LineString{{5, 0}, {5, 10}},
			distance: 2.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
					return
				}
				bound := poly.Bound()
				if bound.Min[0] > 3.5 || bound.Max[0] < 6.5 {
					t.Errorf("unexpected X bounds: [%f, %f]", bound.Min[0], bound.Max[0])
				}
			},
		},
		{
			name:     "buffer multi-segment line",
			line:     orb.LineString{{0, 0}, {5, 0}, {5, 5}, {10, 5}},
			distance: 1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
					return
				}
			},
		},
		{
			name:     "buffer short line",
			line:     orb.LineString{{0, 0}, {1, 0}},
			distance: 0.5,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Buffer(tt.line, tt.distance)
			tt.validate(t, result)
		})
	}
}

func TestBufferPolygon(t *testing.T) {
	square := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}

	tests := []struct {
		name     string
		poly     orb.Polygon
		distance float64
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			name:     "expand square polygon",
			poly:     square,
			distance: 1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
					return
				}
				bound := poly.Bound()
				// Should extend beyond original bounds
				if bound.Min[0] > -0.5 || bound.Max[0] < 10.5 {
					t.Errorf("unexpected X bounds: [%f, %f]", bound.Min[0], bound.Max[0])
				}
				if bound.Min[1] > -0.5 || bound.Max[1] < 10.5 {
					t.Errorf("unexpected Y bounds: [%f, %f]", bound.Min[1], bound.Max[1])
				}
			},
		},
		{
			name:     "shrink square polygon",
			poly:     square,
			distance: -1.0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) == 0 {
					t.Error("expected non-empty polygon")
					return
				}
				bound := poly.Bound()
				// Should be smaller than original
				if bound.Min[0] < 0.5 || bound.Max[0] > 9.5 {
					t.Errorf("unexpected X bounds: [%f, %f]", bound.Min[0], bound.Max[0])
				}
			},
		},
		{
			name:     "zero distance returns same polygon",
			poly:     square,
			distance: 0,
			validate: func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) != 1 {
					t.Errorf("expected 1 ring, got %d", len(poly))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Buffer(tt.poly, tt.distance)
			tt.validate(t, result)
		})
	}
}

func TestBufferWithParams(t *testing.T) {
	point := orb.Point{0, 0}

	t.Run("custom quadrant segments", func(t *testing.T) {
		params := DefaultBufferParams()
		params.QuadrantSegments = 4 // Fewer segments = fewer points

		result := BufferWithParams(point, 1.0, params)
		poly, ok := result.(orb.Polygon)
		if !ok {
			t.Fatalf("expected Polygon, got %T", result)
		}

		// With 4 quadrant segments, we get 16 points + closing point = 17
		expectedPoints := 17
		if len(poly[0]) != expectedPoints {
			t.Errorf("expected %d points with 4 quadrant segments, got %d", expectedPoints, len(poly[0]))
		}
	})

	t.Run("cap styles on linestring", func(t *testing.T) {
		line := orb.LineString{{0, 0}, {5, 0}}

		// Round cap (default) should have more points than flat
		roundParams := DefaultBufferParams()
		roundParams.CapStyle = CapRound
		roundResult := BufferWithParams(line, 1.0, roundParams)

		flatParams := DefaultBufferParams()
		flatParams.CapStyle = CapFlat
		flatResult := BufferWithParams(line, 1.0, flatParams)

		roundPoly := roundResult.(orb.Polygon)
		flatPoly := flatResult.(orb.Polygon)

		if len(flatPoly[0]) >= len(roundPoly[0]) {
			t.Error("expected round cap to have more points than flat cap")
		}
	})
}

func TestBufferMultiPoint(t *testing.T) {
	mp := orb.MultiPoint{{0, 0}, {10, 0}}

	result := Buffer(mp, 1.0)

	// Result could be Polygon or MultiPolygon depending on overlap
	switch r := result.(type) {
	case orb.Polygon:
		if len(r) == 0 {
			t.Error("expected non-empty polygon")
		}
	case orb.MultiPolygon:
		if len(r) == 0 {
			t.Error("expected non-empty multipolygon")
		}
	default:
		t.Errorf("expected Polygon or MultiPolygon, got %T", result)
	}
}

func TestBufferMultiLineString(t *testing.T) {
	mls := orb.MultiLineString{
		{{0, 0}, {5, 0}},
		{{0, 5}, {5, 5}},
	}

	result := Buffer(mls, 1.0)

	switch r := result.(type) {
	case orb.Polygon:
		if len(r) == 0 {
			t.Error("expected non-empty polygon")
		}
	case orb.MultiPolygon:
		if len(r) == 0 {
			t.Error("expected non-empty multipolygon")
		}
	default:
		t.Errorf("expected Polygon or MultiPolygon, got %T", result)
	}
}

func TestBufferMultiPolygon(t *testing.T) {
	mp := orb.MultiPolygon{
		{{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}},
		{{{10, 0}, {15, 0}, {15, 5}, {10, 5}, {10, 0}}},
	}

	result := Buffer(mp, 1.0)

	switch r := result.(type) {
	case orb.Polygon:
		if len(r) == 0 {
			t.Error("expected non-empty polygon")
		}
	case orb.MultiPolygon:
		if len(r) == 0 {
			t.Error("expected non-empty multipolygon")
		}
	default:
		t.Errorf("expected Polygon or MultiPolygon, got %T", result)
	}
}

func TestBufferNil(t *testing.T) {
	result := Buffer(nil, 1.0)
	if result != nil {
		t.Errorf("expected nil for nil geometry, got %T", result)
	}
}

func TestBufferCollection(t *testing.T) {
	coll := orb.Collection{
		orb.Point{0, 0},
		orb.LineString{{5, 0}, {10, 0}},
	}

	result := Buffer(coll, 1.0)

	switch r := result.(type) {
	case orb.Polygon:
		if len(r) == 0 {
			t.Error("expected non-empty polygon")
		}
	case orb.MultiPolygon:
		if len(r) == 0 {
			t.Error("expected non-empty multipolygon")
		}
	default:
		t.Errorf("expected Polygon or MultiPolygon, got %T", result)
	}
}

func TestBufferRing(t *testing.T) {
	ring := orb.Ring{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}

	result := Buffer(ring, 1.0)

	poly, ok := result.(orb.Polygon)
	if !ok {
		t.Errorf("expected Polygon, got %T", result)
		return
	}
	if len(poly) == 0 {
		t.Error("expected non-empty polygon")
	}
}

// Benchmark tests
func BenchmarkBufferPoint(b *testing.B) {
	point := orb.Point{0, 0}
	for i := 0; i < b.N; i++ {
		Buffer(point, 1.0)
	}
}

func BenchmarkBufferLineString(b *testing.B) {
	line := orb.LineString{{0, 0}, {5, 0}, {5, 5}, {10, 5}}
	for i := 0; i < b.N; i++ {
		Buffer(line, 1.0)
	}
}

func BenchmarkBufferPolygon(b *testing.B) {
	poly := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	for i := 0; i < b.N; i++ {
		Buffer(poly, 1.0)
	}
}
