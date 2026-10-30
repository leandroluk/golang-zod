package z

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s OptionalSchema[T]) Meta(fn func(m *MetaBuilder)) OptionalSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s NullableSchema[T]) Meta(fn func(m *MetaBuilder)) NullableSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s NullishSchema[T]) Meta(fn func(m *MetaBuilder)) NullishSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s DefaultSchema[T]) Meta(fn func(m *MetaBuilder)) DefaultSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s PrefaultSchema[T]) Meta(fn func(m *MetaBuilder)) PrefaultSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s CatchSchema[T]) Meta(fn func(m *MetaBuilder)) CatchSchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s ReadonlySchema[T]) Meta(fn func(m *MetaBuilder)) ReadonlySchema[T] {
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	s.SetMeta(b.Build())
	s.updateSlot(s)
	return s
}

// Meta sets schema metadata (description, examples, ...) on the wrapper.
func (s *PtrSchema[T]) Meta(fn func(m *MetaBuilder)) *PtrSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}
