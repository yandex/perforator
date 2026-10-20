package nativepipeline

import (
	"context"

	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

type Language uint8

// InterpreterFrameRequest identifies a frame in a process. Data is
// borrowed and valid only until ResolveInterpreterFrames returns.
type InterpreterFrameRequest struct {
	Language Language
	Key      linux.ProcessKey
	Data     []byte
}

type ResolvedInterpreterFrame = models.ResolvedInterpreterFrame

type JITRuntimeID uint8

type NativeFrameRequest struct {
	Key     linux.ProcessKey
	Address uint64
	// Optional method metadata; JIT lookup uses Address regardless of runtime ID.
	JITRuntimeID     JITRuntimeID
	JITMethodAddress uint64
}

type ResolvedNativeFrame struct {
	JITMapping string
	JITSymbols []string
	JVMName    string
	// Targets that preserve the sampled address ignore ClearAddress.
	ClearAddress bool
}

// RuntimeResolver returns one result per request in the same order. Methods
// must not retain requests or their payloads. Returned strings and slices must
// remain immutable; a resolution can be retained for an arena resize retry.
type RuntimeResolver interface {
	// A batch error discards all results; unresolved frames use Unsymbolized.
	ResolveInterpreterFrames(ctx context.Context, requests []InterpreterFrameRequest) ([]ResolvedInterpreterFrame, error)
	ResolveNativeFrames(ctx context.Context, requests []NativeFrameRequest) []ResolvedNativeFrame
}
