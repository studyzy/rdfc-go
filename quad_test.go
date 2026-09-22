// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import "testing"

func TestParseAndSerializeRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // canonical N-Quads of the single statement ("" = parse error expected)
	}{
		{"simple", `<http://a/s> <http://a/p> "v" .`, `<http://a/s> <http://a/p> "v" .`},
		{"bnode", `_:b1 <http://a/p> _:b2 .`, `_:b1 <http://a/p> _:b2 .`},
		{"graph", `_:s <http://a/p> "o" <http://a/g> .`, `_:s <http://a/p> "o" <http://a/g> .`},
		{"langtag", `<s> <p> "hi"@en .`, `<s> <p> "hi"@en .`},
		{"extra whitespace", "<s>   <p>\t<o>\n.", `<s> <p> <o> .`},
		{"comment", "# comment\n<s> <p> <o> . # trailing\n", `<s> <p> <o> .`},
		{"IRI escape decode", `<s\u0065> <p> <o> .`, `<se> <p> <o> .`},
		{"IRI surrogate escape", `<s\U0001F303> <p> <o> .`, "<s\U0001F303> <p> <o> ."},
		{"IRI lone surrogate escape kept", `<s\uD800> <p> <o> .`, `<s\uD800> <p> <o> .`},
		{"literal escapes", `<s> <p> "a\tb\nc\\d\"e\'f" .`, `<s> <p> "a\tb\nc\\d\"e'f" .`},
		{"literal UCHAR uppercase hex", `<s> <p> "\u000B" .`, `<s> <p> "\u000B" .`},
		{"literal UCHAR surrogate pair", `<s> <p> "\uD83C\uDF03" .`, "<s> <p> \"\U0001F303\" ."},
		{"literal lone surrogate kept", `<s> <p> "\uD800" .`, `<s> <p> "\uD800" .`},
		{"literal raw tab kept raw", "<s> <p> \"x\ty\" .", `<s> <p> "x\ty" .`},
		{"datatype", `<s> <p> "1"^^<http://www.w3.org/2001/XMLSchema#integer> .`, `<s> <p> "1"^^<http://www.w3.org/2001/XMLSchema#integer> .`},
		{"xsd:string omitted", `<s> <p> "v"^^<http://www.w3.org/2001/XMLSchema#string> .`, `<s> <p> "v" .`},
		{"empty literal", `<s> <p> "" .`, `<s> <p> "" .`},
		{"error: raw LF in literal", "<s> <p> \"a\nb\" .", ""},
		{"error: bad escape in IRI", `<s\x41> <p> <o> .`, ""},
		{"error: missing dot", `<s> <p> <o>`, ""},
		{"error: literal subject", `"s" <p> <o> .`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quads, err := ParseNQuads([]byte(tc.in))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected parse error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(quads) != 1 {
				t.Fatalf("want 1 quad, got %d", len(quads))
			}
			if got := quads[0].String(); got != tc.want+"\n" {
				t.Fatalf("round trip mismatch:\n want %q\n got  %q", tc.want+"\n", got)
			}
		})
	}
}

func TestDuplicateQuadsDeduplicated(t *testing.T) {
	in := `<s> <p> <o> .
<s> <p> <o> .
<s> <p> _:a .
<s> <p> _:a .
`
	got, err := Canonicalize([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	want := `<s> <p> <o> .
<s> <p> _:c14n0 .
`
	if string(got) != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestIssuedIdentifiers(t *testing.T) {
	in := `<s> <p> _:name .`
	result, err := CanonicalizeDataset([]Quad{{Subject: IRI("s"), Predicate: IRI("p"), Object: BlankNode("name")}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IssuedIdentifiers["name"] != "c14n0" {
		t.Fatalf("want c14n0, got %q", result.IssuedIdentifiers["name"])
	}
	_ = in
}

func TestUnsupportedHashAlgorithm(t *testing.T) {
	_, err := Canonicalize([]byte(`<s> <p> <o> .`), WithHashAlgorithm("MD5"))
	if err == nil {
		t.Fatal("expected error for unsupported hash algorithm")
	}
}

func TestSHA384Diamond(t *testing.T) {
	// test075 of the official suite: the diamond graph with SHA-384.
	in := `<http://example.org/vocab#test> <http://example.org/vocab#A> _:e0 .
<http://example.org/vocab#test> <http://example.org/vocab#B> _:e1 .
_:e0 <http://example.org/vocab#next> _:e2 .
_:e1 <http://example.org/vocab#next> _:e2 .
`
	want := `<http://example.org/vocab#test> <http://example.org/vocab#A> _:c14n0 .
<http://example.org/vocab#test> <http://example.org/vocab#B> _:c14n2 .
_:c14n0 <http://example.org/vocab#next> _:c14n1 .
_:c14n2 <http://example.org/vocab#next> _:c14n1 .
`
	got, err := Canonicalize([]byte(in), WithHashAlgorithm("SHA-384"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestSymmetricDatasetStable(t *testing.T) {
	// test030-style: two isomorphic datasets must canonicalize identically.
	a := `_:a <http://example.org/p> _:b .
_:b <http://example.org/p> _:a .
`
	b := `_:x <http://example.org/p> _:y .
_:y <http://example.org/p> _:x .
`
	ga, err := Canonicalize([]byte(a))
	if err != nil {
		t.Fatal(err)
	}
	gb, err := Canonicalize([]byte(b))
	if err != nil {
		t.Fatal(err)
	}
	if string(ga) != string(gb) {
		t.Fatalf("isomorphic datasets differ:\n%s\nvs\n%s", ga, gb)
	}
}
