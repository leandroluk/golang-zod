# Architecture: golang-zod

## 1. Overview & Directory Structure
```
golang-zod/
├── INSIGHT.md           # Master design document and Zod v4 specifications
├── go.mod               # Go module definition (go 1.22+)
├── .specs/              # Graph-spec-design specifications & NexSpec index
│   ├── project/         # PROJECT.md, ROADMAP.md, STATE.md
│   ├── codebase/        # ARCHITECTURE.md, CONCERNS.md, CONVENTIONS.md
│   └── features/        # Feature specs, designs, and tasks
├── z.go                 # Top-level public constructors (z.String(), z.Email(), z.Object())
├── schema.go            # Core Schema interface and AST Node definitions
├── value.go             # Runtime Value abstraction (Null, Bool, Int, Float, String, Object...)
├── error.go             # 11 Zod v4 IssueCodes, ZodError and 4 formatters
├── i18n.go              # Locale management, pluralization and message formatting
└── type_object.go       # Type-First z.Object[T]() reflection-to-AST compiler with cache
```

## 2. Core Components

### 2.1. Value Model (`value.go`)
Represents untyped and deserialized JSON/Go values with type-safe accessors (`AsInt()`, `AsString()`, `AsMap()`).

### 2.2. Schema & Validator Engine (`schema.go`)
* `Schema` interface providing `Parse(data any) (any, error)`, `SafeParse(data any) *SafeParseResult`, `ParseInto(data any, target any) error`.
* Immutable AST node representations (`StringNode`, `NumberNode`, `ObjectNode`, `ArrayNode`).

### 2.3. Type-First Compiler (`type_object.go`)
* Uses Go `reflect` **strictly at initialization time** to inspect struct types, `json:"..."` tags, and `zod:"..."` validation rules.
* Caches the resulting immutable AST tree globally per type `T`.

### 2.4. Error & i18n Engine (`error.go`, `i18n.go`)
* Structured issues with paths (`users.0.email`), 11 Zod v4 issue codes, and formatters (`Format`, `Flatten`, `Treeify`, `Prettify`).
* Pluggable locales (`LocaleEnUS`, `LocalePtBR`).
