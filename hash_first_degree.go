// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import (
	"fmt"
	"sort"
	"strings"
)

// hashFirstDegreeQuads implements the Hash First Degree Quads algorithm
// (spec section 4.5). It returns the SHA-256 (or configured hash) of the
// sorted canonical n-quads serialization of all quads mentioning the
// reference blank node, with the reference blank node serialized as _:a and
// every other blank node as _:z.
func (s *canonicalizationState) hashFirstDegreeQuads(reference string) (string, error) {
	nquads := make([]string, 0, len(s.bnodeQuads[reference]))
	for _, idx := range s.bnodeQuads[reference] {
		q := s.quads[idx]
		if q.Subject.IsBlankNode() {
			q.Subject.Value = placeholder(reference, q.Subject.Value)
		}
		if q.Object.IsBlankNode() {
			q.Object.Value = placeholder(reference, q.Object.Value)
		}
		if q.Graph != nil && q.Graph.IsBlankNode() {
			g := *q.Graph
			g.Value = placeholder(reference, g.Value)
			q.Graph = &g
		}
		var sb strings.Builder
		writeQuad(&sb, q)
		sb.WriteString(" .\n")
		nquads = append(nquads, sb.String())
	}
	sort.Strings(nquads)
	return s.opts.hashFn(strings.Join(nquads, "")), nil
}

// placeholder returns "a" if identifier is the reference blank node, else "z".
func placeholder(reference, identifier string) string {
	if identifier == reference {
		return "a"
	}
	return "z"
}

// hashRelatedBlankNode implements the Hash Related Blank Node algorithm
// (spec section 4.7).
func (s *canonicalizationState) hashRelatedBlankNode(related string, quad Quad, issuer *IdentifierIssuer, position string) (string, error) {
	// Step 1: input is the position.
	// Step 2: if position is not g, append the predicate.
	var sb strings.Builder
	sb.WriteString(position)
	if position != "g" {
		sb.WriteByte('<')
		escapeIRI(&sb, quad.Predicate.Value)
		sb.WriteByte('>')
	}
	// Step 3-4: append the identifier of related, preferring the canonical
	// identifier, then an identifier issued by the temporary issuer, and
	// falling back to the first degree hash of related.
	if id, ok := s.issuer.getId(related); ok {
		sb.WriteString("_:")
		sb.WriteString(id)
	} else if id, ok := issuer.getId(related); ok {
		sb.WriteString("_:")
		sb.WriteString(id)
	} else {
		hf, err := s.hashFirstDegreeQuads(related)
		if err != nil {
			return "", fmt.Errorf("hash related blank node: %w", err)
		}
		sb.WriteString(hf)
	}
	return s.opts.hashFn(sb.String()), nil
}
