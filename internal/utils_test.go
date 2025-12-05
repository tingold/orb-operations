package internal

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestPointInPolygon(t *testing.T) {
	// Simple square polygon
	poly := orb.Polygon{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}},
	}

	tests := []struct {
		name     string
		point    orb.Point
		polygon  orb.Polygon
		expected bool
	}{
		{"point inside", orb.Point{5, 5}, poly, true},
		{"point outside", orb.Point{15, 15}, poly, false},
		{"point on boundary", orb.Point{5, 0}, poly, true},
		{"point at corner", orb.Point{0, 0}, poly, true},
		{"point outside left", orb.Point{-5, 5}, poly, false},
		{"point outside right", orb.Point{15, 5}, poly, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PointInPolygon(tt.point, tt.polygon)
			if result != tt.expected {
				t.Errorf("PointInPolygon() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPointInPolygonWithHole(t *testing.T) {
	// Polygon with a hole
	poly := orb.Polygon{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}, // Outer ring
		{{2, 2}, {8, 2}, {8, 8}, {2, 8}, {2, 2}},   // Hole
	}

	tests := []struct {
		name     string
		point    orb.Point
		expected bool
	}{
		{"point in outer ring", orb.Point{1, 1}, true},
		{"point in hole", orb.Point{5, 5}, false},
		{"point outside", orb.Point{15, 15}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PointInPolygon(tt.point, poly)
			if result != tt.expected {
				t.Errorf("PointInPolygon() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLineSegmentIntersection(t *testing.T) {
	tests := []struct {
		name      string
		p1, p2    orb.Point
		p3, p4    orb.Point
		expected  bool
		hasIntersection bool
	}{
		{
			"intersecting segments",
			orb.Point{0, 0}, orb.Point{10, 10},
			orb.Point{0, 10}, orb.Point{10, 0},
			true, true,
		},
		{
			"parallel segments",
			orb.Point{0, 0}, orb.Point{10, 0},
			orb.Point{0, 5}, orb.Point{10, 5},
			false, false,
		},
		{
			"non-intersecting segments",
			orb.Point{0, 0}, orb.Point{5, 5},
			orb.Point{10, 10}, orb.Point{15, 15},
			false, false,
		},
		{
			"touching at endpoint",
			orb.Point{0, 0}, orb.Point{5, 5},
			orb.Point{5, 5}, orb.Point{10, 10},
			true, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, intersects := LineSegmentIntersection(tt.p1, tt.p2, tt.p3, tt.p4)
			if intersects != tt.hasIntersection {
				t.Errorf("LineSegmentIntersection() = %v, want %v", intersects, tt.hasIntersection)
			}
		})
	}
}

func TestPolygonOrientation(t *testing.T) {
	// Test that orientation is consistent (doesn't matter which is which)
	clockwise := orb.Ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}
	counterClockwise := orb.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}

	cwResult := PolygonOrientation(clockwise)
	ccwResult := PolygonOrientation(counterClockwise)

	// They should be opposite
	if cwResult == ccwResult {
		t.Errorf("PolygonOrientation() should return opposite values for clockwise and counter-clockwise rings")
	}
}

func TestPointsEqual(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2   orb.Point
		epsilon  float64
		expected bool
	}{
		{"equal points", orb.Point{1.0, 2.0}, orb.Point{1.0, 2.0}, 0.001, true},
		{"different points", orb.Point{1.0, 2.0}, orb.Point{2.0, 3.0}, 0.001, false},
		{"close points within epsilon", orb.Point{1.0, 2.0}, orb.Point{1.0001, 2.0001}, 0.001, true},
		{"close points outside epsilon", orb.Point{1.0, 2.0}, orb.Point{1.002, 2.002}, 0.001, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PointsEqual(tt.p1, tt.p2, tt.epsilon)
			if result != tt.expected {
				t.Errorf("PointsEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}

