package cprofile

// #include <perforator/lib/profile/c/profile.h>
// #include <stdlib.h>
import "C"

import (
	"runtime"
	"unsafe"
)

////////////////////////////////////////////////////////////////////////////////

type Profile struct {
	ptr C.TPerforatorProfile
}

func newProfile(profile C.TPerforatorProfile) *Profile {
	res := &Profile{profile}
	runtime.SetFinalizer(res, func(m *Profile) {
		m.Free()
	})
	return res
}

func Parse(data []byte) (*Profile, error) {
	return parseVia(data, func(ptr *C.char, size C.size_t, profile *C.TPerforatorProfile) C.TPerforatorError {
		return C.PerforatorProfileParse(ptr, size, profile)
	})
}

func ParsePProf(data []byte) (*Profile, error) {
	return parseVia(data, func(ptr *C.char, size C.size_t, profile *C.TPerforatorProfile) C.TPerforatorError {
		return C.PerforatorProfileParsePProf(ptr, size, profile)
	})
}

func parseVia(data []byte, parser func(ptr *C.char, size C.size_t, profile *C.TPerforatorProfile) C.TPerforatorError) (*Profile, error) {
	ptr, size, done := cgobuf(data)
	defer done()

	var profile C.TPerforatorProfile
	perr := parser(ptr, size, &profile)
	if err := unwrap(perr); err != nil {
		return nil, err
	}

	return newProfile(profile), nil
}

func (p *Profile) Free() {
	if p.ptr != nil {
		C.PerforatorProfileDispose(p.ptr)
		p.ptr = nil
	}
}

func (p *Profile) Marshal() ([]byte, error) {
	return p.marshalVia(func(profile C.TPerforatorProfile, result *C.TPerforatorString) C.TPerforatorError {
		return C.PerforatorProfileSerialize(profile, result)
	})
}

func (p *Profile) MarshalPProf() ([]byte, error) {
	return p.marshalVia(func(profile C.TPerforatorProfile, result *C.TPerforatorString) C.TPerforatorError {
		return C.PerforatorProfileSerializePProf(profile, result)
	})
}

func (p *Profile) marshalVia(marshaller func(profile C.TPerforatorProfile, result *C.TPerforatorString) C.TPerforatorError) ([]byte, error) {
	var str C.TPerforatorString
	perr := marshaller(p.ptr, &str)
	if err := unwrap(perr); err != nil {
		return nil, err
	}
	defer C.PerforatorStringDispose(str)

	res := C.GoBytes(
		unsafe.Pointer(C.PerforatorStringData(str)),
		C.int(C.PerforatorStringSize(str)),
	)
	return res, nil
}

////////////////////////////////////////////////////////////////////////////////
