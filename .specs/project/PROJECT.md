# Project: golang-zod

Traceability: on

## Vision
A complete, idiomatic Go port of **Zod v4** focused on a **pure runtime fluent DSL and dual paradigm** (Schema-First via `z.Object(z.Shape{...})` and Type-First via `z.Object[T]()` with zero-reflection in hot paths), delivering ergonomic, high-throughput data validation for modern Go backends and microservices.

## Core Principles
1. **Zero-Reflection in Hot Paths**: Reflection is used strictly during startup/compilation of `z.Object[T]()`. Hot execution paths run through compiled AST nodes and function closures.
2. **Dual Paradigms (Schema-First & Type-First)**:
   - Dynamic: `z.Object(z.Shape{"name": z.String().Min(2)})`
   - Static/Typed: `z.Object[User]()` reading `json:"..."` and `zod:"..."` tags, with `ParseInto(&dest)` and typed returns `z.Parse[T](schema, input)`.
3. **Zod v4 Pure Spec**: All formats top-level (`z.Email()`, `z.UUID()`, `z.ISO.DateTime()`, `z.Int()`), zero legacy v3 deprecations.
4. **Default Validation Messages & i18n**: Out-of-the-box human-friendly validation messages with multi-locale support (`en`, `pt-BR`, `es`), structured path tracking (`ZodIssue`), and the 4 Zod v4 formatters (`Format`, `Flatten`, `Treeify`, `Prettify`).
5. **Concurrency-Safe & Zero-Allocation Conscious**: Imutabilidade de schemas por padrão para compartilhamento livre entre goroutines sem locks, aliado a `sync.Pool` para reutilização de contextos de execução e buffers.
6. **Multi-Target Exporters**: Geração nativa de **JSON Schema** (Draft 2020-12 / Draft 07) e **OpenAPI 3.1**, além de importação reversa via `z.FromJSONSchema`.
