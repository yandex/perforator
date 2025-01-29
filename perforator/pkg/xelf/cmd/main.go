package main

import (
	"fmt"
	"os"

	"github.com/yandex/perforator/perforator/pkg/xelf"
)

func main() {
	for _, path := range os.Args {
		id, err := xelf.GetBuildID(path)
		if err != nil {
			fmt.Printf("Failed to get buildid for file %s: %+v\n", path, err)
		} else {
			fmt.Printf("File %s has buildid %q\n", path, id)
		}
	}
}
