package buildinfo

import (
	"fmt"
	"io"

	"github.com/yandex/perforator/library/go/core/buildinfo"
)

func Dump(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s", buildinfo.Info.ProgramVersion)
	return err
}
