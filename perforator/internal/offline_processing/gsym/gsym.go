package gsym

import (
	// #include <stdlib.h>
	// #include <perforator/symbolizer/lib/gsym/gsym.h>
	"C"
)
import (
	"fmt"
	"unsafe"
)

func ConvertDWARFToGsym(input string, output string, convertNumThreads uint32) error {
	cInput := C.CString(input)
	defer C.free(unsafe.Pointer(cInput))

	cOutput := C.CString(output)
	defer C.free(unsafe.Pointer(cOutput))

	cError := C.ConvertDWARFToGSYM(cInput, cOutput, C.ui32(convertNumThreads))
	defer C.free(unsafe.Pointer(cError))
	if cError != nil {
		return fmt.Errorf("%s", C.GoString(cError))
	}

	return nil
}
