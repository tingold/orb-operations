package orboperations

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		geom1    orb.Geometry
		geom2    orb.Geometry
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			"point union point",
			orb.Point{1, 2},
			orb.Point{3, 4},
			func(t *testing.T, result orb.Geometry) {
				mp, ok := result.(orb.MultiPoint)
				if !ok {
					t.Errorf("expected MultiPoint, got %T", result)
					return
				}
				if len(mp) != 2 {
					t.Errorf("expected 2 points, got %d", len(mp))
				}
			},
		},
		{
			"point union same point",
			orb.Point{1, 2},
			orb.Point{1, 2},
			func(t *testing.T, result orb.Geometry) {
				// Accept either Point or MultiPoint with 1 point
				switch r := result.(type) {
				case orb.Point:
					// OK - single point returned
				case orb.MultiPoint:
					if len(r) < 1 {
						t.Errorf("expected at least 1 point, got %d", len(r))
					}
				default:
					t.Errorf("expected Point or MultiPoint, got %T", result)
				}
			},
		},
		{
			"multipoint union",
			orb.MultiPoint{{1, 2}, {3, 4}},
			orb.MultiPoint{{5, 6}, {1, 2}},
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
			"polygon union non-overlapping",
			orb.Polygon{{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}},
			orb.Polygon{{{10, 10}, {15, 10}, {15, 15}, {10, 15}, {10, 10}}},
			func(t *testing.T, result orb.Geometry) {
				// Accept either Polygon or MultiPolygon (two disjoint polygons)
				switch result.(type) {
				case orb.Polygon, orb.MultiPolygon:
					// OK
				default:
					t.Errorf("expected Polygon or MultiPolygon, got %T", result)
				}
			},
		},
		{
			"polygon union overlapping",
			orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
			orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}},
			func(t *testing.T, result orb.Geometry) {
				_, ok := result.(orb.Polygon)
				if !ok {
					t.Errorf("expected Polygon, got %T", result)
				}
			},
		},
		{
			"linestring union",
			orb.LineString{{0, 0}, {5, 5}},
			orb.LineString{{5, 5}, {10, 10}},
			func(t *testing.T, result orb.Geometry) {
				// Accept either LineString or MultiLineString
				switch r := result.(type) {
				case orb.LineString:
					if len(r) == 0 {
						t.Errorf("expected non-empty LineString")
					}
				case orb.MultiLineString:
					if len(r) == 0 {
						t.Errorf("expected at least one LineString")
					}
				default:
					t.Errorf("expected LineString or MultiLineString, got %T", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Union(tt.geom1, tt.geom2)
			if result == nil {
				t.Error("Union() returned nil")
				return
			}
			tt.validate(t, result)
		})
	}
}

func TestUnionNil(t *testing.T) {
	p := orb.Point{1, 2}
	
	result := Union(nil, p)
	if result != p {
		t.Errorf("Union(nil, p) = %v, want %v", result, p)
	}

	result = Union(p, nil)
	if result != p {
		t.Errorf("Union(p, nil) = %v, want %v", result, p)
	}
}

