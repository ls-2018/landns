package testutil

import (
	"github.com/macrat/landns/lib-landns"
)

// DummyResponseWriter is array stub of landns.ResponseWriter.
type DummyResponseWriter struct {
	Records       []landns.Record
	Authoritative bool
}

// NewDummyResponseWriter is constructor of DummyResponseWriter.
func NewDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{
		Records:       make([]landns.Record, 0, 10),
		Authoritative: true,
	}
}

// Add is adding record into DummyResponseWriter.Records.
func (rw *DummyResponseWriter) Add(r landns.Record) error {
	rw.Records = append(rw.Records, r)
	return nil
}

// IsAuthoritative is returns value of DummyResponseWriter.Authoritative.
func (rw *DummyResponseWriter) IsAuthoritative() bool {
	return rw.Authoritative
}

// SetNoAuthoritative is set value to DummyResponseWriter.Authoritative.
func (rw *DummyResponseWriter) SetNoAuthoritative() {
	rw.Authoritative = false
}

// EmptyResponseWriter is empty stub of landns.ResponseWriter.
type EmptyResponseWriter struct{}

// Add is nothing to do.
func (rw EmptyResponseWriter) Add(r landns.Record) error {
	return nil
}

// IsAuthoritative is always returns true.
func (rw EmptyResponseWriter) IsAuthoritative() bool {
	return true
}

// SetNoAuthoritative is nothing to do.
func (rw EmptyResponseWriter) SetNoAuthoritative() {
}
