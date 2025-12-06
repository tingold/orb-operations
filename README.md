# orb-operations

A pure Go library providing spatial operations (Union, Intersection, Difference, SymmetricDifference, Buffer) for the [Orb geometry library](https://github.com/paulmach/orb).

## Features

- **Pure Go implementation** - No C dependencies (no GEOS, etc.)
- **Minimal dependencies** - Only depends on `github.com/paulmach/orb`
- **Comprehensive geometry support** - Works with Points, MultiPoints, LineStrings, MultiLineStrings, Polygons, and MultiPolygons
- **Five core operations**:
  - `Union` - Combines two geometries
  - `Intersection` - Finds the common parts of two geometries
  - `Difference` - Subtracts one geometry from another
  - `SymmetricDifference` - Returns parts of geometries that don't overlap
  - `Buffer` - Creates a buffer zone around a geometry at a specified distance

## Installation

```bash
go get github.com/tingold/orb-operations
```

## Usage

### Basic Examples

#### Union

```go
package main

import (
	"fmt"
	"github.com/paulmach/orb"
	"github.com/tingold/orb-operations"
)

func main() {
	// Union of two polygons
	poly1 := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	poly2 := orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}}
	
	result := orboperations.Union(poly1, poly2)
	fmt.Printf("Union result: %v\n", result)
}
```

#### Intersection

```go
	// Intersection of two polygons
	poly1 := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	poly2 := orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}}
	
	result := orboperations.Intersection(poly1, poly2)
	fmt.Printf("Intersection result: %v\n", result)
```

#### Difference

```go
	// Difference (poly1 - poly2)
	poly1 := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	poly2 := orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}}
	
	result := orboperations.Difference(poly1, poly2)
	fmt.Printf("Difference result: %v\n", result)
```

#### Symmetric Difference

```go
	// Symmetric difference
	poly1 := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	poly2 := orb.Polygon{{{5, 5}, {15, 5}, {15, 15}, {5, 15}, {5, 5}}}
	
	result := orboperations.SymmetricDifference(poly1, poly2)
	fmt.Printf("Symmetric difference result: %v\n", result)
```

#### Buffer

```go
	// Buffer a point (creates a circular polygon)
	point := orb.Point{0, 0}
	buffered := orboperations.Buffer(point, 5.0)
	fmt.Printf("Buffered point: %v\n", buffered)
	
	// Buffer a linestring
	line := orb.LineString{{0, 0}, {10, 10}}
	bufferedLine := orboperations.Buffer(line, 2.0)
	fmt.Printf("Buffered line: %v\n", bufferedLine)
	
	// Buffer a polygon (expand or contract)
	poly := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	expanded := orboperations.Buffer(poly, 2.0)   // Expand by 2 units
	contracted := orboperations.Buffer(poly, -1.0) // Contract by 1 unit
	
	// Buffer with custom parameters
	params := orboperations.DefaultBufferParams()
	params.QuadrantSegments = 16  // More segments for smoother curves
	params.CapStyle = orboperations.CapSquare
	params.JoinStyle = orboperations.JoinMiter
	bufferedCustom := orboperations.BufferWithParams(line, 3.0, params)
```

### Working with Points

```go
	// Point operations
	p1 := orb.Point{1, 2}
	p2 := orb.Point{3, 4}
	
	union := orboperations.Union(p1, p2)        // Returns MultiPoint
	intersection := orboperations.Intersection(p1, p2)  // Returns MultiPoint (empty if different)
	difference := orboperations.Difference(p1, p2)     // Returns MultiPoint
```

### Working with LineStrings

```go
	ls1 := orb.LineString{{0, 0}, {5, 5}}
	ls2 := orb.LineString{{5, 5}, {10, 10}}
	
	union := orboperations.Union(ls1, ls2)  // Returns MultiLineString
	intersection := orboperations.Intersection(ls1, ls2)  // Returns MultiPoint (intersection points)
```

### Point in Polygon Operations

```go
	point := orb.Point{5, 5}
	poly := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	
	// Check if point is in polygon
	intersection := orboperations.Intersection(point, poly)
	if mp, ok := intersection.(orb.MultiPoint); ok && len(mp) > 0 {
		fmt.Println("Point is inside polygon")
	}
```

## API Reference

### Functions

#### `Union(geom1, geom2 orb.Geometry) orb.Geometry`

Computes the union of two geometries. Returns a geometry representing all points that are in either `geom1` or `geom2`.

**Supported combinations:**
- Point × Point → MultiPoint
- Point × MultiPoint → MultiPoint
- Point × Polygon → MultiPoint or Polygon
- MultiPoint × MultiPoint → MultiPoint
- LineString × LineString → MultiLineString
- Polygon × Polygon → Polygon
- MultiPolygon × MultiPolygon → Polygon or MultiPolygon

#### `Intersection(geom1, geom2 orb.Geometry) orb.Geometry`

Computes the intersection of two geometries. Returns a geometry representing all points that are in both `geom1` and `geom2`.

**Supported combinations:**
- Point × Point → MultiPoint (empty if different)
- Point × Polygon → MultiPoint (point if inside, empty otherwise)
- Polygon × Polygon → Polygon
- LineString × LineString → MultiPoint (intersection points)

#### `Difference(geom1, geom2 orb.Geometry) orb.Geometry`

Computes the difference of two geometries (`geom1 - geom2`). Returns a geometry representing all points that are in `geom1` but not in `geom2`.

#### `SymmetricDifference(geom1, geom2 orb.Geometry) orb.Geometry`

Computes the symmetric difference of two geometries. Equivalent to `Union(Difference(geom1, geom2), Difference(geom2, geom1))`.

#### `Buffer(geom orb.Geometry, distance float64) orb.Geometry`

Computes a buffer around a geometry at the specified distance. Positive distance expands the geometry, negative distance shrinks polygons. Returns a Polygon or MultiPolygon representing the buffered area.

**Supported geometry types:**
- `Point` → Creates a circular polygon approximation
- `MultiPoint` → Buffers each point and unions the results
- `LineString` → Creates a polygon around the line at the given distance
- `MultiLineString` → Buffers each line and unions the results
- `Polygon` → Expands (positive) or contracts (negative) the polygon
- `MultiPolygon` → Buffers each polygon and unions the results
- `Ring` → Treated as a closed LineString
- `Collection` → Buffers each element and unions the results

#### `BufferWithParams(geom orb.Geometry, distance float64, params BufferParams) orb.Geometry`

Computes a buffer with custom parameters. See `BufferParams` for available options.

#### `DefaultBufferParams() BufferParams`

Returns default buffer parameters:
- `QuadrantSegments`: 8 (32 segments for a full circle)
- `CapStyle`: `CapRound` (rounded ends)
- `JoinStyle`: `JoinRound` (rounded corners)
- `MiterLimit`: 5.0

#### Buffer Parameters

**Cap Styles:**
- `CapRound` - Rounded ends (default)
- `CapFlat` - Flat/butt ends
- `CapSquare` - Square ends extending past endpoint

**Join Styles:**
- `JoinRound` - Rounded corners (default)
- `JoinMiter` - Pointed corners (uses `MiterLimit`)
- `JoinBevel` - Flat corners

## Implementation Notes

### Algorithms

- **Polygon operations**: Uses a robust Vatti-based clipping algorithm for polygon boolean operations. The implementation combines multiple algorithms:
  - **Union**: Uses Greiner-Hormann algorithm for robust polygon union operations
  - **Intersection**: Uses Sutherland-Hodgman clipping algorithm
  - **Difference & Symmetric Difference**: Uses Vatti-based approach with proper handling of complex cases
  - Handles polygons with holes correctly
- **LineString operations**: Uses line segment intersection algorithms to find intersection points and combines segments.
- **Point operations**: Uses set-based operations with epsilon tolerance for floating-point comparisons.
- **Point-in-polygon**: Uses the ray casting algorithm.
- **Buffer operations**: 
  - **Points**: Creates circular polygon approximations using parametric circle generation
  - **LineStrings**: Generates offset curves and creates polygons around line segments with configurable cap and join styles
  - **Polygons**: Uses offset curve generation for expansion/contraction, handling both positive (expansion) and negative (contraction) distances
  - Supports multiple cap styles (Round, Flat, Square) and join styles (Round, Miter, Bevel)

### Limitations

- LineString operations with Polygons are simplified and may not clip line strings precisely.
- Floating-point precision is handled with epsilon tolerance (1e-9).

## Testing

Run the test suite:

```bash
go test ./...
```

### JTS Compatibility Testing

This library includes support for running JTS (Java Topology Suite) test fixtures to validate correctness against the industry-standard JTS implementation. This provides a high level of assurance about the correctness of the operations.

To use JTS test fixtures:

1. Download JTS test fixtures from the [JTS repository](https://github.com/locationtech/jts) or [GeoTools repository](https://github.com/geotools/geotools)
2. Place XML test fixture files in the `testdata/jts/` directory
3. Run the JTS compatibility tests:

```bash
# Run all JTS tests
go test ./... -run TestJTSOperations -v

# Print summary of available test fixtures
go test ./... -run TestJTSSummary -v
```

See `testdata/jts/README.md` for detailed instructions on obtaining and using JTS test fixtures.

## License

This project is licensed under the MIT License.

## Related Projects

- [orb](https://github.com/paulmach/orb) - Core geometry types
- [orb-predicates](https://github.com/tingold/orb-predicates) - Geometry predicates (Contains, Intersects, etc.)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

