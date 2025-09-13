package cprofile

// #include <perforator/lib/profile/c/error.h>
import "C"
import "fmt"

func unwrap(err C.TPerforatorError) error {
	if err == nil {
		return nil
	}

	s := C.GoString(C.PerforatorErrorString(err))
	C.PerforatorErrorDispose(err)

	return fmt.Errorf("cgo call failed: %s", s)
}
