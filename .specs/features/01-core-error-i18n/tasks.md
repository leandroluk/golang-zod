# Tasks: 01 — Core Engine, Typed Schema, Errors, Formatters & i18n

### [x] TASK-CORE-001: Option and Nullable presence types
- **REQ**: REQ-CORE-004
- **Symbols**: Option, Some, None, Nullable, NullableOf, NullableNull
- **What**: Implement `Option[T]` (presence/absence) and `Nullable[T]` (explicit null) with constructors, query methods (`IsSome`, `IsNull`, `Value`, `OrElse`), and JSON v2 serialization/deserialization methods.
- **Where**: `option.go`
- **Depends on**: none
- **Done when**: Unit tests for Option and Nullable pass including JSON marshaling and unmarshaling.
- **Gate**: `go test -race -v -run "TestREQ_CORE_004"`

### [x] TASK-CORE-002: Error taxonomy, issues, and ZodError
- **REQ**: REQ-CORE-005, REQ-CORE-006
- **Symbols**: IssueCode, PathSegment, Issue, ZodError
- **What**: Define the 11 `IssueCode` constants, `PathSegment` (key/index), `Issue` struct containing code-specific fields matching Zod v4, and `*ZodError` implementing `error` and `Issues() []Issue`.
- **Where**: `error.go`
- **Depends on**: none
- **Done when**: Unit tests verifying issue codes, issue fields, and ZodError error interface pass.
- **Gate**: `go test -race -v -run "TestREQ_CORE_005|TestREQ_CORE_006"`

### [x] TASK-CORE-003: Error formatters Flatten, Treeify, Prettify
- **REQ**: REQ-CORE-007
- **Symbols**: Flatten, Treeify, Prettify, FlattenedErrors, ErrorTreeNode
- **What**: Implement `Flatten` (first-segment field keys, form errors), `Treeify` (tree structure with properties/items), and `Prettify` (formatted string with path and mark).
- **Where**: `formatter.go`
- **Depends on**: TASK-CORE-002
- **Done when**: Unit tests asserting formatting output match Zod v4 specification.
- **Gate**: `go test -race -v -run "TestREQ_CORE_007"`

### [x] TASK-CORE-004: i18n system, catalogs, and Accept-Language negotiation
- **REQ**: REQ-CORE-008, REQ-CORE-009, REQ-CORE-013, REQ-CORE-014
- **Symbols**: Locale, LocaleEN, LocalePtBR, LocaleES, SetLocale, RegisterLocale, WithLocale, getActiveLocale
- **What**: Implement locale catalogs with message resolvers for all 11 issue codes and sizing origins; thread-safe atomic global default `SetLocale`; locale registry with built-ins `en`, `pt-BR`, `es`; header parsing for `Accept-Language` with q-factor sorting and prefix fallback.
- **Where**: `i18n.go`, `locale_en.go`, `locale_ptbr.go`, `locale_es.go`
- **Depends on**: TASK-CORE-002
- **Done when**: Messages correctly format in EN, pt-BR, and ES, and Accept-Language negotiation chooses the correct locale.
- **Gate**: `go test -race -v -run "TestREQ_CORE_008|TestREQ_CORE_009|TestREQ_CORE_013|TestREQ_CORE_014"`

### [x] TASK-CORE-005: Metadata and MetaBuilder
- **REQ**: REQ-CORE-010
- **Symbols**: Meta, MetaBuilder
- **What**: Implement `Meta` struct, `MetaBuilder` with chainable setter methods (`ID`, `Title`, `Describe`, `Examples`, `Deprecated`, `Custom`, `Reset`), and construction-time check ensuring duplicate non-empty IDs panic.
- **Where**: `meta.go`
- **Depends on**: none
- **Done when**: MetaBuilder merges existing meta and duplicate IDs panic.
- **Gate**: `go test -race -v -run "TestREQ_CORE_010"`

### [x] TASK-CORE-006: Untyped execution engine and parse context pool
- **REQ**: REQ-CORE-001, REQ-CORE-012, REQ-CORE-017
- **Symbols**: node, nodeKind, parseCtx, parseCtxPool
- **What**: Define internal `node` interface, node kinds, `parseCtx` with path stack, issue collection, abortEarly support, active locale, and context cancellation non-blocking checks (`ctx.Err()`); manage `sync.Pool` lifecycle and clean reset.
- **Where**: `node.go`
- **Depends on**: TASK-CORE-002, TASK-CORE-004
- **Done when**: `parseCtx` pool recycles cleanly without leakage and context cancellation halts parsing.
- **Gate**: `go test -race -v -run "TestREQ_CORE_001|TestREQ_CORE_012|TestREQ_CORE_017"`

### [x] TASK-CORE-007: Typed Schema interface, entry points, and global context configuration
- **REQ**: REQ-CORE-002, REQ-CORE-003, REQ-CORE-011, REQ-CORE-016
- **Symbols**: Schema, BaseSchema, Config, ContextOptions, LocaleContextKey, AbortEarlyContextKey, WithAbortEarly
- **What**: Implement `Schema[T]` interface, base typed wrapper calling `node.parse`, JSON v2 / jsontext stream decoding in `ParseJSON`/`Decode`, global `Config.UseContext(ContextOptions)` for key replacement, and well-known context keys.
- **Where**: `schema.go`, `config.go`, `z.go`
- **Depends on**: TASK-CORE-005, TASK-CORE-006
- **Done when**: `Parse`, `ParseJSON`, `Decode` execute end-to-end, `Config.UseContext` replaces context keys, and no typed-nil error is returned.
- **Gate**: `go test -race -v -run "TestREQ_CORE_002|TestREQ_CORE_003|TestREQ_CORE_011|TestREQ_CORE_016"`

### [x] TASK-CORE-008: Comprehensive verification and test suite for Milestone 1
- **REQ**: REQ-CORE-015, REQ-CORE-017
- **Symbols**: TestREQ_*
- **What**: Implement complete end-to-end and concurrency test suite for feature 01, verifying thread-safety with `-race`, context cancellation, custom registered locales, Fiber-like use cases, and benchmarks.
- **Where**: `feature01_test.go`
- **Depends on**: TASK-CORE-001, TASK-CORE-002, TASK-CORE-003, TASK-CORE-004, TASK-CORE-005, TASK-CORE-006, TASK-CORE-007
- **Done when**: All tests pass under `go test -race ./...` and `go vet ./...` is clean.
- **Gate**: `go test -race -v ./...`
