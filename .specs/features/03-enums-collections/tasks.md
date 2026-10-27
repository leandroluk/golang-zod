# Tasks: 03 — Enums & Collections

### [x] TASK-COLL-001: Enum constructor and operations
- **REQ**: REQ-COLL-001, REQ-COLL-006
- **Symbols**: Enum, EnumSchema, Values, Extract, Exclude
- **What**: Implement z.Enum[T ~string] with variadic values, extraction, exclusion, and InvalidValue issues.
- **Where**: `enum.go`
- **Depends on**: none
- **Done when**: Enum parses string inputs and supports Extract/Exclude operations.
- **Gate**: `go test -race -v -run "TestREQ_COLL_001"`

### [x] TASK-COLL-002: Array schema and sizing checks
- **REQ**: REQ-COLL-002, REQ-COLL-006
- **Symbols**: Array, ArraySchema, Element, Unwrap, Min, Max, Length, NonEmpty
- **What**: Implement full z.Array constructor with Min, Max, Length, NonEmpty, and element unwrapping.
- **Where**: `array.go`
- **Depends on**: none
- **Done when**: Array parses element items with correct path indices and sizing checks.
- **Gate**: `go test -race -v -run "TestREQ_COLL_002"`

### [x] TASK-COLL-003: Heterogeneous Tuple schema
- **REQ**: REQ-COLL-003, REQ-COLL-006
- **Symbols**: Tuple, TupleSchema, AnySchema, Rest
- **What**: Implement z.Tuple with fixed-length schema array and optional .Rest schema for variable trailing elements.
- **Where**: `tuple.go`
- **Depends on**: none
- **Done when**: Tuple validates positional schemas and lengths.
- **Gate**: `go test -race -v -run "TestREQ_COLL_003"`

### [x] TASK-COLL-004: Set schema
- **REQ**: REQ-COLL-004, REQ-COLL-006
- **Symbols**: Set, SetSchema, Size
- **What**: Implement z.Set[T] producing map[T]struct{} with collapsed duplicates and size checks.
- **Where**: `set.go`
- **Depends on**: none
- **Done when**: Set collapses duplicates and enforces size boundaries.
- **Gate**: `go test -race -v -run "TestREQ_COLL_004"`

### [x] TASK-COLL-005: Map schema
- **REQ**: REQ-COLL-005, REQ-COLL-006
- **Symbols**: Map, MapSchema
- **What**: Implement z.Map[K, V] validating keys and values with InvalidKey for bad keys.
- **Where**: `map.go`
- **Depends on**: none
- **Done when**: Map correctly validates key/value pairs and reports key errors.
- **Gate**: `go test -race -v -run "TestREQ_COLL_005"`

### [x] TASK-COLL-006: Comprehensive tests for Milestone 3
- **REQ**: REQ-COLL-001..006
- **Symbols**: TestREQ_COLL_*
- **What**: Implement full test suite for enums, arrays, tuples, sets, maps, and modifier wrappers.
- **Where**: `feature03_test.go`
- **Depends on**: TASK-COLL-001, TASK-COLL-002, TASK-COLL-003, TASK-COLL-004, TASK-COLL-005
- **Done when**: All tests pass under go test -race ./...
- **Gate**: `go test -race -v ./...`
