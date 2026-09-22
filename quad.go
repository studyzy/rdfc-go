// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package rdfc implements the W3C RDF Dataset Canonicalization Algorithm
// version 1.0 (RDFC-1.0) as defined by https://www.w3.org/TR/rdf-canon/.
//
// The package is dependency free. Input and output use the canonical
// N-Quads form defined by the specification (appendix A of RDFC-1.0).
package rdfc

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TermKind identifies the kind of an RDF term.
type TermKind uint8

// Kinds of RDF terms.
const (
	KindIRI TermKind = iota
	KindBlankNode
	KindLiteral
)

// XSDString is the datatype IRI of simple literals in RDF 1.1.
const XSDString = "http://www.w3.org/2001/XMLSchema#string"

// Term is an RDF term: an IRI, a blank node, or a literal.
type Term struct {
	Kind     TermKind
	Value    string // IRI without <>, blank node label without _:, or the decoded lexical form of a literal
	LangTag  string // language tag for literals (without @), empty otherwise
	Datatype string // datatype IRI for typed literals (without <>), empty for simple literals
}

// IRI returns an IRI term.
func IRI(value string) Term { return Term{Kind: KindIRI, Value: value} }

// BlankNode returns a blank node term with the given label.
func BlankNode(label string) Term { return Term{Kind: KindBlankNode, Value: label} }

// Literal returns a literal term. lang and datatype may be empty. A datatype
// of XSDString is treated as a simple literal, per RDF 1.1.
func Literal(value, langTag, datatype string) Term {
	if datatype == XSDString {
		datatype = ""
	}
	return Term{Kind: KindLiteral, Value: value, LangTag: langTag, Datatype: datatype}
}

// IsBlankNode reports whether the term is a blank node.
func (t Term) IsBlankNode() bool { return t.Kind == KindBlankNode }

// String returns the canonical N-Quads serialization of the term.
func (t Term) String() string {
	var sb strings.Builder
	writeTerm(&sb, t)
	return sb.String()
}

// Quad is an RDF quad: a generalized triple with an optional graph name.
// A nil Graph means the default graph.
type Quad struct {
	Subject   Term
	Predicate Term
	Object    Term
	Graph     *Term
}

// String returns the canonical N-Quads serialization of the quad, including
// the trailing " ." and line feed.
func (q Quad) String() string {
	var sb strings.Builder
	writeQuad(&sb, q)
	sb.WriteString(" .\n")
	return sb.String()
}

// writeQuad serializes the quad body (without the terminator).
func writeQuad(sb *strings.Builder, q Quad) {
	writeTerm(sb, q.Subject)
	sb.WriteByte(' ')
	writeTerm(sb, q.Predicate)
	sb.WriteByte(' ')
	writeTerm(sb, q.Object)
	if q.Graph != nil {
		sb.WriteByte(' ')
		writeTerm(sb, *q.Graph)
	}
}

func writeTerm(sb *strings.Builder, t Term) {
	switch t.Kind {
	case KindIRI:
		sb.WriteByte('<')
		escapeIRI(sb, t.Value)
		sb.WriteByte('>')
	case KindBlankNode:
		sb.WriteString("_:")
		sb.WriteString(t.Value)
	case KindLiteral:
		sb.WriteByte('"')
		escapeLiteral(sb, t.Value)
		sb.WriteByte('"')
		if t.LangTag != "" {
			sb.WriteByte('@')
			sb.WriteString(t.LangTag)
		} else if t.Datatype != "" {
			sb.WriteString("^^<")
			escapeIRI(sb, t.Datatype)
			sb.WriteByte('>')
		}
	}
}

// escapeIRI escapes an IRI reference per the N-Quads IRIREF grammar:
// every code point excluded by the grammar (U+0000-U+0020 and <>"{}|^\`)
// is written as UCHAR. All other code points are written natively.
func escapeIRI(sb *strings.Builder, s string) {
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			i++
			if c <= 0x20 || strings.ContainsRune("<>\"{}|^`\\", rune(c)) {
				writeUChar(sb, rune(c))
			} else {
				sb.WriteByte(c)
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8: only possible if the string holds a lone
			// surrogate stored using the surrogate UTF-8 encoding
			// (see decodeSurrogateBytes in nquads.go).
			if sr, ok := decodeSurrogateBytes(s[i:]); ok {
				writeUChar(sb, sr)
				i += 3
				continue
			}
			// Should not happen; emit the replacement character.
			sb.WriteRune(utf8.RuneError)
			i++
			continue
		}
		if r <= 0x20 || strings.ContainsRune("<>\"{}|^`\\", r) {
			writeUChar(sb, r)
		} else {
			sb.WriteString(s[i : i+size])
		}
		i += size
	}
}

// escapeLiteral escapes a literal value per the canonical N-Quads
// STRING_LITERAL_QUOTE rules (RDFC-1.0 appendix A):
//   - ECHAR: \b \t \n \f \r \" \\
//   - UCHAR (lowercase "u", four uppercase HEX digits): U+0000-U+0007,
//     U+000B, U+000E-U+001F, U+007F, and code points that are not
//     characters per the XML 1.1 Char production (this includes the
//     surrogate range).
//   - everything else (space, quotes brackets, non-ASCII) is written natively.
func escapeLiteral(sb *strings.Builder, s string) {
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			i++
			switch {
			case c == 0x08:
				sb.WriteString(`\b`)
			case c == 0x09:
				sb.WriteString(`\t`)
			case c == 0x0A:
				sb.WriteString(`\n`)
			case c == 0x0C:
				sb.WriteString(`\f`)
			case c == 0x0D:
				sb.WriteString(`\r`)
			case c == '"':
				sb.WriteString(`\"`)
			case c == '\\':
				sb.WriteString(`\\`)
			case c <= 0x07 || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F:
				writeUChar(sb, rune(c))
			default:
				sb.WriteByte(c)
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8: only possible if the string holds a lone
			// surrogate stored using the surrogate UTF-8 encoding
			// (see decodeEscape in nquads.go).
			if sr, ok := decodeSurrogateBytes(s[i:]); ok {
				writeUChar(sb, sr)
				i += 3
				continue
			}
			// Should not happen; emit the replacement character.
			sb.WriteRune(utf8.RuneError)
			i++
			continue
		}
		if r <= 0x1F || r == 0x7F || !isXML11Char(r) {
			writeUChar(sb, r)
		} else {
			sb.WriteString(s[i : i+size])
		}
		i += size
	}
}

// writeUChar writes a UCHAR escape: lowercase "u" followed by four
// uppercase HEX digits. Code points above U+FFFF use "U" and eight digits.
func writeUChar(sb *strings.Builder, r rune) {
	if r > 0xFFFF {
		fmt.Fprintf(sb, `\U%08X`, r)
		return
	}
	fmt.Fprintf(sb, `\u%04X`, r)
}

// isXML11Char reports whether r matches the Char production of XML 1.1.
func isXML11Char(r rune) bool {
	return (r >= 0x1 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD) ||
		(r >= 0x10000 && r <= 0x10FFFF)
}
