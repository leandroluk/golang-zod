package z_test

import (
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-CORE-010
func TestCoverERegisterSchemaIDEmpty(t *testing.T) {
	a, b := &struct{ n int }{1}, &struct{ n int }{2}
	// Empty IDs are never registered, so any number of owners may use them.
	z.RegisterSchemaID("", a)
	z.RegisterSchemaID("", b)

	z.RegisterSchemaID("coverE_registry_x", a)
	z.RegisterSchemaID("coverE_registry_x", a) // same owner: fine
	func() {
		defer func() {
			if recover() == nil {
				t.Error("different owner must panic")
			}
		}()
		z.RegisterSchemaID("coverE_registry_x", b)
	}()
}
