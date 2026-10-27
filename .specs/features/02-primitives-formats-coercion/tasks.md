# Tasks: 02 — Primitives, Top-Level String Formats, Numbers & Coercion

### [x] TASK-PRIM-001: Core primitives and literal
- **REQ**: REQ-PRIM-001, REQ-PRIM-009
- **Symbols**: Boolean, Null, Any, Unknown, Never, Date, Bytes, Literal
- **What**: Implement primitive constructors: Boolean, Null (validates null, outputs nil), Any, Unknown, Never (always fails), Date (accepts time.Time and RFC3339 string), Bytes (accepts []byte and base64 string), Literal (matches exact values).
- **Where**: `primitives.go`
- **Depends on**: none
- **Done when**: Unit tests for all primitive types pass.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_001"`

### [x] TASK-PRIM-002: StringSchema, checks, and transforms
- **REQ**: REQ-PRIM-002, REQ-PRIM-009
- **Symbols**: String, StringSchema, Min, Max, Length, NonEmpty, Regex, Includes, StartsWith, EndsWith, Lowercase, Uppercase, Trim, ToLowerCase, ToUpperCase, Slugify
- **What**: Implement String() constructor, fluent checks, and copy-on-write transforms.
- **Where**: `string.go`
- **Depends on**: none
- **Done when**: String checks and transforms pass unit tests.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_002"`

### [x] TASK-PRIM-003: Top-level string formats
- **REQ**: REQ-PRIM-003, REQ-PRIM-004, REQ-PRIM-005
- **Symbols**: UUID, UUIDv4, UUIDv6, UUIDv7, GUID, NanoID, CUID2, ULID, XID, KSUID, Email, URL, HTTPURL, Hostname, IPv4, IPv6, MAC, CIDRv4, CIDRv6, Base64, Base64URL, Hex, JWT, Emoji, E164, CreditCard, IBAN, CurrencyCode, Hash, StringFormat
- **What**: Implement format constructors returning *StringSchema tagged with specific format validators.
- **Where**: `formats.go`
- **Depends on**: TASK-PRIM-002
- **Done when**: All format constructors pass verification with valid and invalid inputs.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_003|TestREQ_PRIM_004|TestREQ_PRIM_005"`

### [x] TASK-PRIM-004: ISO namespace
- **REQ**: REQ-PRIM-006
- **Symbols**: ISO, isoNamespace, ISODate, ISOTime, ISODateTime, ISODuration
- **What**: Implement z.ISO namespace with Date, Time, DateTime (with precision/offset/local options), and Duration validators.
- **Where**: `iso.go`
- **Depends on**: TASK-PRIM-002
- **Done when**: ISO formats pass validation unit tests.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_006"`

### [x] TASK-PRIM-005: Numbers and numeric checks
- **REQ**: REQ-PRIM-007, REQ-PRIM-009
- **Symbols**: Number, Float64, Float32, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, BigInt, NumberSchema, BigIntSchema, GT, GTE, LT, LTE, Positive, NonNegative, Negative, NonPositive, MultipleOf
- **What**: Implement generic NumberSchema[T], constructors for all Go numeric primitives, BigInt with *big.Int, and range/multiple checks.
- **Where**: `number.go`
- **Depends on**: none
- **Done when**: Numeric constructors and checks correctly enforce boundaries without float precision loss for 64-bit integers.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_007"`

### [x] TASK-PRIM-006: Coerce namespace
- **REQ**: REQ-PRIM-008
- **Symbols**: Coerce, coerceNamespace
- **What**: Implement z.Coerce namespace with String, Number, Int, Float, Boolean, BigInt, Date coercing input prior to validation.
- **Where**: `coerce.go`
- **Depends on**: TASK-PRIM-001, TASK-PRIM-002, TASK-PRIM-005
- **Done when**: Coercion transforms strings/numbers/booleans according to Zod v4 truthiness and parsing rules.
- **Gate**: `go test -race -v -run "TestREQ_PRIM_008"`

### [x] TASK-PRIM-007: Comprehensive tests for Milestone 2
- **REQ**: REQ-PRIM-001..009
- **Symbols**: TestREQ_PRIM_*
- **What**: Implement comprehensive test suite for all Feature 02 primitives, formats, numbers, coercion, and modifier methods.
- **Where**: `feature02_test.go`
- **Depends on**: TASK-PRIM-001, TASK-PRIM-002, TASK-PRIM-003, TASK-PRIM-004, TASK-PRIM-005, TASK-PRIM-006
- **Done when**: All tests pass under go test -race ./...
- **Gate**: `go test -race -v ./...`
