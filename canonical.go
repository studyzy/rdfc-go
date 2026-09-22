// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strings"
)

// canonicalizationState holds the state of the canonicalization algorithm
// (spec section 3.1).
type canonicalizationState struct {
	quads []Quad

	// blank node to quads map: blank node identifier -> indexes into quads
	bnodeQuads map[string][]int
	// hash to blank nodes map: first degree hash -> blank node identifiers
	hashToBnodes map[string][]string
	// canonical issuer, initialized with prefix "c14n"
	issuer *IdentifierIssuer

	opts options
	// hnCalls counts invocations of the Hash N-Degree Quads algorithm to
	// defend against dataset poisoning (spec section 5.2).
	hnCalls int
}

// canonicalizeDataset runs the RDFC-1.0 canonicalization algorithm on the
// input dataset and returns the issued identifiers map together with the
// canonical N-Quads serialization.
func canonicalizeDataset(quads []Quad, opts options) (*Result, error) {
	// An RDF dataset is a *set* of quads (RDF 1.1): duplicate statements
	// are dropped before canonicalization (see test076/test077 of the
	// official suite).
	if len(quads) > 1 {
		seen := make(map[string]struct{}, len(quads))
		uniq := make([]Quad, 0, len(quads))
		for _, q := range quads {
			key := q.String()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			uniq = append(uniq, q)
		}
		quads = uniq
	}

	s := &canonicalizationState{
		quads:        quads,
		bnodeQuads:   make(map[string][]int),
		hashToBnodes: make(map[string][]string),
		issuer:       newIdentifierIssuer("c14n"),
		opts:         opts,
	}

	// Step 1-2: initialization; build the blank node to quads map
	// (spec 4.4.3 steps 1-2).
	for idx, q := range quads {
		for _, c := range []Term{q.Subject, q.Object} {
			if c.IsBlankNode() {
				s.bnodeQuads[c.Value] = append(s.bnodeQuads[c.Value], idx)
			}
		}
		if q.Graph != nil && q.Graph.IsBlankNode() {
			s.bnodeQuads[q.Graph.Value] = append(s.bnodeQuads[q.Graph.Value], idx)
		}
	}

	// Step 3: compute first degree hashes for every blank node
	// (spec 4.4.3 step 3).
	for n := range s.bnodeQuads {
		hf, err := s.hashFirstDegreeQuads(n)
		if err != nil {
			return nil, err
		}
		s.hashToBnodes[hf] = append(s.hashToBnodes[hf], n)
	}

	// Step 4: issue canonical identifiers for blank nodes with a unique
	// first degree hash, in code point order of the hashes
	// (spec 4.4.3 step 4).
	for _, hash := range sortedKeys(s.hashToBnodes) {
		list := s.hashToBnodes[hash]
		if len(list) > 1 {
			continue
		}
		s.issuer.issue(list[0])
		delete(s.hashToBnodes, hash)
	}

	// Step 5: for each remaining (shared) hash, in code point order,
	// compute n-degree hashes and assign canonical identifiers
	// (spec 4.4.3 step 5).
	for _, hash := range sortedKeys(s.hashToBnodes) {
		list := s.hashToBnodes[hash]

		// Step 5.1-5.2: hash path list.
		type hashPath struct {
			hash   string
			issuer *IdentifierIssuer
		}
		var hashPathList []hashPath
		for _, n := range list {
			if s.issuer.has(n) {
				continue
			}
			tmp := newIdentifierIssuer("b")
			tmp.issue(n)
			hn, iss, err := s.hashNDegreeQuads(n, tmp)
			if err != nil {
				return nil, err
			}
			hashPathList = append(hashPathList, hashPath{hash: hn, issuer: iss})
		}
		// Step 5.3: process results in code point order of their hashes.
		sort.Slice(hashPathList, func(i, j int) bool {
			return hashPathList[i].hash < hashPathList[j].hash
		})
		for _, result := range hashPathList {
			for _, existing := range result.issuer.issuedOrder() {
				s.issuer.issue(existing)
			}
		}
	}

	// Step 6: serialize the canonicalized dataset (spec 4.4.3 step 6 and
	// section 4.8): every blank node replaced by its canonical identifier,
	// quads serialized in canonical n-quads form, sorted in code point
	// order and concatenated.
	lines := make([]string, 0, len(quads))
	for _, q := range quads {
		if q.Subject.IsBlankNode() {
			q.Subject.Value = s.issuer.issued[q.Subject.Value]
		}
		if q.Graph != nil && q.Graph.IsBlankNode() {
			g := *q.Graph
			g.Value = s.issuer.issued[g.Value]
			q.Graph = &g
		}
		if q.Object.IsBlankNode() {
			q.Object.Value = s.issuer.issued[q.Object.Value]
		}
		lines = append(lines, q.String())
	}
	sort.Strings(lines)

	// Every blank node must have received a canonical identifier; if any
	// did not, the algorithm would silently produce invalid output.
	for label := range s.bnodeQuads {
		if _, ok := s.issuer.issued[label]; !ok {
			return nil, fmt.Errorf("blank node %q was not issued a canonical identifier", label)
		}
	}

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(line)
	}
	return &Result{
		CanonicalNQuads:   []byte(sb.String()),
		IssuedIdentifiers: s.issuer.mapping(),
	}, nil
}

// sortedKeys returns the map keys sorted in Unicode code point order, which
// for UTF-8 strings is identical to the native byte-wise string order in Go.
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// newHasher returns a hash function name -> hex digest function for the
// configured algorithm.
func newHasher(name string) (func(string) string, error) {
	var newHash func() hash.Hash
	switch name {
	case "SHA-256":
		newHash = sha256.New
	case "SHA-384":
		newHash = sha512.New384
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q (supported: SHA-256, SHA-384)", name)
	}
	return func(s string) string {
		h := newHash()
		h.Write([]byte(s))
		return hex.EncodeToString(h.Sum(nil))
	}, nil
}
