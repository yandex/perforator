package autofdo

// #include <stdlib.h>
// #include <perforator/symbolizer/lib/autofdo/autofdo_c.h>
import "C"
import (
	"fmt"
	"unsafe"
)

type BatchInputBuilder struct {
	builder unsafe.Pointer
}

type AutofdoMetadata struct {
	TotalProfiles uint64

	TotalSamples    uint64
	TotalBranches   uint64
	BogusLbrEntries uint64

	BranchCountMapSize  uint64
	RangeCountMapSize   uint64
	AddressCountMapSize uint64
}

type ProcessedLBRData struct {
	AutofdoInput string
	BoltInput    string
	MetaData     AutofdoMetadata
}

func NewBatchInputBuilder(builders uint64, buildID string) (*BatchInputBuilder, error) {
	cBuildID := C.CString(buildID)
	defer C.free(unsafe.Pointer(cBuildID))

	builder := C.MakeBatchBuilder(C.ui64(builders), cBuildID)

	return &BatchInputBuilder{
		builder: builder,
	}, nil
}

func (b *BatchInputBuilder) Destroy() {
	C.DestroyBatchBuilder(b.builder)
}

func (b *BatchInputBuilder) AddProfile(builderIndex uint64, profileBytes []byte) error {
	if len(profileBytes) == 0 {
		return nil
	}

	C.AddProfile(
		b.builder,
		C.ui64(builderIndex),
		(*C.char)(unsafe.Pointer(&profileBytes[0])),
		C.ui64(len(profileBytes)),
	)

	return nil
}

func (b *BatchInputBuilder) Finalize() (ProcessedLBRData, error) {
	var totalProfiles C.ui64
	var totalBranches, totalSamples, bogusLbrEntries C.ui64
	var branchCountMapSize, rangeCountMapSize, addressCountMapSize C.ui64
	var cAutofdoInput, cBoltInput *C.char

	C.Finalize(
		b.builder,
		// metadata
		&totalProfiles,
		&totalBranches,
		&totalSamples,
		&bogusLbrEntries,
		&branchCountMapSize,
		&rangeCountMapSize,
		&addressCountMapSize,
		// output
		&cAutofdoInput, &cBoltInput,
	)
	defer C.free(unsafe.Pointer(cAutofdoInput))
	defer C.free(unsafe.Pointer(cBoltInput))

	return ProcessedLBRData{
		AutofdoInput: C.GoString(cAutofdoInput),
		BoltInput:    C.GoString(cBoltInput),
		MetaData: AutofdoMetadata{
			TotalProfiles:       uint64(totalProfiles),
			TotalBranches:       uint64(totalBranches),
			TotalSamples:        uint64(totalSamples),
			BogusLbrEntries:     uint64(bogusLbrEntries),
			BranchCountMapSize:  uint64(branchCountMapSize),
			RangeCountMapSize:   uint64(rangeCountMapSize),
			AddressCountMapSize: uint64(addressCountMapSize),
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
