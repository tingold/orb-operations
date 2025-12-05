# JTS Test Fixtures

This directory contains XML test fixtures from the JTS (Java Topology Suite) library for validating geometry operations.

## Downloading Test Fixtures

The JTS test fixtures can be obtained from the JTS repository. The test fixtures are typically located in the JTS TestBuilder or test suite.

### Option 1: From JTS Repository

1. Clone or download the JTS repository:
   ```bash
   git clone https://github.com/locationtech/jts.git
   ```

2. Locate test fixtures in the repository. They may be in:
   - `jts/testxml/` directory
   - TestBuilder test files
   - Test suite XML files

3. Copy relevant XML files to this directory (`testdata/jts/`)

### Option 2: From GeoTools

GeoTools also includes JTS test fixtures:

1. Clone the GeoTools repository:
   ```bash
   git clone https://github.com/geotools/geotools.git
   ```

2. Look for test fixtures in:
   - `modules/library/jts/src/test/resources/org/geotools/geometry/jts/`

3. Copy relevant XML files to this directory

## XML Format

The test fixtures should follow this XML format:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<run>
  <test name="test_case_1" op="UNION">
    <a>POLYGON((0 0, 10 0, 10 10, 0 10, 0 0))</a>
    <b>POLYGON((5 5, 15 5, 15 15, 5 15, 5 5))</b>
    <result>POLYGON((0 0, 10 0, 10 5, 15 5, 15 15, 5 15, 5 10, 0 10, 0 0))</result>
  </test>
  <test name="test_case_2" op="INTERSECTION">
    <a>POLYGON((0 0, 10 0, 10 10, 0 10, 0 0))</a>
    <b>POLYGON((5 5, 15 5, 15 15, 5 15, 5 5))</b>
    <result>POLYGON((5 5, 10 5, 10 10, 5 10, 5 5))</result>
  </test>
</run>
```

### Supported Operations

- `UNION` - Tests the Union operation
- `INTERSECTION` or `INTERSECT` - Tests the Intersection operation
- `DIFFERENCE` or `DIFF` - Tests the Difference operation
- `SYMDIFFERENCE` or `SYMMETRICDIFFERENCE` - Tests the SymmetricDifference operation

### Supported Geometry Types

The test fixtures support all WKT geometry types:
- `POINT`
- `MULTIPOINT`
- `LINESTRING`
- `MULTILINESTRING`
- `POLYGON`
- `MULTIPOLYGON`
- `EMPTY` (for empty geometries)

## Running Tests

JTS-based tests are opt-in so the regular `go test ./...` target stays fast and deterministic. Set `RUN_JTS_TESTS=1` when you want to execute them:

```bash
# Run all JTS tests
RUN_JTS_TESTS=1 go test ./... -run TestJTSOperations -v

# Print summary of available test fixtures
RUN_JTS_TESTS=1 go test ./... -run TestJTSSummary -v
```

## Test Results

The JTS test fixtures provide a comprehensive validation suite. Some tests may fail due to:

1. **Implementation differences**: The current implementation uses simplified algorithms that may not handle all edge cases (e.g., adjacent polygons, complex intersections)
2. **Geometry comparison**: The comparison function checks exact vertex order, but equivalent geometries may have different vertex sequences
3. **Missing features**: Some advanced operations (e.g., handling degenerate cases, precision issues) may not be fully implemented

These test fixtures serve as a validation tool to identify areas where the implementation can be improved to match JTS behavior more closely.

## Notes

- Test fixtures are not included in the repository by default to keep the repository size manageable
- You can add your own test fixtures by creating XML files in this directory
- The test runner will automatically discover and run all XML files in this directory

