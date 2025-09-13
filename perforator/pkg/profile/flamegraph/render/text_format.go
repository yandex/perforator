package render

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/profile/labels"
	"github.com/yandex/perforator/perforator/pkg/tls"
)

const (
	zeroIndent   = ""
	singleIndent = "  "
	doubleIndent = "    "
)

// TextFormatRenderer handles rendering profiles in a human-readable text format
type TextFormatRenderer struct {
	format Format

	locationFrameOptions LocationFrameOptions
	maxSamples           int // Maximum number of samples to render, 0 and less means no limit

	locationsCache map[locationMeta][]locationData
	profile        *pprof.Profile
}

func NewTextFormatRenderer() *TextFormatRenderer {
	return &TextFormatRenderer{
		locationFrameOptions: LocationFrameOptions{
			FileNames:      true,
			FilePathPrefix: "",
		},
		locationsCache: make(map[locationMeta][]locationData),
		format:         PlainTextFormat,
		maxSamples:     0,
	}
}

func (t *TextFormatRenderer) SetMaxSamples(maxSamples int) {
	t.maxSamples = maxSamples
}

func (t *TextFormatRenderer) SetLineNumbers(value bool) {
	t.locationFrameOptions.LineNumbers = value
}

func (t *TextFormatRenderer) SetFileNames(value bool) {
	t.locationFrameOptions.FileNames = value
}

func (t *TextFormatRenderer) SetAddressRenderPolicy(policy AddressRenderPolicy) {
	t.locationFrameOptions.AddressPolicy = policy
}

func (t *TextFormatRenderer) SetFormat(format Format) {
	t.format = format
}

func (t *TextFormatRenderer) AddProfile(profile *pprof.Profile) error {
	t.profile = profile
	return nil
}

func (t *TextFormatRenderer) Render(w io.Writer) error {
	if t.profile == nil {
		return fmt.Errorf("no profile to render")
	}
	return t.renderProfile(t.profile, w)
}

func (t *TextFormatRenderer) RenderBytes() ([]byte, error) {
	var buf bytes.Buffer
	err := t.Render(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TextFormatRenderer) RenderPProf(profile *pprof.Profile, w io.Writer) error {
	if err := t.AddProfile(profile); err != nil {
		return err
	}
	return t.Render(w)
}

func (t *TextFormatRenderer) renderProfile(p *pprof.Profile, w io.Writer) error {
	switch t.format {
	case PlainTextFormat:
		return t.renderToPlainText(p, w)
	case JSONFormat:
		return t.renderToJsonText(p, w)
	default:
		return fmt.Errorf("unsupported format: %s", t.format)
	}
}

func (t *TextFormatRenderer) renderToJsonText(p *pprof.Profile, w io.Writer) error {
	return errors.New("not supported yet")
}

func (t *TextFormatRenderer) writeProcessInfo(indent string, sample *pprof.Sample, w io.Writer) error {
	procinfo := labels.ExtractProcessInfo(sample)

	if procinfo.ProcessName != "" && procinfo.Pid != nil {
		_, err := fmt.Fprintf(w, "%s%s %d (process)\n", indent, procinfo.ProcessName, *procinfo.Pid)
		if err != nil {
			return err
		}
	} else {
		if pid := procinfo.Pid; pid != nil {
			_, err := fmt.Fprintf(w, "%s%d (process)\n", indent, *pid)
			if err != nil {
				return err
			}
		}
		if name := procinfo.ProcessName; name != "" {
			_, err := fmt.Fprintf(w, "%s%s (process)\n", indent, name)
			if err != nil {
				return err
			}
		}
	}

	if name := procinfo.ThreadName; name != "" {
		_, err := fmt.Fprintf(w, "%s%s (thread)\n", indent, name)
		if err != nil {
			return err
		}
	}
	for _, signal := range sample.Label["signal:name"] {
		_, err := fmt.Fprintf(w, "%s%s (signal)\n", indent, signal)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TextFormatRenderer) writeLabels(indent string, sample *pprof.Sample, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%sLabels:\n", indent)
	if err != nil {
		return err
	}
	for k, values := range sample.Label {
		// Skip environment variables and thread local variables
		if _, ok := env.BuildEnvKeyFromLabelKey(k); ok {
			continue
		}
		if _, ok := tls.BuildTLSKeyFromLabelKey(k); ok {
			continue
		}

		for _, v := range values {
			_, err := fmt.Fprintf(w, "%s%s: %s\n", indent+singleIndent, k, v)
			if err != nil {
				return err
			}
		}
	}

	for k, values := range sample.NumLabel {
		for _, v := range values {
			_, err := fmt.Fprintf(w, "%s%s: %v\n", indent+singleIndent, k, v)
			if err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}

	return nil
}

func (t *TextFormatRenderer) writeEnvironmentVariables(indent string, sample *pprof.Sample, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%sEnvironment variables:\n", indent)
	if err != nil {
		return err
	}
	for k, values := range sample.Label {
		if name, ok := env.BuildEnvKeyFromLabelKey(k); ok {
			for _, v := range values {
				_, err := fmt.Fprintf(w, "%s%s: %s\n", indent+singleIndent, name, v)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}

	return nil
}

func (t *TextFormatRenderer) writeThreadLocalVariables(indent string, sample *pprof.Sample, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%sThread local variables:\n", indent)
	if err != nil {
		return err
	}

	for k, values := range sample.Label {
		if name, ok := tls.BuildTLSKeyFromLabelKey(k); ok {
			for _, v := range values {
				_, err := fmt.Fprintf(w, "%s%s: %s\n", indent+singleIndent, name, v)
				if err != nil {
					return err
				}
			}
		}
	}
	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}

	return nil
}

func (t *TextFormatRenderer) writeStackTrace(indent string, sample *pprof.Sample, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%sStack Trace (most recent call first):\n", indent)
	if err != nil {
		return err
	}

	index := 1
	for _, loc := range sample.Location {
		frames := t.getLocationFramesCached(loc)
		if len(frames) == 0 {
			// Location with no frames
			mapping := "<unknown>"
			if loc.Mapping != nil && loc.Mapping.File != "" {
				mapping = loc.Mapping.File
			}
			_, err := fmt.Fprintf(w, "%s%d: ?? [%s] {%#x}\n", indent+singleIndent, index, mapping, loc.Address)
			if err != nil {
				return err
			}
			index++
			continue
		}

		for _, frame := range frames {
			_, err := fmt.Fprintf(w, "%s%d: %s", indent+singleIndent, index, frame.name)
			if err != nil {
				return err
			}
			if frame.file != "" {
				_, err := fmt.Fprintf(w, " %s", frame.file)
				if err != nil {
					return err
				}
			}
			if frame.inlined {
				_, err := fmt.Fprintf(w, " (inlined)")
				if err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(w, "\n")
			if err != nil {
				return err
			}
			index++
		}
	}

	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		return err
	}
	return nil
}

func (t *TextFormatRenderer) renderToPlainText(p *pprof.Profile, w io.Writer) error {
	defer t.clearLocationsCache()

	sampleIndex := 0
	for i, st := range p.SampleType {
		if st.Type == p.DefaultSampleType {
			sampleIndex = i
			break
		}
	}

	_, err := fmt.Fprintf(w, "Profile Sample Type: %s (%s)\n", p.DefaultSampleType, p.SampleType[sampleIndex].Unit)
	if err != nil {
		return err
	}

	samplesToRender := len(p.Sample)
	if t.maxSamples > 0 && t.maxSamples < len(p.Sample) {
		samplesToRender = t.maxSamples
	}

	for i := 0; i < samplesToRender; i++ {
		sample := p.Sample[i]
		_, err := fmt.Fprintf(w, "Sample #%d:\n", i+1)
		if err != nil {
			return err
		}

		err = t.writeProcessInfo(zeroIndent, sample, w)
		if err != nil {
			return err
		}

		err = t.writeLabels(singleIndent, sample, w)
		if err != nil {
			return err
		}

		err = t.writeEnvironmentVariables(singleIndent, sample, w)
		if err != nil {
			return err
		}

		err = t.writeThreadLocalVariables(singleIndent, sample, w)
		if err != nil {
			return err
		}

		err = t.writeStackTrace(singleIndent, sample, w)
		if err != nil {
			return err
		}
	}

	if samplesToRender < len(p.Sample) {
		_, err := fmt.Fprintf(w, "\nProfile was truncated: showing %d out of %d samples\n", samplesToRender, len(p.Sample))
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TextFormatRenderer) getLocationFrames(loc *pprof.Location) []locationData {
	frames := getLocationFrames(loc, t.locationFrameOptions)

	// because inlined functions are reversed.
	slices.Reverse(frames)

	return frames
}

func (t *TextFormatRenderer) getLocationFramesCached(loc *pprof.Location) []locationData {
	if loc.Mapping == nil {
		return t.getLocationFrames(loc)
	}

	meta := locationMeta{
		address:   loc.Address,
		mappingID: loc.Mapping.ID,
	}
	frames, found := t.locationsCache[meta]
	if !found {
		frames = t.getLocationFrames(loc)
		t.locationsCache[meta] = frames
	}

	return frames
}

func (t *TextFormatRenderer) clearLocationsCache() {
	t.locationsCache = make(map[locationMeta][]locationData)
}
