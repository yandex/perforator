package cprofile

// #include <perforator/lib/profile/c/profile.h>
// #include <perforator/lib/profile/c/merge.h>
// #include <stdlib.h>
import "C"

import (
	"runtime"
	"unsafe"

	"google.golang.org/protobuf/proto"

	pb "github.com/yandex/perforator/perforator/proto/profile"
)

////////////////////////////////////////////////////////////////////////////////

type MergeManager struct {
	manager C.TPerforatorProfileMergeManager
}

func NewMergeManager(threadCount int) (*MergeManager, error) {
	var mgr C.TPerforatorProfileMergeManager
	err := C.PerforatorMakeMergeManager(C.int(threadCount), &mgr)
	if err := unwrap(err); err != nil {
		return nil, err
	}

	res := &MergeManager{mgr}
	runtime.SetFinalizer(res, func(m *MergeManager) {
		C.PerforatorDestroyMergeManager(m.manager)
	})

	return res, nil
}

func cgobuf(buf []byte) (ptr *C.char, size C.size_t, free func()) {
	ptr = (*C.char)(C.CBytes(buf))
	size = C.size_t(len(buf))
	free = func() {
		C.free(unsafe.Pointer(ptr))
	}
	return
}

func (m *MergeManager) Start(opts *pb.MergeOptions) (*MergeSession, error) {
	buf, err := proto.Marshal(opts)
	if err != nil {
		return nil, err
	}

	ptr, size, free := cgobuf(buf)
	defer free()

	var merger C.TPerforatorProfileMerger
	perr := C.PerforatorMergerStart(m.manager, ptr, size, &merger)
	if err := unwrap(perr); err != nil {
		return nil, err
	}

	session := &MergeSession{merger}
	runtime.SetFinalizer(session, func(m *MergeSession) {
		session.Close()
	})

	return session, nil
}

////////////////////////////////////////////////////////////////////////////////

type MergeSession struct {
	session C.TPerforatorProfileMerger
}

func (s *MergeSession) AddPProfProfile(data []byte) error {
	ptr, size, done := cgobuf(data)
	defer done()

	var profile C.TPerforatorProfile
	perr := C.PerforatorProfileParsePProf(ptr, size, &profile)
	if err := unwrap(perr); err != nil {
		return err
	}
	defer C.PerforatorProfileDispose(profile)

	perr = C.PerforatorMergerAddProfile(s.session, profile)
	if err := unwrap(perr); err != nil {
		return err
	}

	return nil
}

func (s *MergeSession) Finish() (*Profile, error) {
	var profile C.TPerforatorProfile
	perr := C.PerforatorMergerFinish(s.session, &profile)
	if err := unwrap(perr); err != nil {
		return nil, err
	}
	return newProfile(profile), nil
}

func (s *MergeSession) Close() {
	if s.session != nil {
		C.PerforatorMergerDispose(s.session)
		s.session = nil
	}
}

////////////////////////////////////////////////////////////////////////////////
