// Copyright 2026 rdfc-go authors
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Command rdfc-go canonicalizes RDF datasets with the W3C RDFC-1.0
// algorithm (https://www.w3.org/TR/rdf-canon/).
//
// Usage:
//
//	rdfc-go [-hash SHA-256|SHA-384] [-map] [file ...]
//
// With no file operands, or when a file is "-", standard input is read.
// Input must be N-Quads; output is the canonical N-Quads form written to
// standard output. With -map, the blank node identifier mapping is written
// to standard error as JSON instead.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/studyzy/rdfc-go"
)

func main() {
	hash := flag.String("hash", "SHA-256", "hash algorithm: SHA-256 or SHA-384")
	showMap := flag.Bool("map", false, "print the blank node identifier mapping (JSON) to stderr instead of the canonical form")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-hash SHA-256|SHA-384] [-map] [file ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var opts []rdfc.Option
	if *hash != "" {
		opts = append(opts, rdfc.WithHashAlgorithm(*hash))
	}

	files := flag.Args()
	if len(files) == 0 {
		files = []string{"-"}
	}
	status := 0
	for _, name := range files {
		if err := run(name, *showMap, opts); err != nil {
			fmt.Fprintf(os.Stderr, "rdfc-go: %v\n", err)
			status = 1
		}
	}
	os.Exit(status)
}

func run(name string, showMap bool, opts []rdfc.Option) error {
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(name)
	}
	if err != nil {
		return err
	}

	quads, err := rdfc.ParseNQuads(data)
	if err != nil {
		return err
	}
	result, err := rdfc.CanonicalizeDataset(quads, opts...)
	if err != nil {
		return err
	}
	if showMap {
		enc, err := json.MarshalIndent(result.IssuedIdentifiers, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, string(enc))
		return nil
	}
	_, err = os.Stdout.Write(result.CanonicalNQuads)
	return err
}
