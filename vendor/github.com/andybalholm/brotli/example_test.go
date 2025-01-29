// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package brotli

import (
	"bytes"
	"io"
	"log"
	"os"
)

func ExampleWriter_Reset() {
	proverbs := []string{
		"Don't communicate by sharing memory, share memory by communicating.\n",
		"Concurrency is not parallelism.\n",
		"The bigger the interface, the weaker the abstraction.\n",
		"Documentation is for users.\n",
	}

	var b bytes.Buffer

	bw := NewWriter(nil)
	br := NewReader(nil)

	for _, s := range proverbs {
		b.Reset()

		// Reset the compressor and encode from some input stream.
		bw.Reset(&b)
		if _, err := io.WriteString(bw, s); err != nil {
			log.Fatal(err)
		}
		if err := bw.Close(); err != nil {
			log.Fatal(err)
		}

		// Reset the decompressor and decode to some output stream.
		if err := br.Reset(&b); err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(os.Stdout, br); err != nil {
			log.Fatal(err)
		}
	}

	// Output:
	// Don't communicate by sharing memory, share memory by communicating.
	// Concurrency is not parallelism.
	// The bigger the interface, the weaker the abstraction.
	// Documentation is for users.
}
