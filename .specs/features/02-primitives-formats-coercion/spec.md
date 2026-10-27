# Spec: 02 — Primitives, Top-Level String Formats, Numbers & Coercion

## Summary
Implements Milestone 2 per [INSIGHT.md](../../../INSIGHT.md) §1:
1. Primitives: `z.String()`, `z.Boolean()`, `z.Null()`, `z.Any()`, `z.Unknown()`, `z.Never()`, `z.Date()` (`time.Time`), `z.Bytes()`, `z.Literal(...)`.
2. `StringSchema` checks (preserve `string`, fluent): `.Min(n)`, `.Max(n)`, `.Length(n)`, `.NonEmpty()`, `.Regex(re *regexp.Regexp)`, `.Includes(s)`, `.StartsWith(s)`, `.EndsWith(s)`, `.Lowercase()`, `.Uppercase()`; transforms: `.Trim()`, `.ToLowerCase()`, `.ToUpperCase()`, `.Slugify()`, `.Normalize()` (see REQ-PRIM-002).
3. Top-level formats returning `*StringSchema` (format-tagged):
   - Identifiers: `UUID`, `UUIDv4`, `UUIDv6`, `UUIDv7`, `GUID`, `NanoID`, `CUID2`, `ULID`, `XID`, `KSUID`.
   - Web & network: `Email`, `URL`, `HTTPURL`, `Hostname`, `IPv4`, `IPv6`, `MAC`, `CIDRv4`, `CIDRv6`.
   - Coding & tokens: `Base64`, `Base64URL`, `Hex`, `JWT`, `Emoji`, `E164`, `CreditCard`, `IBAN`, `CurrencyCode`, `Hash(alg, opts)`, `StringFormat(name, check)`.
   - ISO under the namespace value `z.ISO`: `z.ISO.Date()`, `z.ISO.Time()`, `z.ISO.DateTime()`, `z.ISO.Duration()`.
4. Numbers: `*NumberSchema[T]` over `type number interface{ ~int|~int8|~int16|~int32|~int64|~uint|~uint8|~uint16|~uint32|~uint64|~float32|~float64 }`. Constructors `Number()` (float64), `Float64()`, `Float32()`, `Int()`, `Int8()`, `Int16()`, `Int32()`, `Int64()`, `Uint()`, `Uint8()`, `Uint16()`, `Uint32()`, `Uint64()`, `BigInt()` (`*BigIntSchema`, output `*big.Int`).
5. Checks (preserve type): `.GT(v)`, `.GTE(v)`/`.Min(v)`, `.LT(v)`, `.LTE(v)`/`.Max(v)`, `.Positive()`, `.NonNegative()`, `.Negative()`, `.NonPositive()`, `.MultipleOf(v)`.
6. Coercion under namespace value `z.Coerce`: `String()`, `Number()`, `Int()`, `Float()`, `Boolean()`, `BigInt()`, `Date()`.

## Requirements
- REQ-PRIM-001: Implement `Boolean`, `Null` (a `Schema[any]` that accepts only `null` and outputs `nil`), `Any`, `Unknown` (both `Schema[any]`), `Never` (always `InvalidType`), `Date` (accepts `time.Time` and RFC 3339 strings), `Bytes` (`[]byte`, accepts base64 string in JSON input), and `Literal[T comparable](v ...T) Schema[T]` (multiple values allowed; failure is `InvalidValue`).
- REQ-PRIM-002: String checks and transforms as listed. A transform returns `*StringSchema` and is applied before subsequent checks in chain order. `.Normalize()` is implemented only if it can be done with the standard library alone; otherwise it MUST NOT be exported and the decision recorded in STATE.md.
- REQ-PRIM-003: Identifier formats (`UUID` accepts any RFC 9562 version; `UUIDv4/6/7` check the version nibble; `GUID` is the loose 8-4-4-4-12 hex). Failure code `InvalidFormat` with `Format` set to the format name.
- REQ-PRIM-004: Network/web formats. `Email` uses the Zod v4 default regex (not `net/mail`); `URL` parses with `net/url` and requires scheme+host; `HTTPURL` restricts to `http`/`https`; `IPv4/IPv6` use `net/netip`; `MAC`, `CIDRv4`, `CIDRv6`, `Hostname` per RFC 1123.
- REQ-PRIM-005: Encoding/token formats (`Base64`, `Base64URL`, `Hex`, `JWT` (3 base64url segments, header decodes to JSON with `alg`), `Emoji`, `E164`, `CreditCard` (Luhn), `IBAN` (mod-97), `CurrencyCode` (ISO 4217 list), `Hash`, `StringFormat`).
- REQ-PRIM-006: `z.ISO` namespace (`var ISO isoNamespace`, methods `Date/Time/DateTime/Duration`). `DateTime` options: precision, offset allowed, local allowed.
- REQ-PRIM-007: Numeric constructors and checks. Integer kinds reject non-integral and out-of-range values with `InvalidType`/`TooBig`/`TooSmall` (`Origin: "number"`). `Int64`, `Uint64`, `BigInt` parse from `json.Number` text, never via `float64`. `NaN` and `±Inf` are rejected by default. `MultipleOf` failure is `NotMultipleOf`.
- REQ-PRIM-008: `z.Coerce` namespace (`var Coerce coerceNamespace`): coercion runs before validation (`String` uses `fmt`-free `strconv`/`json` text; `Boolean` follows JS truthiness only for the documented inputs; `Number/Int/Float/BigInt` parse strings; `Date` accepts RFC 3339 and unix millis).
- REQ-PRIM-009: Every schema type in this feature carries the standard modifier set declared explicitly: `.Optional()`, `.Nullable()`, `.Nullish()`, `.Default(v)`, `.Array()`, `.Meta(m)` (implemented in feature 06 but the method stubs are added here per type so the pattern is consistent).

## Affected Components
- `string.go`, `formats.go`, `iso.go`, `number.go`, `primitives.go`, `coerce.go`
- depends on feature 01 (`node`, `parseCtx`, `Issue`).

## Out of Scope
- `z.number().int()`-style v3 forms; `z.Symbol`, `z.Undefined`, `z.Void`, `z.NaN`.
- Third-party regex libraries.

## Open Questions
- Keep both `Min/GTE` and `Max/LTE` aliases (Zod has both). Decision: **keep** (recorded in STATE.md).
