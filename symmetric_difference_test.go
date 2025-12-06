package orboperations

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestSymmetricDifference(t *testing.T) {
	tests := []struct {
		name     string
		geom1    orb.Geometry
		geom2    orb.Geometry
		validate func(t *testing.T, result orb.Geometry)
	}{
		{
			"point symmetric difference same point",
			orb.Point{1, 2},
			orb.Point{1, 2},
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
			"point symmetric difference different points",
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
			"multipoint symmetric difference",
			orb.MultiPoint{{1, 2}, {3, 4}},
			orb.MultiPoint{{3, 4}, {5, 6}},
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
			"polygon symmetric difference",
			orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
			orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}},
			func(t *testing.T, result orb.Geometry) {
				// Symmetric difference should return a geometry
				if result == nil {
					t.Error("SymmetricDifference() returned nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SymmetricDifference(tt.geom1, tt.geom2)
			if result == nil {
				t.Error("SymmetricDifference() returned nil")
				return
			}
			tt.validate(t, result)
		})
	}
}
