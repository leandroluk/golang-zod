package z

import (
	"fmt"
	"slices"
	"sync"
)

// @spec REQ-CORE-010
// Meta represents immutable schema metadata.
type Meta struct {
	ID          string         `json:"id,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Examples    []any          `json:"examples,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty"`
	ReadOnly    bool           `json:"readOnly,omitempty"`
	Custom      map[string]any `json:"custom,omitempty"`

	errOv *errorOverride

	// lineage identifies the schema family that registered ID: clones derived from the same schema
	// (via modifiers/checks) share it, so re-registering the same ID from a clone is not a duplicate.
	lineage     *metaLineage
	isDefaultID bool
}

// metaLineage is an opaque token identifying the origin of a registered schema ID.
type metaLineage struct{ _ byte }

// @spec REQ-CORE-010
// MetaBuilder provides a fluent builder to configure schema metadata.
type MetaBuilder struct {
	meta Meta
}

var (
	idRegistryMu sync.Mutex
	idRegistry   = make(map[string]any)
)

// RegisterSchemaID registers a non-empty schema ID for uniqueness checks.
// Panics if the same non-empty ID is registered by a different schema.
func RegisterSchemaID(id string, owner any) {
	if id == "" {
		return
	}
	idRegistryMu.Lock()
	defer idRegistryMu.Unlock()

	if existingOwner, exists := idRegistry[id]; exists {
		if existingOwner != owner {
			panic(fmt.Sprintf("duplicate schema ID %q: already registered by another schema", id))
		}
		return
	}
	idRegistry[id] = owner
}

// NewMetaBuilder creates a new MetaBuilder initialized with a copy of current Meta.
func NewMetaBuilder(current Meta) *MetaBuilder {
	b := &MetaBuilder{
		meta: current,
	}
	if len(current.Examples) > 0 {
		b.meta.Examples = slices.Clone(current.Examples)
	}
	if len(current.Custom) > 0 {
		b.meta.Custom = make(map[string]any, len(current.Custom))
		for k, v := range current.Custom {
			b.meta.Custom[k] = v
		}
	}
	return b
}

// @spec REQ-CORE-010
// ID sets the unique schema identifier.
func (b *MetaBuilder) ID(id string) *MetaBuilder {
	b.meta.isDefaultID = false
	if id != b.meta.ID {
		b.meta.lineage = nil
	}
	b.meta.ID = id
	return b
}

// @spec REQ-CORE-010
// Title sets the title of the schema.
func (b *MetaBuilder) Title(title string) *MetaBuilder {
	b.meta.Title = title
	return b
}

// @spec REQ-CORE-010
// Description sets the description of the schema (Zod's .describe()).
func (b *MetaBuilder) Description(desc string) *MetaBuilder {
	b.meta.Description = desc
	return b
}

// @spec REQ-CORE-010
// Examples appends example values to the schema metadata. T is inferred, so a typed slice
// can be spread directly (m.Examples(values...)); use Examples[any] to mix value types.
func (b *MetaBuilder) Examples[T any](examples ...T) *MetaBuilder {
	for _, e := range examples {
		b.meta.Examples = append(b.meta.Examples, e)
	}
	return b
}

// @spec REQ-CORE-010
// ReadOnly marks whether the schema is readOnly.
func (b *MetaBuilder) ReadOnly(ro bool) *MetaBuilder {
	b.meta.ReadOnly = ro
	return b
}

// @spec REQ-CORE-010
// Deprecated marks whether the schema is deprecated.
func (b *MetaBuilder) Deprecated(deprecated bool) *MetaBuilder {
	b.meta.Deprecated = deprecated
	return b
}

// @spec REQ-CORE-010
// Custom sets a custom metadata key-value pair.
func (b *MetaBuilder) Custom(key string, val any) *MetaBuilder {
	if b.meta.Custom == nil {
		b.meta.Custom = make(map[string]any)
	}
	b.meta.Custom[key] = val
	return b
}

// @spec REQ-CORE-010
// Reset clears all metadata, discarding inherited values.
func (b *MetaBuilder) Reset() *MetaBuilder {
	b.meta = Meta{}
	return b
}

// Build returns the compiled immutable Meta struct.
func (b *MetaBuilder) Build() Meta {
	return b.meta
}

// GetMeta extracts metadata from any Schema or schema object.
func GetMeta(s any) Meta {
	if s == nil {
		return Meta{}
	}
	if g, ok := s.(interface{ getMeta() Meta }); ok {
		return g.getMeta()
	}
	if g, ok := s.(interface{ GetMeta() Meta }); ok {
		return g.GetMeta()
	}
	return Meta{}
}
