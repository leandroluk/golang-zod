# Spec: 09 — Allocation & Throughput Optimizations (Go)

## Summary
Specifies Milestone 7 per [INSIGHT.md](../../../INSIGHT.md) §4. Goals are **measured** (benchmarks with `-benchmem`), not guarantees. Correctness (features 01–08) must be complete before starting.
1. Single-pass streaming parse for `ParseJSON`/`Decode` using `encoding/json/jsontext.Decoder`, validating and building the output (`map[string]any`) in one pass, without materializing an intermediate decoded `any`.
2. Inline path stack in `parseCtx` (fixed-size array, spill to slice) so a valid payload allocates no path memory.
3. Pooled `parseCtx` with complete reset; pooled issue slices.
4. Fast paths for scalar schemas (`String`, `Int`, `Boolean`, `Float64`) avoiding closures and boxing.
5. Benchmark harness and allocation budgets.

## Requirements
- REQ-OPT-001: `parseCtx` path is `[8]PathSegment` plus an overflow slice; pushing/popping keys allocates nothing; a heap `[]PathSegment` is built only when an `Issue` is emitted. Test with 12 nesting levels (spill).
- REQ-OPT-002: `ParseJSON`/`Decode` stream tokens through the node tree and build the output directly; fall back to the baseline path for schemas containing nodes without a streaming implementation (Union, Lazy, Codec, Transform, Catchall). Both paths must produce identical results and issues (differential test over a fuzz corpus).
- REQ-OPT-003: Numbers are validated from token text; `Int`, `Int64`, `Uint64`, `BigInt` never touch `float64`.
- REQ-OPT-004: `sync.Pool` usage verified: no leaked state across parses (test running 10 000 mixed valid/invalid parses in parallel with `-race`).
- REQ-OPT-005: `bench_test.go` with benchmarks: primitive string, email, nested object (6 fields) via `Parse`/`ParseJSON`, the same via `z.Struct`, invalid object with issue paths. Record baseline numbers in this spec's Status section when done. Budget goal: primitive 0 allocs/op; valid nested object: only the output map and its entries.
- REQ-OPT-007a: `*StructSchema[T]` (feature 04) streams tokens directly into `T` through its field setters; goal for a valid payload: allocations limited to the returned `T` and its inner slices/strings. Provide `ParseTo(data []byte, dst *T, ctx ...context.Context) error` to reuse a destination (zero allocations when `dst` is reused and fields are scalars).
- REQ-OPT-006: `StructSchema[T]` field setters must not allocate beyond what filling the destination requires (verify with `-benchmem`).
- REQ-OPT-007: All optimizations keep 100% of existing tests, formatters and `go test -race` green.

## Affected Components
- `node.go` (`parseCtx`), `object.go` (streaming), `bench_test.go`.

## Out of Scope
- `unsafe`-based offset writes (needs a decision recorded in STATE.md first), code generation.

## Open Questions
- None.
