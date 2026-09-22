// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

// Result is the outcome of canonicalizing a dataset.
type Result struct {
	// CanonicalNQuads is the canonical N-Quads serialization of the
	// canonicalized dataset: quads with canonical blank node identifiers,
	// sorted in code point order, each terminated by LF.
	CanonicalNQuads []byte

	// IssuedIdentifiers maps input blank node identifiers to the
	// canonical identifiers issued by the algorithm (without the "_:"
	// prefix, e.g. "e0" -> "c14n0").
	IssuedIdentifiers map[string]string
}

type options struct {
	hashName string
	hashFn   func(string) string
	maxCalls int
}

// Option configures the canonicalization algorithm.
type Option func(*options)

// WithHashAlgorithm sets the hash algorithm used by the three hashing
// sub-algorithms. Supported values: "SHA-256" (default, required by
// RDFC-1.0) and "SHA-384".
func WithHashAlgorithm(name string) Option {
	return func(o *options) {
		o.hashName = name
	}
}

// WithMaxCalls bounds the combined number of Hash N-Degree Quads
// invocations and path permutations processed for one canonicalization run.
// It defends against dataset poisoning (spec section 5.2). When the limit
// is exceeded, Canonicalize returns an error. A value of 0 or negative
// keeps the default.
func WithMaxCalls(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxCalls = n
		}
	}
}

// defaultMaxCalls returns the poison-guard budget for a dataset with n
// blank nodes. It is generous enough for all official conformance tests
// while bounding pathological inputs such as cliques.
func defaultMaxCalls(n int) int {
	budget := 1000 * (n + 1)
	if budget < 10000 {
		budget = 10000
	}
	return budget
}

// Canonicalize parses an N-Quads document, applies the RDFC-1.0
// canonicalization algorithm and returns the canonical N-Quads
// serialization of the dataset.
func Canonicalize(nquads []byte, opts ...Option) ([]byte, error) {
	quads, err := ParseNQuads(nquads)
	if err != nil {
		return nil, err
	}
	result, err := CanonicalizeDataset(quads, opts...)
	if err != nil {
		return nil, err
	}
	return result.CanonicalNQuads, nil
}

// CanonicalizeDataset applies the RDFC-1.0 canonicalization algorithm to an
// in-memory dataset.
func CanonicalizeDataset(quads []Quad, opts ...Option) (*Result, error) {
	o := options{hashName: "SHA-256"}
	for _, apply := range opts {
		apply(&o)
	}
	hashFn, err := newHasher(o.hashName)
	if err != nil {
		return nil, err
	}
	o.hashFn = hashFn
	if o.maxCalls <= 0 {
		n := 0
		seen := make(map[string]struct{})
		for _, q := range quads {
			for _, c := range []Term{q.Subject, q.Object} {
				if c.IsBlankNode() {
					if _, ok := seen[c.Value]; !ok {
						seen[c.Value] = struct{}{}
						n++
					}
				}
			}
			if q.Graph != nil && q.Graph.IsBlankNode() {
				if _, ok := seen[q.Graph.Value]; !ok {
					seen[q.Graph.Value] = struct{}{}
					n++
				}
			}
		}
		o.maxCalls = defaultMaxCalls(n)
	}
	return canonicalizeDataset(quads, o)
}
