// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import "fmt"

// IdentifierIssuer issues new blank node identifiers, implementing the
// Issue Identifier Algorithm of RDFC-1.0 (spec section 4.4).
//
// An issuer is safe to clone: the temporary issuers created while hashing
// paths in the Hash N-Degree Quads algorithm are deep copies of a parent
// issuer.
type IdentifierIssuer struct {
	prefix  string
	counter int
	issued  map[string]string
	order   []string // identifiers in the order they were issued
}

func newIdentifierIssuer(prefix string) *IdentifierIssuer {
	return &IdentifierIssuer{prefix: prefix, issued: make(map[string]string)}
}

// issue returns the identifier issued for existing, creating one on first
// use (spec section 4.4).
func (i *IdentifierIssuer) issue(existing string) string {
	if id, ok := i.issued[existing]; ok {
		return id
	}
	id := fmt.Sprintf("%s%d", i.prefix, i.counter)
	i.counter++
	i.issued[existing] = id
	i.order = append(i.order, existing)
	return id
}

// getId returns the issued identifier for existing and whether one exists.
func (i *IdentifierIssuer) getId(existing string) (string, bool) {
	id, ok := i.issued[existing]
	return id, ok
}

// has reports whether an identifier has been issued for existing.
func (i *IdentifierIssuer) has(existing string) bool {
	_, ok := i.issued[existing]
	return ok
}

// issuedOrder returns the source identifiers in the order their canonical
// identifiers were issued. Used by the main algorithm to assign canonical
// identifiers in the temporary issuance order.
func (i *IdentifierIssuer) issuedOrder() []string { return i.order }

// mapping returns a copy of the issued identifiers map.
func (i *IdentifierIssuer) mapping() map[string]string {
	m := make(map[string]string, len(i.issued))
	for k, v := range i.issued {
		m[k] = v
	}
	return m
}

// clone deep copies the issuer.
func (i *IdentifierIssuer) clone() *IdentifierIssuer {
	c := &IdentifierIssuer{
		prefix:  i.prefix,
		counter: i.counter,
		issued:  make(map[string]string, len(i.issued)),
		order:   append([]string(nil), i.order...),
	}
	for k, v := range i.issued {
		c.issued[k] = v
	}
	return c
}
