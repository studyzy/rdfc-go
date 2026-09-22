// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import (
	"fmt"
	"strings"
)

// hashNDegreeQuads implements the Hash N-Degree Quads algorithm
// (spec section 4.8). It returns a hash for identifier together with the
// identifier issuer (a copy of pathIssuer extended with temporary
// identifiers) that produced the chosen path.
func (s *canonicalizationState) hashNDegreeQuads(identifier string, pathIssuer *IdentifierIssuer) (string, *IdentifierIssuer, error) {
	// Poison guard: bound the total number of hash n-degree calls and
	// permutations processed for one canonicalization run.
	s.hnCalls++
	if s.hnCalls > s.opts.maxCalls {
		return "", nil, fmt.Errorf("maximum number of hash n-degree quads calls (%d) exceeded; the dataset may be poisoned", s.opts.maxCalls)
	}

	// Step 1: create Hn, relating hashes to related blank nodes.
	hn := make(map[string][]string)

	// Step 2: quads is the mention set of identifier.
	for _, idx := range s.bnodeQuads[identifier] {
		quad := s.quads[idx]
		for _, comp := range quadComponents(quad) {
			c := comp.term
			if !c.IsBlankNode() || c.Value == identifier {
				continue
			}
			hash, err := s.hashRelatedBlankNode(c.Value, quad, pathIssuer, comp.position)
			if err != nil {
				return "", nil, err
			}
			hn[hash] = append(hn[hash], c.Value)
		}
	}

	// Step 3: for each related hash, in code point order, explore the
	// gossip paths of its blank nodes and record the chosen path.
	var dataToHash strings.Builder
	for _, relatedHash := range sortedKeys(hn) {
		dataToHash.WriteString(relatedHash)

		bnodeList := hn[relatedHash]
		chosenPath := ""
		chosenIssuer := (*IdentifierIssuer)(nil)

		var permErr error
		permAbort := false
		permutations(bnodeList, func(p []string) bool {
			if permAbort {
				return false
			}
			// Poison guard: each permutation costs at least O(len(p)).
			s.hnCalls++
			if s.hnCalls > s.opts.maxCalls {
				permErr = fmt.Errorf("maximum number of hash n-degree quads permutations (%d) exceeded; the dataset may be poisoned", s.opts.maxCalls)
				return false
			}

			issuerCopy := pathIssuer.clone()
			var path strings.Builder
			var recursionList []string

			// Step 3.2.4: for each related in the permutation.
			skip := false
			for _, related := range p {
				if id, ok := s.issuer.getId(related); ok {
					path.WriteString("_:")
					path.WriteString(id)
				} else {
					if !issuerCopy.has(related) {
						recursionList = append(recursionList, related)
					}
					path.WriteString("_:")
					path.WriteString(issuerCopy.issue(related))
				}
				if chosenPath != "" && len(path.String()) >= len(chosenPath) && path.String() > chosenPath {
					skip = true
					break
				}
			}

			// Step 3.2.5: recurse over blank nodes that were issued
			// temporary identifiers.
			if !skip {
				for _, related := range recursionList {
					hnHash, iss, err := s.hashNDegreeQuads(related, issuerCopy)
					if err != nil {
						permErr = err
						permAbort = true
						return false
					}
					issuerCopy = iss
					path.WriteString("_:")
					path.WriteString(issuerCopy.issue(related))
					path.WriteByte('<')
					path.WriteString(hnHash)
					path.WriteByte('>')
					if chosenPath != "" && len(path.String()) >= len(chosenPath) && path.String() > chosenPath {
						skip = true
						break
					}
				}
			}

			// Step 3.2.6: track the chosen path.
			if !skip && (chosenPath == "" || path.String() < chosenPath) {
				chosenPath = path.String()
				chosenIssuer = issuerCopy
			}
			return true
		})
		if permErr != nil {
			return "", nil, permErr
		}

		dataToHash.WriteString(chosenPath)
		pathIssuer = chosenIssuer
	}

	return s.opts.hashFn(dataToHash.String()), pathIssuer, nil
}

// quadComponents returns the subject, object and graph name of a quad with
// their positions ("s", "o", "g").
func quadComponents(q Quad) []struct {
	position string
	term     Term
} {
	components := []struct {
		position string
		term     Term
	}{{"s", q.Subject}, {"o", q.Object}}
	if q.Graph != nil {
		components = append(components, struct {
			position string
			term     Term
		}{"g", *q.Graph})
	}
	return components
}

// permutations lazily enumerates every permutation of list in lexicographic
// order of indices, calling fn for each. fn returns false to abort
// enumeration.
func permutations(list []string, fn func(p []string) bool) {
	n := len(list)
	if n == 0 {
		fn(nil)
		return
	}
	// Work on a copy: Heap's algorithm reorders the slice in place and the
	// caller's list (an Hn entry) must stay untouched.
	list = append([]string(nil), list...)
	c := make([]int, n)
	perm := append([]string(nil), list...)
	if !fn(perm) {
		return
	}
	// Iterative Heap's algorithm.
	for i := 0; i < n; {
		if c[i] < i {
			if i%2 == 0 {
				list[0], list[i] = list[i], list[0]
			} else {
				list[c[i]], list[i] = list[i], list[c[i]]
			}
			perm = append(perm[:0], list...)
			if !fn(perm) {
				return
			}
			c[i]++
			i = 0
		} else {
			c[i] = 0
			i++
		}
	}
}
