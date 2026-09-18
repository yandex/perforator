package cprofile

// #include <perforator/lib/profile/c/flamegraph.h>
// #include <stdlib.h>
import "C"

import (
	"unsafe"

	"google.golang.org/protobuf/proto"

	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

////////////////////////////////////////////////////////////////////////////////

func RenderFlameGraph(profile *Profile, opts *profilepb.RenderOptions) ([]byte, error) {
	var str C.TPerforatorString

	optsPtr, optsSize, done := marshalOptions(opts)
	defer done()

	perr := C.PerforatorRenderFlameGraph(profile.ptr, optsPtr, optsSize, &str)
	if err := unwrap(perr); err != nil {
		return nil, err
	}
	defer C.PerforatorStringDispose(str)

	return copyCBytes(
		unsafe.Pointer(C.PerforatorStringData(str)),
		uintptr(C.PerforatorStringSize(str)),
	)
}

func RenderFlameGraphFromPProf(data []byte, opts *profilepb.RenderOptions) ([]byte, error) {
	ptr, size, done := cgobuf(data)
	defer done()

	optsPtr, optsSize, optsDone := marshalOptions(opts)
	defer optsDone()

	var str C.TPerforatorString
	perr := C.PerforatorRenderFlameGraphFromPProf(ptr, size, optsPtr, optsSize, &str)
	if err := unwrap(perr); err != nil {
		return nil, err
	}
	defer C.PerforatorStringDispose(str)

	return copyCBytes(
		unsafe.Pointer(C.PerforatorStringData(str)),
		uintptr(C.PerforatorStringSize(str)),
	)
}

func marshalOptions(opts *profilepb.RenderOptions) (*C.char, C.size_t, func()) {
	if opts == nil {
		return nil, 0, func() {}
	}
	optsBuf, err := proto.Marshal(opts)
	if err != nil {
		return nil, 0, func() {}
	}
	return cgobuf(optsBuf)
}

////////////////////////////////////////////////////////////////////////////////
