package orboperations

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestIntersection(t *testing.T) {
	tests := []struct {
		name     string
		geom1    orb.Geometry
		geom2    orb.Geometry
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			"point intersection same point",
			orb.Point{1, 2},
			orb.Point{1, 2},
			func(t *testing.T, result orb.Geometry) {
				// Accept either Point or MultiPoint with 1 point
				switch r := result.(type) {
				case orb.Point:
					// OK - single point returned
				case orb.MultiPoint:
					if len(r) != 1 {
						t.Errorf("expected 1 point, got %d", len(r))
					}
				default:
					t.Errorf("expected Point or MultiPoint, got %T", result)
				}
			},
		},
		{
			"point intersection different points",
			orb.Point{1, 2},
			orb.Point{3, 4},
			func(t *testing.T, result orb.Geometry) {
				mp, ok := result.(orb.MultiPoint)
				if !ok {
					t.Errorf("expected MultiPoint, got %T", result)
					return
				}
				if len(mp) != 0 {
					t.Errorf("expected 0 points, got %d", len(mp))
				}
			},
		},
		{
			"multipoint intersection",
			orb.MultiPoint{{1, 2}, {3, 4}, {5, 6}},
			orb.MultiPoint{{3, 4}, {5, 6}, {7, 8}},
			func(t *testing.T, result orb.Geometry) {
				mp, ok := result.(orb.MultiPoint)
				if !ok {
					t.Errorf("expected MultiPoint, got %T", result)
					return
				}
				if len(mp) < 2 {
					t.Errorf("expected at least 2 points, got %d", len(mp))
				}
			},
		},
		{
			"polygon intersection overlapping",
			orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
			orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}},
			func(t *testing.T, result orb.Geometry) {
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
		{
			"polygon intersection non-overlapping",
			orb.Polygon{{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}},
			orb.Polygon{{{10, 10}, {15, 10}, {15, 15}, {10, 15}, {10, 10}}},
			func(t *testing.T, result orb.Geometry) {
				poly, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
					return
				}
				if len(poly) > 0 {
					t.Error("expected empty polygon for non-overlapping polygons")
				}
			},
		},
		{
			"point in polygon intersection",
			orb.Point{5, 5},
			orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
			func(t *testing.T, result orb.Geometry) {
				// Accept either Point or MultiPoint with 1 point
				switch r := result.(type) {
				case orb.Point:
					// OK - single point returned
				case orb.MultiPoint:
					if len(r) != 1 {
						t.Errorf("expected 1 point, got %d", len(r))
					}
				default:
					t.Errorf("expected Point or MultiPoint, got %T", result)
				}
			},
		},
		{
			"point outside polygon intersection",
			orb.Point{15, 15},
			orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
			func(t *testing.T, result orb.Geometry) {
				mp, ok := result.(orb.MultiPoint)
				if !ok {
					t.Errorf("expected MultiPoint, got %T", result)
					return
				}
				if len(mp) != 0 {
					t.Errorf("expected 0 points, got %d", len(mp))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Intersection(tt.geom1, tt.geom2)
			if result == nil {
				t.Error("Intersection() returned nil")
				return
			}
			tt.validate(t, result)
		})
	}
}
