package collapsed

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Sample struct {
	Stack []string
	Value int64
}

type Profile struct {
	Samples []Sample
}

func Decode(r io.Reader) (*Profile, error) {
	res := &Profile{
		Samples: make([]Sample, 0),
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		idx := strings.LastIndexByte(line, ' ')
		if idx == -1 {
			return nil, errors.New("collapsed: malformed input")
		}
		count, err := strconv.ParseInt(line[idx+1:], 0, 64)
		if err != nil {
			return nil, fmt.Errorf("collapsed: malformed input: %w", err)
		}
		res.Samples = append(res.Samples, Sample{
			Stack: strings.Split(line[:idx], ";"),
			Value: count,
		})
	}

	return res, nil
}

func Encode(profile *Profile, w io.Writer) error {
	return encodeImpl(profile, w, " ", "")
}

func EncodeDSV(profile *Profile, w io.Writer) error {
	return encodeImpl(profile, w, "\t", "v=")
}

func encodeImpl(profile *Profile, w io.Writer, sep, prefix string) error {
	for _, sample := range profile.Samples {
		stack := strings.Join(sample.Stack, ";")
		_, err := fmt.Fprintf(w, "%s%s%s%s%d\n", prefix, stack, sep, prefix, sample.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func Unmarshal(buf []byte) (*Profile, error) {
	return Decode(bytes.NewBuffer(buf))
}

func Marshal(profile *Profile) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := Encode(profile, buf)
	return buf.Bytes(), err
}
