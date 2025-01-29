package maxprocs

import (
	"fmt"
	"os"

	"go.uber.org/automaxprocs/maxprocs"
)

func Adjust() {
	_, err := maxprocs.Set()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set GOMAXPROCS: %v\n", err)
	}
}
