package main

import (
	"github.com/yandex/perforator/perforator/internal/offline_processing/cmd"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
)

func main() {
	maxprocs.Adjust()
	cmd.Execute()
}
