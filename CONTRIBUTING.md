# Contributing to rdfc-go

Thank you for considering a contribution! This document describes how to set up
the project and submit changes.

## Development setup

Requirements: Go 1.24+ (see `go.mod`).

```sh
git clone https://github.com/studyzy/rdfc-go.git
cd rdfc-go
make test   # run all tests, including the official W3C suite
make lint   # gofmt + go vet + golangci-lint
make race   # tests with the race detector
```

## Project conventions

- **Zero dependencies**: the library uses only the Go standard library. Do not
  introduce new module requirements without prior discussion.
- **Conformance first**: canonicalization output is fixed by the
  [RDFC-1.0 specification](https://www.w3.org/TR/rdf-canon/). Changes affecting
  output bytes or error behavior must keep the official suite in `testdata/`
  passing (86/86) and cite the relevant spec section in the PR description.
- Keep diffs minimal and focused; match the style of the surrounding code.

## Submitting changes

1. Fork the repository and create a topic branch.
2. Make your change with tests. New sub-algorithm behavior should come with a
   case from the official test suite or a hand-verified example from the spec.
3. Run `make lint` and `make test`; CI must pass.
4. Open a pull request describing the motivation and the verification you did.

## Reporting bugs

Please use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and
include a minimal N-Quads input together with the expected canonical output.
For security issues, see [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE) that covers this project.
