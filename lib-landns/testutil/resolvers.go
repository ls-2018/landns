package testutil

import (
	"github.com/macrat/landns/lib-landns"
)

// DummyResolver is stub of landns.Resolver.
type DummyResolver struct {
	Error     bool
	Recursion bool
}

// Resolve is returns error if DummyResolver.Error is true, otherwise nothing to do.
func (dr DummyResolver) Resolve(w landns.ResponseWriter, r landns.Request) error {
	if dr.Error {
		return landns.Error{Type: landns.TypeInternalError, Message: "test error"}
	}
	return nil
}

// RecursionAvailable is returns value of DummyResolver.Recursion.
func (dr DummyResolver) RecursionAvailable() bool {
	return dr.Recursion
}

// Close is nothing to do.
func (dr DummyResolver) Close() error {
	return nil
}
