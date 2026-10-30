package z

// Meta sets schema metadata (description, examples, ...).
func (s *CodecSchema[A, B]) Meta(fn func(m *MetaBuilder)) *CodecSchema[A, B] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *CustomSchema[T]) Meta(fn func(m *MetaBuilder)) *CustomSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *JSONSchema) Meta(fn func(m *MetaBuilder)) *JSONSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *NaNSchema) Meta(fn func(m *MetaBuilder)) *NaNSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *NonOptionalSchema[T]) Meta(fn func(m *MetaBuilder)) *NonOptionalSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *PipeSchema[T, U]) Meta(fn func(m *MetaBuilder)) *PipeSchema[T, U] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *PreprocessSchema[T]) Meta(fn func(m *MetaBuilder)) *PreprocessSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *RefinedSchema[T]) Meta(fn func(m *MetaBuilder)) *RefinedSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *SuccessSchema) Meta(fn func(m *MetaBuilder)) *SuccessSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// Meta sets schema metadata (description, examples, ...).
func (s *TransformSchema[T, U]) Meta(fn func(m *MetaBuilder)) *TransformSchema[T, U] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}
