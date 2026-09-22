# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-09-22

### Added

- RDFC-1.0 canonicalization engine (`Canonicalize`, `CanonicalizeDataset`) implementing
  the full [W3C RDF Dataset Canonicalization 1.0](https://www.w3.org/TR/rdf-canon/)
  algorithm: Hash First Degree Quads, Hash Related Blank Node, Hash N-Degree Quads
  with permutation pruning, and canonical N-Quads serialization.
- Dependency-free N-Quads parser and canonical serializer, including UCHAR/ECHAR
  escaping per canonical form (appendix A) and lone surrogate handling.
- SHA-256 (default) and SHA-384 hash algorithms via `WithHashAlgorithm`.
- Dataset poisoning guard via `WithMaxCalls`, raising an error instead of running
  unbounded, per spec section 7.1.
- `Result.IssuedIdentifiers` mapping input blank node identifiers to canonical ones.
- `rdfc-go` command line tool with `-hash` and `-map` flags.
- Official W3C conformance suite embedded in `testdata/` (86/86 passing:
  64 EvalTest, 21 MapTest, 1 NegativeEvalTest).

[Unreleased]: https://github.com/studyzy/rdfc-go/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/studyzy/rdfc-go/releases/tag/v1.0.0
