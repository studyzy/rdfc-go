# Security Policy

## Supported versions

The latest released version and the `master` branch receive security fixes.

## Reporting a vulnerability

Please report vulnerabilities privately via
[GitHub Security Advisories](https://github.com/studyzy/rdfc-go/security/advisories/new)
rather than opening a public issue. Include a minimal reproducing input if
possible. You can expect an initial response within 7 days.

## Scope notes

- rdfc-go parses untrusted N-Quads documents; panics, unbounded memory growth
  or hangs on crafted input are in scope. The library already bounds
  canonicalization work with a call budget (see `WithMaxCalls`) to defend
  against [dataset poisoning](https://www.w3.org/TR/rdf-canon/#dfn-poison)
  denial of service, per the specification's security considerations.
- Output byte differences on non-adversarial input are conformance issues,
  not security issues — please file them as regular bugs.
