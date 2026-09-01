package autofdo

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/autofdo/autofdo_c.h>
import "C"
import (
	"fmt"
	"unsafe"
)

type BatchInputBuilder struct {
	builder    unsafe.Pointer
	binaryPath string
}

type AutofdoMetadata struct {
	TotalProfiles uint64

	TotalSamples    uint64
	TotalBranches   uint64
	BogusLbrEntries uint64

	BranchCountMapSize  uint64
	RangeCountMapSize   uint64
	AddressCountMapSize uint64

	ProfilesCountByService map[string]uint64
}

type ProcessedLBRData struct {
	AutofdoInput string
	BoltInput    string
	MetaData     AutofdoMetadata
}

func NewBatchInputBuilder(builders uint64, buildID, binaryPath string) (*BatchInputBuilder, error) {
	cBuildID := C.CString(buildID)
	defer C.free(unsafe.Pointer(cBuildID))
	cBinaryPath := C.CString(binaryPath)
	defer C.free(unsafe.Pointer(cBinaryPath))

	var cError C.TPerforatorError
	builder := C.MakeBatchBuilder(C.ui64(builders), cBuildID, cBinaryPath, &cError)
	if cError != nil {
		message := C.GoString(C.PerforatorErrorString(cError))
		C.PerforatorErrorDispose(cError)
		return nil, fmt.Errorf("failed to initialize profile address conversion: %s", message)
	}

	return &BatchInputBuilder{
		builder:    builder,
		binaryPath: binaryPath,
	}, nil
}

func (b *BatchInputBuilder) Destroy() {
	C.DestroyBatchBuilder(b.builder)
}

func (b *BatchInputBuilder) AddProfile(builderIndex uint64, serviceName string, profileBytes []byte) error {
	if len(profileBytes) == 0 {
		return nil
	}

	cServiceName := C.CString(serviceName)
	defer C.free(unsafe.Pointer(cServiceName))

	cError := C.AddProfile(
		b.builder,
		C.ui64(builderIndex),
		cServiceName,
		(*C.char)(unsafe.Pointer(&profileBytes[0])),
		C.ui64(len(profileBytes)),
	)
	if cError != nil {
		message := C.GoString(C.PerforatorErrorString(cError))
		C.PerforatorErrorDispose(cError)
		return fmt.Errorf("failed to process profile addresses: %s", message)
	}

	return nil
}

func consumeProfilesByServiceMap(
	cProfilesByServiceMapLen C.ui64,
	cProfilesByServiceMapServices **C.char,
	cProfilesByServiceMapCounts *C.ui64,
) map[string]uint64 {
	servicesArray := (*[1 << 30]*C.char)(unsafe.Pointer(cProfilesByServiceMapServices))
	countsArray := (*[1 << 30]uint64)(unsafe.Pointer(cProfilesByServiceMapCounts))

	defer func() {
		for i := uint64(0); i < uint64(cProfilesByServiceMapLen); i += 1 {
			C.free(unsafe.Pointer(servicesArray[i]))
		}
		C.free(unsafe.Pointer(cProfilesByServiceMapServices))

		C.free(unsafe.Pointer(cProfilesByServiceMapCounts))
	}()

	result := make(map[string]uint64, int(cProfilesByServiceMapLen))
	for i := uint64(0); i < uint64(cProfilesByServiceMapLen); i += 1 {
		result[C.GoString(servicesArray[i])] += countsArray[i]
	}

	return result
}

// Finalize converts addresses produced from Perforator pprof/yaprof
// into the coordinate systems expected by AutoFDO and BOLT.
func (b *BatchInputBuilder) Finalize() (ProcessedLBRData, error) {
	cBinaryPath := C.CString(b.binaryPath)
	defer C.free(unsafe.Pointer(cBinaryPath))
	var totalProfiles C.ui64
	var totalBranches, totalSamples, bogusLbrEntries C.ui64
	var branchCountMapSize, rangeCountMapSize, addressCountMapSize C.ui64

	var profilesByServiceMapLen C.ui64
	var profilesByServiceMapServices **C.char
	var profilesByServiceMapCounts *C.ui64

	var cAutofdoInput, cBoltInput *C.char

	cError := C.Finalize(
		b.builder,
		cBinaryPath,
		// metadata
		&totalProfiles,
		&totalBranches,
		&totalSamples,
		&bogusLbrEntries,
		&branchCountMapSize,
		&rangeCountMapSize,
		&addressCountMapSize,
		// profiles count by service
		&profilesByServiceMapLen,
		&profilesByServiceMapServices,
		&profilesByServiceMapCounts,
		// output
		&cAutofdoInput, &cBoltInput,
	)
	if cError != nil {
		message := C.GoString(C.PerforatorErrorString(cError))
		C.PerforatorErrorDispose(cError)
		return ProcessedLBRData{}, fmt.Errorf("failed to convert profile addresses: %s", message)
	}
	defer C.free(unsafe.Pointer(cAutofdoInput))
	defer C.free(unsafe.Pointer(cBoltInput))

	profilesCountByService := consumeProfilesByServiceMap(
		profilesByServiceMapLen,
		profilesByServiceMapServices,
		profilesByServiceMapCounts,
	)

	return ProcessedLBRData{
		AutofdoInput: C.GoString(cAutofdoInput),
		BoltInput:    C.GoString(cBoltInput),
		MetaData: AutofdoMetadata{
			TotalProfiles:          uint64(totalProfiles),
			TotalBranches:          uint64(totalBranches),
			TotalSamples:           uint64(totalSamples),
			BogusLbrEntries:        uint64(bogusLbrEntries),
			BranchCountMapSize:     uint64(branchCountMapSize),
			RangeCountMapSize:      uint64(rangeCountMapSize),
			AddressCountMapSize:    uint64(addressCountMapSize),
			ProfilesCountByService: profilesCountByService,
		},
	}, nil
}

func GetBinaryExecutableBytes(binaryPath string) (uint64, error) {
	cBinaryPath := C.CString(binaryPath)
	defer C.free(unsafe.Pointer(cBinaryPath))

	return uint64(C.GetBinaryExecutableBytes(cBinaryPath)), nil
}

///////////////////////////////////////////////////////////////////////////////////////////

type BatchBuildIdGuesser struct {
	guesser unsafe.Pointer
}

func NewBuildIDGuesser(guessers uint64) (*BatchBuildIdGuesser, error) {
	guesser := C.MakeBatchBuildIdGuesser(C.ui64(guessers))

	return &BatchBuildIdGuesser{
		guesser: guesser,
	}, nil
}

func (g *BatchBuildIdGuesser) Destroy() {
	C.DestroyBatchBuildIdGuesser(g.guesser)
}

func (g *BatchBuildIdGuesser) FeedProfile(guesserIndex uint64, profileBytes []byte) error {
	if len(profileBytes) == 0 {
		return nil
	}

	C.FeedProfileIntoGuesser(
		g.guesser,
		C.ui64(guesserIndex),
		(*C.char)(unsafe.Pointer(&profileBytes[0])),
		C.ui64(len(profileBytes)))

	return nil
}

func (g *BatchBuildIdGuesser) GuessBuildID() (string, error) {
	cBuildID := C.TryGuessBuildID(g.guesser)
	if cBuildID == nil {
		return "", fmt.Errorf("Failed to guess buildid")
	}

	defer C.free(unsafe.Pointer(cBuildID))

	return C.GoString(cBuildID), nil
}
