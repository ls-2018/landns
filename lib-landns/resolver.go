package landns

import (
	"fmt"
	"io"
)

// Resolver is the interface of record resolver.
type Resolver interface {
	io.Closer

	Resolve(ResponseWriter, Request) error
	RecursionAvailable() bool // Checking that recursion resolve is available or not.
}

// ResolverSet is list of Resolver.
//
// ResolverSet will merge all resolver's responses unlike AlternateResolver.
type ResolverSet []Resolver

// Resolve is resolver using all upstream resolvers.
func (rs ResolverSet) Resolve(resp ResponseWriter, req Request) error {
	for _, r := range rs {
		if err := r.Resolve(resp, req); err != nil {
			return err
		}
	}
	return nil
}

// RecursionAvailable is returns `true` if upstream resolvers at least one returns `true`.
func (rs ResolverSet) RecursionAvailable() bool {
	for _, r := range rs {
		if r.RecursionAvailable() {
			return true
		}
	}
	return false
}

// Close is close all upstream resolvers.
func (rs ResolverSet) Close() error {
	for _, r := range rs {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// String is returns simple human readable string.
func (rs ResolverSet) String() string {
	return fmt.Sprintf("ResolverSet%s", []Resolver(rs))
}

// AlternateResolver is list of Resolver.
//
// AlternateResolver will response only first respond unlike ResolverSet.
type AlternateResolver []Resolver

// Resolve is resolver using first respond upstream resolvers.
func (ar AlternateResolver) Resolve(resp ResponseWriter, req Request) error {
	resolved := false

	resp = ResponseWriterHook{
		Writer: resp,
		OnAdd: func(r Record) error {
			resolved = true
			return nil
		},
	}

	for _, r := range ar {
		if err := r.Resolve(resp, req); err != nil {
			return err
		}

		if resolved {
			return nil
		}
	}
	return nil
}

// RecursionAvailable is returns `true` if upstream resolvers at least one returns `true`.
func (ar AlternateResolver) RecursionAvailable() bool {
	for _, r := range ar {
		if r.RecursionAvailable() {
			return true
		}
	}
	return false
}

// Close is close all upstream resolvers.
func (ar AlternateResolver) Close() error {
	for _, r := range ar {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// String is returns simple human readable string.
func (ar AlternateResolver) String() string {
	return fmt.Sprintf("AlternateResolver%s", []Resolver(ar))
}
