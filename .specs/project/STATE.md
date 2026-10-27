# Project State: golang-zod

## Active Context
- **Current Milestone**: `M1-foundation`
- **Active Feature**: Specifying and initializing repository foundation (`01-core-error-i18n`).
- **Focus**: Porting pure Zod v4 to Go with zero-reflection in hot paths, dual paradigms (Schema-First & Type-First), structured error taxonomy and i18n.

## Recent Progress (last 5 items)
- Initialized repository with Go module `github.com/leandroluk/golang-zod`.
- Defined full architecture, paradigms, and Zod v4 mapping in `INSIGHT.md`.
- Established `.specs/` graph-spec-design structure (`PROJECT.md`, `ROADMAP.md`, `STATE.md`).

## Recent Decisions (last 5 items)
- **ADR-001**: Pure Zod v4 spec only — reject v3 legacy deprecations (`z.string().email()`, `passthrough()`, etc.).
- **ADR-002**: Bootstrap-only reflection for `z.Object[T]()` — build and cache immutable AST nodes once; zero reflection in hot validation loops.
- **ADR-003**: Native concurrency safety — schemas are immutable value pointers, safe to share across goroutines without mutexes.
- **ADR-004**: Memory efficiency via `sync.Pool` for `ParseContext` and issue path buffers.

## Blockers & Open Questions
- None.

## Cost
- Initial index sync pending.
