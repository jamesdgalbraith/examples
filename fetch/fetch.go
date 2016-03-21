// Copyright ©2013 The bíogo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Fetch is a simple illustration of using biogo.entrez to retrieve a large
// set of sequences to a file.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/biogo/ncbi"
	"github.com/biogo/ncbi/entrez"
)

const (
	db    = "protein"
	query = `"reverse transcriptase" or "transposon" or "repetitive element" or ` +
		`"RNA-directed DNA polymerase" or "pol protein" or "non-LTR retrotransposon" or ` +
		`"mobile element" or "retroelement" or "polyprotein" or "retrovirus" or ` +
		`("group-specific antigen" "gag") or "polymerase (pol)"`

	tool = "biogo.example"
)

var (
	clQuery = flag.String("query", query, "query specifies the search query for record retrieval.")
	rettype = flag.String("rettype", "fasta", "rettype specifies the format of the returned data.")
	retmax  = flag.Int("retmax", 500, "retmax specifies the number of records to be retrieved per request.")
	out     = flag.String("out", "", "out specifies destination of the returned data (default to stdout).")
	email   = flag.String("email", "", "email specifies the email address to be sent to the server (required).")
	retries = flag.Int("retry", 5, "retry specifies the number of attempts to retrieve the data.")
	help    = flag.Bool("help", false, "help prints this message.")
)

func main() {
	ncbi.SetTimeout(0)

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *email == "" {
		flag.Usage()
		os.Exit(1)
	}

	h := entrez.History{}
	s, err := entrez.DoSearch(db, *clQuery, nil, &h, tool, *email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Will retrieve %d records.\n", s.Count)

	var of *os.File
	if *out == "" {
		of = os.Stdout
	} else {
		of, err = os.Create(*out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer of.Close()
	}

	var (
		buf   = &bytes.Buffer{}
		p     = &entrez.Parameters{RetMax: *retmax, RetType: *rettype, RetMode: "text"}
		bn, n int64
	)
	for p.RetStart = 0; p.RetStart < s.Count; p.RetStart += p.RetMax {
		fmt.Fprintf(os.Stderr, "Attempting to retrieve %d records starting from %d with %d retries.\n", p.RetMax, p.RetStart, *retries)
		var t int
		for t = 0; t < *retries; t++ {
			buf.Reset()
			var (
				r   io.ReadCloser
				_bn int64
			)
			r, err = entrez.Fetch(db, p, tool, *email, &h)
			if err != nil {
				if r != nil {
					r.Close()
				}
				fmt.Fprintf(os.Stderr, "Failed to retrieve on attempt %d... error: %v retrying.\n", t, err)
				continue
			}
			_bn, err = io.Copy(buf, r)
			bn += _bn
			r.Close()
			if err == nil {
				break
			}
			fmt.Fprintf(os.Stderr, "Failed to buffer on attempt %d... error: %v retrying.\n", t, err)
		}
		if err != nil {
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Retrieved records with %d retries... writing out.\n", t)
		_n, err := io.Copy(of, buf)
		n += _n
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	if bn != n {
		fmt.Fprintf(os.Stderr, "Writethrough mismatch: %d != %d\n", bn, n)
	}
}
