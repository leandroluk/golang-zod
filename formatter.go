package z

import (
	"fmt"
	"slices"
	"strings"
)

// @spec REQ-CORE-007
// FlattenedErrors represents errors divided between top-level form errors
// and first-segment field errors, matching Zod v4 `flattenError`.
type FlattenedErrors struct {
	FormErrors  []string            `json:"formErrors"`
	FieldErrors map[string][]string `json:"fieldErrors"`
}

// @spec REQ-CORE-007
// Flatten splits issues into top-level form errors (empty path) and field errors,
// keyed strictly by the first path segment.
func (e *ZodError) Flatten() FlattenedErrors {
	res := FlattenedErrors{
		FormErrors:  make([]string, 0),
		FieldErrors: make(map[string][]string),
	}
	if e == nil {
		return res
	}
	for _, issue := range e.issues {
		if len(issue.Path) == 0 {
			res.FormErrors = append(res.FormErrors, issue.Message)
		} else {
			first := issue.Path[0].String()
			res.FieldErrors[first] = append(res.FieldErrors[first], issue.Message)
		}
	}
	return res
}

// @spec REQ-CORE-007
// ErrorTreeNode represents a hierarchical node in the error tree produced by Treeify.
type ErrorTreeNode struct {
	Errors     []string                 `json:"errors,omitempty"`
	Properties map[string]ErrorTreeNode `json:"properties,omitempty"`
	Items      map[int]ErrorTreeNode    `json:"items,omitempty"`
}

// @spec REQ-CORE-007
// Treeify organizes issues into a nested tree matching object properties and array items.
func (e *ZodError) Treeify() ErrorTreeNode {
	root := ErrorTreeNode{}
	if e == nil {
		return root
	}

	for _, issue := range e.issues {
		root = insertTreeNode(root, issue.Path, issue.Message)
	}
	return root
}

func insertTreeNode(node ErrorTreeNode, path []PathSegment, msg string) ErrorTreeNode {
	if len(path) == 0 {
		node.Errors = append(node.Errors, msg)
		return node
	}

	head := path[0]
	tail := path[1:]

	if head.IsIndex() {
		idx := head.IndexVal()
		if node.Items == nil {
			node.Items = make(map[int]ErrorTreeNode)
		}
		child := node.Items[idx]
		node.Items[idx] = insertTreeNode(child, tail, msg)
	} else {
		prop := head.Key()
		if node.Properties == nil {
			node.Properties = make(map[string]ErrorTreeNode)
		}
		child := node.Properties[prop]
		node.Properties[prop] = insertTreeNode(child, tail, msg)
	}

	return node
}

// @spec REQ-CORE-007
// Prettify formats issues into human-readable multi-line strings, sorted by path length.
// Each issue starts with '✖ message' and, if path is non-empty, adds '  → at path'.
func (e *ZodError) Prettify() string {
	if e == nil || len(e.issues) == 0 {
		return ""
	}

	// Sort copies of issues by path length ascending
	sorted := slices.Clone(e.issues)
	slices.SortStableFunc(sorted, func(a, b Issue) int {
		return len(a.Path) - len(b.Path)
	})

	var lines []string
	for _, issue := range sorted {
		msg := issue.Message
		if msg == "" {
			msg = string(issue.Code)
		}
		lines = append(lines, fmt.Sprintf("✖ %s", msg))
		if p := formatPath(issue.Path); p != "" {
			lines = append(lines, fmt.Sprintf("  → at %s", p))
		}
	}

	return strings.Join(lines, "\n")
}
