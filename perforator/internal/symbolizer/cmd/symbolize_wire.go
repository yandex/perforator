package cmd

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/google/pprof/profile"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/proto/pprofprofile"
)

type wireEdit struct {
	start, end  int
	replacement []byte
}

// writeSymbolizedProfileWire is private to the local batch append-only path.
// PRECONDITIONS: raw is the complete *uncompressed* protobuf already accepted by
// official ParseUncompressed + CheckValid (or ParseData); p is that same profile
// after only AddLine mutations. No sample/metadata/table reordering or editing.
// Original Functions and Location.Line prefixes must remain unchanged. This is
// not a generic Profile writer or validator. Legacy inputs pass nil raw.
func writeSymbolizedProfileWire(w io.Writer, raw []byte, p *profile.Profile) error {
	edits, newStrings, suffix, ok := symbolizedWireEdits(raw, p)
	if !ok {
		return p.WriteUncompressed(w)
	}
	bw := bufio.NewWriterSize(w, 64<<10)
	from := 0
	for _, e := range edits {
		if _, err := bw.Write(raw[from:e.start]); err != nil {
			return err
		}
		if _, err := bw.Write(e.replacement); err != nil {
			return err
		}
		from = e.end
	}
	if _, err := bw.Write(raw[from:]); err != nil {
		return err
	}
	// Planning and all fallback decisions are complete before the first write.
	// Strings stay referenced in intern order, never copied into a giant suffix.
	var scratch [11]byte
	for _, s := range newStrings {
		header := protowire.AppendTag(scratch[:0], 6, protowire.BytesType)
		header = protowire.AppendVarint(header, uint64(len(s)))
		if _, err := bw.Write(header); err != nil {
			return err
		}
		if _, err := bw.WriteString(s); err != nil {
			return err
		}
	}
	if _, err := bw.Write(suffix); err != nil {
		return err
	}
	return bw.Flush()
}

func symbolizedWireEdits(raw []byte, p *profile.Profile) (edits []wireEdit, newStrings []string, suffix []byte, ok bool) {
	if len(raw) == 0 {
		return
	}
	strings := make(map[string]int64)
	var stringCount int64
	functions, locations := 0, 0
	for at := 0; at < len(raw); {
		tag, kind, n := protowire.ConsumeTag(raw[at:])
		if n < 0 {
			return nil, nil, nil, false
		}
		size := protowire.ConsumeFieldValue(tag, kind, raw[at+n:])
		if size < 0 {
			return nil, nil, nil, false
		}
		end := at + n + size
		var payload []byte
		if tag >= 4 && tag <= 6 {
			if kind != protowire.BytesType {
				return nil, nil, nil, false
			}
			payload, n = protowire.ConsumeBytes(raw[at+n : end])
			if n < 0 {
				return nil, nil, nil, false
			}
		}
		switch tag {
		case 4:
			if locations >= len(p.Location) {
				return nil, nil, nil, false
			}
			loc := p.Location[locations]
			locations++
			var encoded pprofprofile.Location
			if err := proto.Unmarshal(payload, &encoded); err != nil {
				return nil, nil, nil, false
			}
			oldLines := len(encoded.Line)
			if loc == nil || loc.ID != encoded.Id || oldLines > len(loc.Line) {
				return nil, nil, nil, false
			}
			if oldLines < len(loc.Line) {
				// Preserve producer bytes; encode only the appended Line fields.
				encoded.Reset()
				for _, line := range loc.Line[oldLines:] {
					if line.Function == nil {
						return nil, nil, nil, false
					}
					encoded.Line = append(encoded.Line, &pprofprofile.Line{
						FunctionId: line.Function.ID,
						Line:       line.Line,
						Column:     line.Column,
					})
				}
				data, err := (proto.MarshalOptions{}).MarshalAppend(append([]byte(nil), payload...), &encoded)
				if err != nil {
					return nil, nil, nil, false
				}
				replacement := protowire.AppendBytes(protowire.AppendTag(nil, 4, protowire.BytesType), data)
				edits = append(edits, wireEdit{at, end, replacement})
			}
		case 5:
			functions++
		case 6:
			s := string(payload)
			// A nonzero index pointing at "" can keep an otherwise empty
			// label alive on Parse; official preEncode normalizes it away.
			// Avoid scanning Samples by rejecting this rare table shape.
			if s == "" && stringCount != 0 {
				return nil, nil, nil, false
			}
			if _, exists := strings[s]; !exists {
				strings[s] = stringCount
			}
			stringCount++ // Not len(map): producer duplicates retain their indices.
		}
		at = end
	}
	if locations != len(p.Location) || functions > len(p.Function) || stringCount == 0 {
		return nil, nil, nil, false
	}
	intern := func(s string) int64 {
		if id, found := strings[s]; found {
			return id
		}
		id := stringCount
		stringCount++
		strings[s] = id
		newStrings = append(newStrings, s)
		return id
	}
	var encoded pprofprofile.Function
	for _, f := range p.Function[functions:] {
		if f == nil {
			return nil, nil, nil, false
		}
		encoded.Id = f.ID
		encoded.Name = intern(f.Name)
		encoded.SystemName = intern(f.SystemName)
		encoded.Filename = intern(f.Filename)
		encoded.StartLine = f.StartLine
		data, err := proto.Marshal(&encoded)
		if err != nil {
			return nil, nil, nil, false
		}
		suffix = protowire.AppendBytes(protowire.AppendTag(suffix, 5, protowire.BytesType), data)
	}
	return edits, newStrings, suffix, true
}

// parseLocalBatchProfileWire retains protobuf only after full official decoding
// and validation. Normal protobuf inputs are inflated once; legacy/error fallback
// must use the ORIGINAL bytes so nested gzip is not accidentally accepted.
func parseLocalBatchProfileWire(input []byte) (*profile.Profile, []byte, error) {
	rawPB := input
	if len(input) >= 2 && input[0] == 0x1f && input[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(input))
		if err != nil {
			return nil, nil, fmt.Errorf("decompressing profile: %v", err)
		}
		rawPB, err = io.ReadAll(gz) // Through EOF: includes trailer/CRC validation.
		_ = gz.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("decompressing profile: %v", err)
		}
	}
	p, err := profile.ParseUncompressed(rawPB)
	if err != nil {
		p, err = profile.ParseData(input)
		return p, nil, err
	}
	if err := p.CheckValid(); err != nil {
		return nil, nil, fmt.Errorf("malformed profile: %v", err)
	}
	return p, rawPB, nil
}
