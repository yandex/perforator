package symbolize

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/stacks_sampling/stacks_sampling_c.h>
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/yandex/perforator/perforator/pkg/storage/profile"
)

type StacksSampler struct {
	sampler unsafe.Pointer
}

func NewStacksSampler(rate uint64) (*StacksSampler, error) {
	return &StacksSampler{
		sampler: C.CreateAggregatingStacksSampler(C.ui64(rate)),
	}, nil
}

func (s *StacksSampler) Destroy() {
	C.DestroyAggregatingStacksSampler(s.sampler)
}

func (s *StacksSampler) AddProfile(profile profile.ProfileData) {
	C.AddProfileIntoAggregatingStacksSampler(
		s.sampler,
		(*C.char)(unsafe.Pointer(&profile[0])),
		C.ui64(len(profile)),
	)
}

func (s *StacksSampler) ExtractSampledProfile() (profile.ProfileData, error) {
	var profileDataC *C.char
	var profileDataLenC C.ui64
	var isEmptyC C.ui64

	C.ExtractResultingProfileFromSampler(s.sampler, &profileDataC, &profileDataLenC, &isEmptyC)
	defer C.free(unsafe.Pointer(profileDataC))

	if (profileDataC == nil || profileDataLenC == 0) && (isEmptyC != C.ui64(1)) {
		return nil, fmt.Errorf("failed to extract sampled profile")
	}

	if isEmptyC == C.ui64(1) {
		// It's not an error, it's just that we don't have any samples in the sampler.
		// The caller should filter such profiles out.
		return []byte{}, nil
	}
	return C.GoBytes(unsafe.Pointer(profileDataC), C.int(profileDataLenC)), nil
}
