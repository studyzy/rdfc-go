// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package rdfc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// manifestEntry mirrors one test entry of the official W3C RDFC-1.0
// test manifest (testdata/manifest.jsonld).
type manifestEntry struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Action        string `json:"action"`
	Result        string `json:"result"`
	HashAlgorithm string `json:"hashAlgorithm"`
}

type manifest struct {
	Entries []manifestEntry `json:"entries"`
}

func loadManifest(t *testing.T) []manifestEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "manifest.jsonld"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m struct {
		manifest
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return m.Entries
}

func readTestFile(t *testing.T, ref string) []byte {
	t.Helper()
	// Manifest references are relative to the tests directory, e.g.
	// "rdfc10/test001-in.nq".
	data, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(ref)))
	if err != nil {
		t.Fatalf("read %s: %v", ref, err)
	}
	return data
}

// TestOfficialSuite runs every entry of the official W3C RDFC-1.0 test
// suite. Each subtest is named after the manifest entry id.
func TestOfficialSuite(t *testing.T) {
	entries := loadManifest(t)
	for _, entry := range entries {
		entry := entry
		// Normalize manifest hash algorithm names ("SHA384") to the
		// library's naming ("SHA-384"); entries default to SHA-256.
		var opts []Option
		if entry.HashAlgorithm != "" {
			ha := strings.ToUpper(strings.ReplaceAll(entry.HashAlgorithm, "-", ""))
			switch ha {
			case "SHA256":
				opts = append(opts, WithHashAlgorithm("SHA-256"))
			case "SHA384":
				opts = append(opts, WithHashAlgorithm("SHA-384"))
			default:
				t.Fatalf("unsupported hashAlgorithm %q in %s", entry.HashAlgorithm, entry.ID)
			}
		}
		switch entry.Type {
		case "rdfc:RDFC10EvalTest":
			t.Run(strings.TrimPrefix(entry.ID, "#"), func(t *testing.T) {
				got, err := Canonicalize(readTestFile(t, entry.Action), opts...)
				if err != nil {
					t.Fatalf("canonicalize: %v", err)
				}
				want := readTestFile(t, entry.Result)
				if string(got) != string(want) {
					t.Fatalf("canonical output mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
				}
			})
		case "rdfc:RDFC10MapTest":
			t.Run(strings.TrimPrefix(entry.ID, "#"), func(t *testing.T) {
				quads, err := ParseNQuads(readTestFile(t, entry.Action))
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
				result, err := CanonicalizeDataset(quads, opts...)
				if err != nil {
					t.Fatalf("canonicalize: %v", err)
				}
				wantRaw := readTestFile(t, entry.Result)
				var want map[string]string
				if err := json.Unmarshal(wantRaw, &want); err != nil {
					t.Fatalf("parse expected map: %v", err)
				}
				if len(want) != len(result.IssuedIdentifiers) {
					t.Fatalf("identifier map size mismatch: want %d entries, got %d (%v)",
						len(want), len(result.IssuedIdentifiers), result.IssuedIdentifiers)
				}
				for in, expected := range want {
					got, ok := result.IssuedIdentifiers[in]
					if !ok {
						t.Fatalf("no canonical identifier issued for blank node %q", in)
					}
					if got != expected {
						t.Fatalf("blank node %q: want %q, got %q", in, expected, got)
					}
				}
			})
		case "rdfc:RDFC10NegativeEvalTest":
			t.Run(strings.TrimPrefix(entry.ID, "#"), func(t *testing.T) {
				_, err := Canonicalize(readTestFile(t, entry.Action))
				if err == nil {
					t.Fatalf("expected canonicalization to fail (dataset poisoning), but it succeeded")
				}
			})
		default:
			t.Errorf("unknown manifest entry type %q for %s", entry.Type, entry.ID)
		}
	}
}
