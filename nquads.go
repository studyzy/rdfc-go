// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ParseNQuads parses an N-Quads document into a slice of quads.
// Blank node identifiers are retained as-is; duplicate statements are not
// deduplicated.
func ParseNQuads(data []byte) ([]Quad, error) {
	p := &nqParser{data: data}
	return p.parseDocument()
}

type nqParser struct {
	data []byte
	pos  int
}

func (p *nqParser) parseDocument() ([]Quad, error) {
	var quads []Quad
	for {
		p.skipSpace()
		if p.eof() {
			return quads, nil
		}
		q, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		quads = append(quads, q)
	}
}

func (p *nqParser) skipSpace() {
	for !p.eof() {
		switch c := p.data[p.pos]; c {
		case ' ', '\t', '\n', '\r':
			p.pos++
		case '#':
			for !p.eof() && p.data[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

func (p *nqParser) eof() bool { return p.pos >= len(p.data) }

func (p *nqParser) peek() (byte, error) {
	if p.eof() {
		return 0, fmt.Errorf("unexpected end of input")
	}
	return p.data[p.pos], nil
}

func (p *nqParser) parseStatement() (Quad, error) {
	subject, err := p.parseTerm()
	if err != nil {
		return Quad{}, fmt.Errorf("subject: %w", err)
	}
	if subject.Kind == KindLiteral {
		return Quad{}, fmt.Errorf("subject cannot be a literal")
	}
	p.skipSpace()
	predicate, err := p.parseTerm()
	if err != nil {
		return Quad{}, fmt.Errorf("predicate: %w", err)
	}
	if predicate.Kind != KindIRI {
		return Quad{}, fmt.Errorf("predicate must be an IRI")
	}
	p.skipSpace()
	object, err := p.parseTerm()
	if err != nil {
		return Quad{}, fmt.Errorf("object: %w", err)
	}
	q := Quad{Subject: subject, Predicate: predicate, Object: object}

	p.skipSpace()
	if c, _ := p.peek(); c == '<' || (c == '_' && p.hasBNodePrefix()) {
		graph, err := p.parseTerm()
		if err != nil {
			return Quad{}, fmt.Errorf("graph: %w", err)
		}
		if graph.Kind == KindLiteral {
			return Quad{}, fmt.Errorf("graph name cannot be a literal")
		}
		q.Graph = &graph
		p.skipSpace()
	}
	if err := p.expectByte('.'); err != nil {
		return Quad{}, err
	}
	return q, nil
}

func (p *nqParser) expectByte(c byte) error {
	got, err := p.peek()
	if err != nil {
		return err
	}
	if got != c {
		return fmt.Errorf("expected %q but got %q at offset %d", c, got, p.pos)
	}
	p.pos++
	return nil
}

func (p *nqParser) hasBNodePrefix() bool {
	return p.pos+1 < len(p.data) && p.data[p.pos+1] == ':'
}

func (p *nqParser) parseTerm() (Term, error) {
	c, err := p.peek()
	if err != nil {
		return Term{}, err
	}
	switch c {
	case '<':
		iri, err := p.parseIRIValue()
		if err != nil {
			return Term{}, err
		}
		return Term{Kind: KindIRI, Value: iri}, nil
	case '_':
		label, err := p.parseBNodeValue()
		if err != nil {
			return Term{}, err
		}
		return Term{Kind: KindBlankNode, Value: label}, nil
	case '"':
		value, lang, dt, err := p.parseLiteralParts()
		if err != nil {
			return Term{}, err
		}
		return Literal(value, lang, dt), nil
	default:
		return Term{}, fmt.Errorf("unexpected character %q at offset %d", c, p.pos)
	}
}

func (p *nqParser) parseIRIValue() (string, error) {
	if err := p.expectByte('<'); err != nil {
		return "", err
	}
	var sb strings.Builder
	for {
		if p.eof() {
			return "", fmt.Errorf("unterminated IRI")
		}
		c := p.data[p.pos]
		if c == '>' {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' {
			p.pos++ // consume the backslash; parseUChar reads what follows
			r, err := p.parseUChar()
			if err != nil {
				return "", err
			}
			writeDecodedRune(&sb, r)
			continue
		}
		if c < 0x20 || c == '"' || strings.IndexByte("<>{}|^`", c) >= 0 {
			// '"' and <>{}|^` are excluded by the IRIREF grammar even raw.
			if strings.IndexByte("<>{}|^`\\", c) >= 0 || c == '"' {
				return "", fmt.Errorf("invalid character %q in IRI at offset %d", c, p.pos)
			}
		}
		sb.WriteByte(c)
		p.pos++
	}
}

func (p *nqParser) parseBNodeValue() (string, error) {
	if err := p.expectByte('_'); err != nil {
		return "", err
	}
	if err := p.expectByte(':'); err != nil {
		return "", err
	}
	var sb strings.Builder
	for !p.eof() {
		c := p.data[p.pos]
		if isBNodeByte(c) {
			sb.WriteByte(c)
			p.pos++
			continue
		}
		if c >= 0x80 {
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r != utf8.RuneError || size > 1 {
				sb.WriteString(string(r))
				p.pos += size
				continue
			}
		}
		if c == '.' {
			// A dot is part of the label only if followed by another
			// label character (labels must not end with a dot).
			if p.pos+1 < len(p.data) && isBNodeByte(p.data[p.pos+1]) {
				sb.WriteByte(c)
				p.pos++
				continue
			}
		}
		break
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("empty blank node label at offset %d", p.pos)
	}
	return sb.String(), nil
}

func isBNodeByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

func (p *nqParser) parseLiteralParts() (value, langTag, datatype string, err error) {
	if err = p.expectByte('"'); err != nil {
		return "", "", "", err
	}
	var sb strings.Builder
	for {
		if p.eof() {
			return "", "", "", fmt.Errorf("unterminated literal")
		}
		c := p.data[p.pos]
		switch c {
		case '"':
			p.pos++
			return p.parseLiteralSuffix(sb.String())
		case '\\':
			p.pos++
			r, err := p.parseEscape()
			if err != nil {
				return "", "", "", err
			}
			writeDecodedRune(&sb, r)
		case '\n', '\r':
			return "", "", "", fmt.Errorf("raw line break inside literal at offset %d", p.pos)
		default:
			sb.WriteByte(c)
			p.pos++
		}
	}
}

// parseLiteralSuffix parses the optional @lang or ^^<datatype> part.
func (p *nqParser) parseLiteralSuffix(value string) (string, string, string, error) {
	if p.eof() {
		return value, "", "", nil
	}
	switch p.data[p.pos] {
	case '@':
		p.pos++
		start := p.pos
		for !p.eof() && (isAlphaNumByte(p.data[p.pos]) || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.pos == start {
			return "", "", "", fmt.Errorf("empty language tag at offset %d", start)
		}
		return value, string(p.data[start:p.pos]), "", nil
	case '^':
		p.pos++
		if err := p.expectByte('^'); err != nil {
			return "", "", "", err
		}
		iri, err := p.parseIRIValue()
		if err != nil {
			return "", "", "", fmt.Errorf("datatype: %w", err)
		}
		return value, "", iri, nil
	}
	return value, "", "", nil
}

func isAlphaNumByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// parseEscape parses a string escape after the backslash has been consumed.
func (p *nqParser) parseEscape() (rune, error) {
	if p.eof() {
		return 0, fmt.Errorf("unterminated escape sequence")
	}
	c := p.data[p.pos]
	p.pos++
	switch c {
	case 't':
		return '\t', nil
	case 'b':
		return '\b', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 'f':
		return '\f', nil
	case '"':
		return '"', nil
	case '\'':
		return '\'', nil
	case '\\':
		return '\\', nil
	case 'u', 'U':
		width := 4
		if c == 'U' {
			width = 8
		}
		r, err := p.parseHex(width)
		if err != nil {
			return 0, err
		}
		// Combine a high surrogate escape followed by a low surrogate
		// escape into a single code point. A lone surrogate is returned
		// as-is and preserved through the surrogate UTF-8 encoding.
		if width == 4 && r >= 0xD800 && r <= 0xDBFF &&
			p.pos+6 <= len(p.data) && p.data[p.pos] == '\\' && p.data[p.pos+1] == 'u' {
			save := p.pos
			p.pos += 2
			lo, err := p.parseHex(4)
			if err == nil && lo >= 0xDC00 && lo <= 0xDFFF {
				return utf16.DecodeRune(r, lo), nil
			}
			p.pos = save
		}
		return r, nil
	default:
		return 0, fmt.Errorf("invalid escape sequence \\%c", c)
	}
}

// parseUChar parses a \uXXXX or \UXXXXXXXX escape inside an IRI.
func (p *nqParser) parseUChar() (rune, error) {
	if p.eof() {
		return 0, fmt.Errorf("unterminated escape sequence")
	}
	c := p.data[p.pos]
	p.pos++
	if c != 'u' && c != 'U' {
		return 0, fmt.Errorf("invalid IRI escape \\%c", c)
	}
	width := 4
	if c == 'U' {
		width = 8
	}
	return p.parseHex(width)
}

// parseHex parses width hexadecimal digits (case-insensitive) and returns
// the code point.
func (p *nqParser) parseHex(width int) (rune, error) {
	if p.pos+width > len(p.data) {
		return 0, fmt.Errorf("truncated unicode escape")
	}
	var v rune
	for i := 0; i < width; i++ {
		c := p.data[p.pos+i]
		var d rune
		switch {
		case c >= '0' && c <= '9':
			d = rune(c - '0')
		case c >= 'a' && c <= 'f':
			d = rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = rune(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hexadecimal digit %q in unicode escape", c)
		}
		v = v<<4 | d
	}
	p.pos += width
	return v, nil
}

// writeDecodedRune appends r to sb. Lone surrogates (which are legal in
// N-Quads via UCHAR escapes but cannot be encoded in valid UTF-8) are stored
// using the surrogate UTF-8 byte encoding so they can be re-emitted verbatim
// by the canonical serializer.
func writeDecodedRune(sb *strings.Builder, r rune) {
	if r >= 0xD800 && r <= 0xDFFF {
		encodeSurrogateRune(sb, r)
		return
	}
	sb.WriteRune(r)
}

func encodeSurrogateRune(sb *strings.Builder, r rune) {
	sb.WriteByte(0xE0 | byte(r>>12))
	sb.WriteByte(0x80 | byte((r>>6)&0x3F))
	sb.WriteByte(0x80 | byte(r&0x3F))
}

// decodeSurrogateBytes reverses encodeSurrogateRune for the byte sequence
// starting at the beginning of s.
func decodeSurrogateBytes(s string) (rune, bool) {
	if len(s) < 3 || s[0] != 0xED || s[1] < 0xA0 || s[1] > 0xBF || s[2] < 0x80 || s[2] > 0xBF {
		return 0, false
	}
	r := 0xD000 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F)
	return r, true
}
