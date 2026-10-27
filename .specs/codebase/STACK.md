# Stack & Policy

## Toolchain
- **Go 1.27.1** (`go.mod`: `go 1.27.1`). Verified on this toolchain: generic methods on concrete types compile; methods with type parameters in interfaces do not; `Base[T] -> Base[*T]` method is an instantiation cycle; `encoding/json/v2` needs no `GOEXPERIMENT`.
- **Dependencies**: standard library only. No `require` lines in `go.mod`. Use `regexp` (precompile at package init or schema construction), `encoding/json/v2`, `encoding/json/jsontext`, `math/big`, `net/netip`, `net/mail` where useful, `unicode/norm` is NOT stdlib (x/text) so implement `.Normalize()` only if available without deps; otherwise document as unsupported.
- **Module**: `github.com/leandroluk/golang-zod`, single root package named `z`.

## Policy
1. No code generation required to build or test.
2. `unsafe` is allowed ONLY in `struct_unsafe.go` (field setters of `z.Struct`, ADR-010/012), only as `unsafe.Add` on a base pointer with an offset validated at construction, typed to the verified field type. `reflect` only in `struct.go`. Anywhere else needs a new ADR in STATE.md first.
3. `go vet ./...` and `go test -race ./...` must pass for a feature to be done.
4. Tests: external package `z_test` in the repo root (`*_test.go`), one file per feature (`feature01_test.go` ...). Benchmarks in `bench_test.go`.
