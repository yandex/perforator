package server

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

func RenderProfile(ctx context.Context, profile *profile.Profile, format *perforator.RenderFormat) ([]byte, error) {
	buf := new(bytes.Buffer)

	switch v := format.GetFormat().(type) {
	case *perforator.RenderFormat_RawProfile:
		if err := profile.Write(buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case *perforator.RenderFormat_Flamegraph:
		return buildProfileFlamegraph(profile, v.Flamegraph, render.HTMLFormat)
	case *perforator.RenderFormat_JSONFlamegraph:
		return buildProfileFlamegraph(profile, v.JSONFlamegraph, render.JSONFormat)
	}

	return nil, fmt.Errorf("unsupported render format %s", format.String())
}

func buildProfileFlamegraph(profile *profile.Profile, options *perforator.FlamegraphOptions, format render.Format) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0))

	flamegraph := render.NewFlameGraph()
	flamegraph.SetFormat(format)

	err := fillFlamegraphOptions(flamegraph, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fill flamegraph options: %w", err)
	}

	err = flamegraph.RenderPProf(profile, buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to render profile flamegraph: %w", err)
	}

	return buffer.Bytes(), nil
}

const (
	flamegraphDefaultMinWeight = 0.00005
	flamegraphDefaultMaxDepth  = 255
)

func fillFlamegraphOptions(fg *render.FlameGraph, options *perforator.FlamegraphOptions) error {
	if options == nil {
		return nil
	}

	if depth := options.MaxDepth; depth != nil {
		fg.SetDepthLimit(int(*depth))
	} else {
		fg.SetDepthLimit(flamegraphDefaultMaxDepth)
	}

	if weight := options.MinWeight; weight != nil {
		fg.SetMinWeight(*weight)
	} else {
		fg.SetMinWeight(flamegraphDefaultMinWeight)
	}

	if inverse := options.Inverse; inverse != nil {
		fg.SetInverted(*inverse)
	}

	if numbers := options.ShowLineNumbers; numbers != nil {
		fg.SetLineNumbers(*numbers)
	}

	if filenames := options.ShowFileNames; filenames != nil {
		fg.SetFileNames(*filenames)
	}

	switch options.GetRenderAddresses() {
	case perforator.FlamegraphOptions_RenderAddressesNever:
		fg.SetAddressRenderPolicy(render.RenderAddressesNever)
	case perforator.FlamegraphOptions_RenderAddressesUnsymbolized:
		fg.SetAddressRenderPolicy(render.RenderAddressesUnsymbolized)
	case perforator.FlamegraphOptions_RenderAddressesAlways:
		fg.SetAddressRenderPolicy(render.RenderAddressesAlways)
	default:
		return fmt.Errorf("unsupported address rendering policy %v", options.GetRenderAddresses())
	}

	return nil
}
