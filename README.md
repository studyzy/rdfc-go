# rdfc-go

[![CI](https://github.com/studyzy/rdfc-go/actions/workflows/ci.yml/badge.svg)](https://github.com/studyzy/rdfc-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/studyzy/rdfc-go.svg)](https://pkg.go.dev/github.com/studyzy/rdfc-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/studyzy/rdfc-go)](https://goreportcard.com/report/github.com/studyzy/rdfc-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go implementation of the [W3C RDF Dataset Canonicalization Algorithm 1.0
(RDFC-1.0)](https://www.w3.org/TR/rdf-canon/).

- **Zero dependencies** — only the Go standard library.
- **Conformance**: passes the complete official W3C test suite
  (86/86: 64 EvalTest + 21 MapTest + 1 NegativeEvalTest), see
  `testdata/manifest.jsonld`.
- Supports SHA-256 (required by RDFC-1.0) and SHA-384.

## Install

```
go get github.com/studyzy/rdfc-go
```

## Usage

### Library

```go
package main

import (
	"fmt"

	"github.com/studyzy/rdfc-go"
)

func main() {
	input := []byte(`
_:e0 <http://example.org/vocab#next> _:e2 .
_:e1 <http://example.org/vocab#next> _:e2 .
`)

	// N-Quads in, canonical N-Quads out.
	canonical, err := rdfc.Canonicalize(input)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(canonical))

	// Or work with an in-memory dataset and inspect the issued
	// blank node identifiers.
	quads, _ := rdfc.ParseNQuads(input)
	result, _ := rdfc.CanonicalizeDataset(quads)
	for in, out := range result.IssuedIdentifiers {
		fmt.Printf("_:%s -> _:%s\n", in, out)
	}
}
```

Options:

```go
rdfc.Canonicalize(input,
	rdfc.WithHashAlgorithm("SHA-384"), // default: SHA-256
	rdfc.WithMaxCalls(50000),          // dataset poisoning guard (default: 1000*(#blank nodes+1))
)
```

### Command line

```
go install github.com/studyzy/rdfc-go/cmd/rdfc-go@latest

rdfc-go dataset.nq > canonical.nq        # canonical N-Quads to stdout
cat dataset.nq | rdfc-go -map            # blank node mapping as JSON (stderr)
rdfc-go -hash SHA-384 dataset.nq         # alternative hash
```

## What is canonicalization for?

Two RDF datasets that differ only in the *labels* of their blank nodes are
indistinguishable in RDF semantics, yet serialize to different bytes. RDFC-1.0
assigns deterministic identifiers (`_:c14n0`, `_:c14n1`, ...) to blank nodes so
that two datasets produce byte-identical output **if and only if** they are
isomorphic. This is the foundation for signing and hashing RDF data, e.g. in
[Verifiable Credentials](https://www.w3.org/TR/vc-data-integrity/) cryptosuites
using the `rdfc-*` transformation.

The hard part of the problem is graph isomorphism: blank nodes that share the
same neighborhood must be told apart by exploring "gossip paths" through the
dataset (the recursive Hash N-Degree Quads algorithm). Adversarial inputs
(e.g. cliques of blank nodes) can make this exploration explode; the algorithm
therefore allows implementations to abort. `rdfc-go` bounds the work with a
call budget (`WithMaxCalls`) and returns an error — matching the
`RDFC10NegativeEvalTest` conformance behavior.

## API overview

| Symbol | Purpose |
|---|---|
| `Canonicalize(nquads []byte, opts ...Option) ([]byte, error)` | N-Quads document in, canonical N-Quads document out |
| `CanonicalizeDataset(quads []Quad, opts ...Option) (*Result, error)` | dataset-level API, exposes issued identifier map |
| `ParseNQuads(data []byte) ([]Quad, error)` | N-Quads parser |
| `Result.CanonicalNQuads` / `Result.IssuedIdentifiers` | canonical bytes / input→canonical blank node map |
| `Quad`, `Term`, `IRI()`, `BlankNode()`, `Literal()` | data model |
| `WithHashAlgorithm`, `WithMaxCalls` | options |

Sorting everywhere follows Unicode code point order (native Go string
comparison). Canonical N-Quads serialization follows appendix A of the
specification: single-space layout, ECHAR for `\b \t \n \f \r \" \\`,
lowercase-`u`/uppercase-HEX UCHAR for other control characters, `xsd:string`
datatypes omitted, lone surrogates preserved as UCHAR escapes.

## Testing

The test suite embeds the official W3C RDFC-1.0 test data (`testdata/`,
see `testdata/LICENCE.md`) and runs it via table-driven subtests:

```
make test      # all tests (library + official suite)
make lint      # gofmt, go vet, golangci-lint
make race      # tests with the race detector
make cover     # coverage report
```

Continuous integration runs the same checks on GitHub Actions for every push
and pull request (see `.github/workflows/ci.yml`). Contributions are welcome —
please read [CONTRIBUTING.md](CONTRIBUTING.md). For security issues, see
[SECURITY.md](SECURITY.md). Notable changes are tracked in
[CHANGELOG.md](CHANGELOG.md).

## License

MIT. The W3C test data in `testdata/` is distributed under the
W3C Software and Document License (`testdata/LICENCE.md`).

## References

- [RDF Dataset Canonicalization 1.0](https://www.w3.org/TR/rdf-canon/) — W3C Recommendation
- [titanium-rdf-canon](https://github.com/filip26/titanium-rdf-canon) (Java) and
  [rdf-canonize](https://github.com/digitalbazaar/rdf-canonize) (JavaScript) —
  reference implementations consulted during development
- [RDF 1.1 N-Quads](https://www.w3.org/TR/n-quads/)
