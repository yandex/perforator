package nativepipeline

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline/runtimewriter"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// LanguageSection contains an array of BPF frames, without the section header.
// Data uses the layout selected by Language: python_frame, php_frame, lua_frame
// or jvm_frame: see progs/unwinder/python/types.h, php/types.h, lua/types.h and
// jvm/api.h. pack_sample_lang produces them; nativeRuntimeResolver decodes the
// matching types generated in internal/unwinder.
// Data is read-only and borrowed until ResolveRuntimeFrames returns.
type LanguageSection struct {
	Language    Language
	PayloadKind uint8
	FrameSize   uint16
	Data        []byte
}

type RuntimeRequest struct {
	Key         linux.ProcessKey
	NativeStack []uint64
	Languages   []LanguageSection
}

// RuntimeResolver must not modify or retain the borrowed input or writer.
// The caller validates section buffers and known, unique language IDs.
// An error discards all output written during the call.
type RuntimeResolver interface {
	ResolveRuntimeFrames(ctx context.Context, request RuntimeRequest, writer *runtimewriter.Writer) error
}
