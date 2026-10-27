// corpus-test tests and benchmarks a compressor against a corpus of files
// (such as the Silesia corpus). It compresses each file in the `corpus`
// directory, verifies that it decompresses correctly, and prints the
// compressed size, time to compress, etc.
//
// To change which compressor it uses, edit the source code.
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/andybalholm/brotli"
)

func main() {
	filesInCorpus, err := os.ReadDir("corpus")
	if err != nil {
		log.Fatal("Error listing files in corpus: ", err)
	}

	dest := new(bytes.Buffer)
	bw := brotli.NewWriterV2(dest, 0)

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)

	fmt.Fprintln(tw, "FILE\tORIGINAL\tCOMPRESSED\tRATIO\tTIME\tMB/s")

	var totalUncompressed, totalCompressed int
	var totalTime time.Duration

	for _, dirEntry := range filesInCorpus {
		filename := dirEntry.Name()
		if strings.HasPrefix(filename, ".") {
			continue
		}
		fmt.Println(filename)
		data, err := os.ReadFile(filepath.Join("corpus", filename))
		if err != nil {
			log.Fatalf("Error reading %s: %v", filename, err)
		}
		dest.Reset()
		bw.Reset(dest)
		start := time.Now()
		if _, err := bw.Write(data); err != nil {
			log.Fatalf("Error compressing %s: %v", filename, err)
		}
		if err := bw.Close(); err != nil {
			log.Fatalf("Error closing compressor for %s: %v", filename, err)
		}
		elapsed := time.Now().Sub(start)
		compressedSize := dest.Len()

		// check for round-trip integrity
		br := brotli.NewReader(dest)
		decompressed, err := io.ReadAll(br)
		if err != nil {
			log.Fatalf("Error compressing %s: %v", filename, err)
		}
		if !bytes.Equal(data, decompressed) {
			log.Fatalf("Decompressed data for %s doesn't match", filename)
		}

		fmt.Fprintf(tw, "%s\t%d\t%d\t%0.3f\t%v\t%0.2f\n",
			filename,
			len(data),
			compressedSize,
			float64(len(data))/float64(compressedSize),
			elapsed,
			float64(len(data))/float64(1<<20)/(float64(elapsed)/float64(time.Second)),
		)
		totalUncompressed += len(data)
		totalCompressed += compressedSize
		totalTime += elapsed
	}

	fmt.Fprintf(tw, "TOTAL\t%d\t%d\t%0.3f\t%v\t%0.2f\n",
		totalUncompressed,
		totalCompressed,
		float64(totalUncompressed)/float64(totalCompressed),
		totalTime,
		float64(totalUncompressed)/float64(1<<20)/(float64(totalTime)/float64(time.Second)),
	)

	tw.Flush()
}
