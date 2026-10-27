# Architecture Concerns: golang-zod

## 1. Hot-Path Reflection Avoidance
* **Risk**: Reflection in Go can degrade throughput in microservices under high load.
* **Mitigation**: Confine reflection strictly to `z.Object[T]()` compilation. Cache compiled AST nodes in a concurrent map (`sync.Map`) keyed by `reflect.Type`.

## 2. Heap Allocations in Happy Path
* **Risk**: Creating new slices for error paths on every field validation.
* **Mitigation**: Use `sync.Pool` for `ParseContext` and pre-allocate path buffers. Do not construct `Issue` objects unless a rule fails.

## 3. Recursive Structs & Cycles
* **Risk**: Infinite loops during type inspection or recursive schema validation.
* **Mitigation**: Cycle detection in `z.Lazy` and `z.Object[T]()` with memoization pointers.
