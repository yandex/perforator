package build_flags

// one build flag
//go:generate mockgen -destination=mock1/interfaces_mock.go -build_flags=-tags=tag1 . Interface
// multiple build flags
//go:generate mockgen -destination=mock2/interfaces_mock.go "-build_flags=-race -tags=tag2" . Interface
