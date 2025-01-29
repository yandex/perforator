package main

import (
	"os"

	"github.com/yandex/perforator/perforator/pkg/linux/vdso"
)

func main() {
	buf, err := vdso.LoadVDSO()
	if err != nil {
		panic(err)
	}

	_, err = os.Stdout.Write(buf)
	if err != nil {
		panic(err)
	}
}
