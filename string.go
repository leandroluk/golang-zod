package z

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

type stringCheckKind int

const (
	checkMin stringCheckKind = iota
	checkMax
	checkLength
	checkRegex
	checkIncludes
	checkStartsWith
	checkEndsWith
	checkLowercase
	checkUppercase
	checkCustom
)

type stringCheck struct {
	kind      stringCheckKind
	intVal    int
	strVal    string
	regex     *regexp.Regexp
	custom    func(s string) bool
	formatTag string
}

type stringTransformKind int

const (
	transformTrim stringTransformKind = iota
	transformToLowerCase
	transformToUpperCase
	transformSlugify
)

// @spec REQ-PRIM-002
// StringSchema validates and transforms string inputs.
type StringSchema struct {
	BaseSchema[string]
	checks     []stringCheck
	transforms []stringTransformKind
	formatName string
}

type stringNode struct {
	schema *StringSchema
}

func (n *stringNode) kind() nodeKind { return kindString }
func (n *stringNode) parse(ctx *parseCtx, in any) (any, bool) {
	val, ok := in.(string)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "string",
			Input:    in,
		})
		return "", false
	}

	// 1. Apply transforms in order
	for _, t := range n.schema.transforms {
		switch t {
		case transformTrim:
			val = strings.TrimSpace(val)
		case transformToLowerCase:
			val = strings.ToLower(val)
		case transformToUpperCase:
			val = strings.ToUpper(val)
		case transformSlugify:
			val = slugify(val)
		}
	}

	// 2. Apply checks in order
	allOk := true
	for _, ch := range n.schema.checks {
		if err := ctx.checkCancel(); err != nil {
			return "", false
		}
		switch ch.kind {
		case checkMin:
			if len(val) < ch.intVal {
				ctx.addIssue(Issue{
					Code:      TooSmall,
					Origin:    "string",
					Minimum:   ch.intVal,
					Inclusive: true,
					Input:     val,
				})
				allOk = false
			}

		case checkMax:
			if len(val) > ch.intVal {
				ctx.addIssue(Issue{
					Code:      TooBig,
					Origin:    "string",
					Maximum:   ch.intVal,
					Inclusive: true,
					Input:     val,
				})
				allOk = false
			}

		case checkLength:
			if len(val) != ch.intVal {
				ctx.addIssue(Issue{
					Code:      TooSmall,
					Origin:    "string",
					Minimum:   ch.intVal,
					Exact:     true,
					Inclusive: true,
					Input:     val,
				})
				allOk = false
			}

		case checkRegex:
			if !ch.regex.MatchString(val) {
				ctx.addIssue(Issue{
					Code:    InvalidFormat,
					Format:  "regex",
					Pattern: ch.regex.String(),
					Input:   val,
				})
				allOk = false
			}

		case checkIncludes:
			if !strings.Contains(val, ch.strVal) {
				ctx.addIssue(Issue{
					Code:     InvalidFormat,
					Format:   "includes",
					Includes: ch.strVal,
					Input:    val,
				})
				allOk = false
			}

		case checkStartsWith:
			if !strings.HasPrefix(val, ch.strVal) {
				ctx.addIssue(Issue{
					Code:   InvalidFormat,
					Format: "starts_with",
					Prefix: ch.strVal,
					Input:  val,
				})
				allOk = false
			}

		case checkEndsWith:
			if !strings.HasSuffix(val, ch.strVal) {
				ctx.addIssue(Issue{
					Code:   InvalidFormat,
					Format: "ends_with",
					Suffix: ch.strVal,
					Input:  val,
				})
				allOk = false
			}

		case checkLowercase:
			if strings.ToLower(val) != val {
				ctx.addIssue(Issue{
					Code:    InvalidFormat,
					Format:  "lowercase",
					Pattern: "^[a-z0-9_]*$",
					Input:   val,
				})
				allOk = false
			}

		case checkUppercase:
			if strings.ToUpper(val) != val {
				ctx.addIssue(Issue{
					Code:    InvalidFormat,
					Format:  "uppercase",
					Pattern: "^[A-Z0-9_]*$",
					Input:   val,
				})
				allOk = false
			}

		case checkCustom:
			if !ch.custom(val) {
				format := ch.formatTag
				if format == "" {
					format = n.schema.formatName
				}
				ctx.addIssue(Issue{
					Code:   InvalidFormat,
					Format: format,
					Input:  val,
				})
				allOk = false
			}
		}

		if !allOk && ctx.shouldAbort() {
			break
		}
	}

	if !allOk {
		return "", false
	}
	return val, true
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var sb strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			dash = false
		} else if !dash && sb.Len() > 0 {
			sb.WriteRune('-')
			dash = true
		}
	}
	res := sb.String()
	return strings.TrimSuffix(res, "-")
}

// @spec REQ-PRIM-002
// String creates a new string schema.
func String() *StringSchema {
	s := &StringSchema{}
	s.BaseSchema = NewBaseSchema[string](&stringNode{schema: s})
	return s
}

func (s *StringSchema) clone() *StringSchema {
	c := &StringSchema{
		checks:     slices.Clone(s.checks),
		transforms: slices.Clone(s.transforms),
		formatName: s.formatName,
	}
	c.BaseSchema = NewBaseSchema[string](&stringNode{schema: c})
	c.SetMeta(s.GetMeta())
	return c
}

// @spec REQ-PRIM-002
// Min enforces a minimum string character length.
func (s *StringSchema) Min(n int) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkMin, intVal: n})
	return c
}

// @spec REQ-PRIM-002
// NonEmpty enforces that string is not empty (length >= 1).
func (s *StringSchema) NonEmpty() *StringSchema {
	return s.Min(1)
}

// @spec REQ-PRIM-002
// Max enforces a maximum string character length.
func (s *StringSchema) Max(n int) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkMax, intVal: n})
	return c
}

// @spec REQ-PRIM-002
// Length enforces exact character length.
func (s *StringSchema) Length(n int) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkLength, intVal: n})
	return c
}

// @spec REQ-PRIM-002
// Regex validates string against a compiled regular expression.
func (s *StringSchema) Regex(re *regexp.Regexp) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkRegex, regex: re})
	return c
}

// @spec REQ-PRIM-002
// Includes checks if string contains target substring.
func (s *StringSchema) Includes(sub string) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkIncludes, strVal: sub})
	return c
}

// @spec REQ-PRIM-002
// StartsWith checks if string begins with prefix.
func (s *StringSchema) StartsWith(prefix string) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkStartsWith, strVal: prefix})
	return c
}

// @spec REQ-PRIM-002
// EndsWith checks if string terminates with suffix.
func (s *StringSchema) EndsWith(suffix string) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkEndsWith, strVal: suffix})
	return c
}

// @spec REQ-PRIM-002
// Lowercase requires all cased characters in the string to be lowercase.
func (s *StringSchema) Lowercase() *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkLowercase})
	return c
}

// @spec REQ-PRIM-002
// Uppercase requires all cased characters in the string to be uppercase.
func (s *StringSchema) Uppercase() *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{kind: checkUppercase})
	return c
}

// @spec REQ-PRIM-002
// Trim transforms string by removing leading/trailing whitespace before validation.
func (s *StringSchema) Trim() *StringSchema {
	c := s.clone()
	c.transforms = append(c.transforms, transformTrim)
	return c
}

// @spec REQ-PRIM-002
// ToLowerCase transforms string to lower case before validation.
func (s *StringSchema) ToLowerCase() *StringSchema {
	c := s.clone()
	c.transforms = append(c.transforms, transformToLowerCase)
	return c
}

// @spec REQ-PRIM-002
// ToUpperCase transforms string to upper case before validation.
func (s *StringSchema) ToUpperCase() *StringSchema {
	c := s.clone()
	c.transforms = append(c.transforms, transformToUpperCase)
	return c
}

// @spec REQ-PRIM-002
// Slugify transforms string into a lowercase URL slug before validation.
func (s *StringSchema) Slugify() *StringSchema {
	c := s.clone()
	c.transforms = append(c.transforms, transformSlugify)
	return c
}

// WithCustomCheck attaches an internal format check.
func (s *StringSchema) WithCustomCheck(name string, fn func(string) bool) *StringSchema {
	c := s.clone()
	c.checks = append(c.checks, stringCheck{
		kind:      checkCustom,
		formatTag: name,
		custom:    fn,
	})
	if c.formatName == "" {
		c.formatName = name
	}
	return c
}

// @spec REQ-PRIM-009
func (s *StringSchema) Optional() OptionalSchema[string] { return NewOptionalSchema[string](s) }
func (s *StringSchema) Nullable() NullableSchema[string] { return NewNullableSchema[string](s) }
func (s *StringSchema) Nullish() NullishSchema[string]   { return NewNullishSchema[string](s) }
func (s *StringSchema) Default(v string) DefaultSchema[string] {
	return NewDefaultSchema[string](s, v)
}
func (s *StringSchema) Array() ArraySchema[string] { return NewArraySchema[string](s) }
func (s *StringSchema) Meta(fn func(m *MetaBuilder)) *StringSchema {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}
