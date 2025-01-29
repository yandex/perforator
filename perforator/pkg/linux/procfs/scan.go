package procfs

import "strings"

// Manual for writing your own split function can be seen here:
// https://github.com/golang/go/blob/8f5c6904b616fd97dde4a0ba2f5c71114e588afd/src/bufio/scan.go#L67
func splitByNull(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := strings.IndexByte(string(data), '\000'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	// If at end of file and no \0 found, return the entire remaining data.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
