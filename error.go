package z

import (
	"strconv"
	"strings"
)

// @spec REQ-CORE-005
// IssueCode represents the code of a validation issue.
type IssueCode string

const (
	// @spec REQ-CORE-005
	InvalidType      IssueCode = "invalid_type"
	TooBig           IssueCode = "too_big"
	TooSmall         IssueCode = "too_small"
	InvalidFormat    IssueCode = "invalid_format"
	NotMultipleOf    IssueCode = "not_multiple_of"
	UnrecognizedKeys IssueCode = "unrecognized_keys"
	InvalidUnion     IssueCode = "invalid_union"
	InvalidKey       IssueCode = "invalid_key"
	InvalidElement   IssueCode = "invalid_element"
	InvalidValue     IssueCode = "invalid_value"
	Custom           IssueCode = "custom"
)

// @spec REQ-CORE-006
// PathSegment represents a step in the data path: a string object key or integer slice/array index.
type PathSegment struct {
	key     string
	index   int
	isIndex bool
}

// @spec REQ-CORE-006
// Property creates a PathSegment for an object key.
func Property(key string) PathSegment {
	return PathSegment{key: key, isIndex: false}
}

// @spec REQ-CORE-006
// Index creates a PathSegment for a collection index.
func Index(idx int) PathSegment {
	return PathSegment{index: idx, isIndex: true}
}

// @spec REQ-CORE-006
// IsIndex returns true if this segment is a numeric index.
func (p PathSegment) IsIndex() bool {
	return p.isIndex
}

// @spec REQ-CORE-006
// Key returns the property key if this segment is not an index.
func (p PathSegment) Key() string {
	return p.key
}

// @spec REQ-CORE-006
// IndexVal returns the numeric index if this segment is an index.
func (p PathSegment) IndexVal() int {
	return p.index
}

// @spec REQ-CORE-006
// String formats the path segment as string.
func (p PathSegment) String() string {
	if p.isIndex {
		return strconv.Itoa(p.index)
	}
	return p.key
}

// @spec REQ-CORE-006
// Issue describes a single validation failure, matching Zod v4 error taxonomy.
type Issue struct {
	Code    IssueCode     `json:"code"`
	Path    []PathSegment `json:"path"`
	Message string        `json:"message"`
	Input   any           `json:"input,omitempty"`

	// invalid_type (Zod v4 has NO received field; received is formatted into Message)
	Expected string `json:"expected,omitempty"`

	// too_big / too_small
	Origin    string `json:"origin,omitempty"`
	Maximum   any    `json:"maximum,omitempty"`
	Minimum   any    `json:"minimum,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Exact     bool   `json:"exact,omitempty"`

	// invalid_format
	Format    string `json:"format,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Includes  string `json:"includes,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`

	// not_multiple_of
	Divisor any `json:"divisor,omitempty"`

	// unrecognized_keys
	Keys []string `json:"keys,omitempty"`

	// invalid_union
	Errors        [][]Issue `json:"errors,omitempty"`
	Discriminator string    `json:"discriminator,omitempty"`
	Options       []any     `json:"options,omitempty"`
	Matches       []int     `json:"matches,omitempty"`
	Note          string    `json:"note,omitempty"`

	// invalid_key / invalid_element
	Key    any     `json:"key,omitempty"`
	Issues []Issue `json:"issues,omitempty"`

	// invalid_value
	Values []any `json:"values,omitempty"`

	// custom
	Params map[string]any `json:"params,omitempty"`
}

// @spec REQ-CORE-006
// ZodError represents one or more validation issues found during schema parsing.
type ZodError struct {
	issues []Issue
}

// @spec REQ-CORE-006
// NewZodError constructs a *ZodError with the given issues.
func NewZodError(issues []Issue) *ZodError {
	return &ZodError{issues: issues}
}

// @spec REQ-CORE-006
// Issues returns the slice of issues contained in the error.
func (e *ZodError) Issues() []Issue {
	if e == nil {
		return nil
	}
	return e.issues
}

// @spec REQ-CORE-006
// Error implements the standard error interface.
func (e *ZodError) Error() string {
	if e == nil || len(e.issues) == 0 {
		return ""
	}
	return e.Prettify()
}

// formatPath formats a slice of PathSegments into dot/bracket notation (e.g. "users[0].name").
func formatPath(path []PathSegment) string {
	if len(path) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, seg := range path {
		if seg.IsIndex() {
			sb.WriteString("[")
			sb.WriteString(strconv.Itoa(seg.IndexVal()))
			sb.WriteString("]")
		} else {
			if i > 0 {
				sb.WriteString(".")
			}
			sb.WriteString(seg.Key())
		}
	}
	return sb.String()
}
