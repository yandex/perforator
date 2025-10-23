package main

import (
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/validateconfig"
)

func init() {
	gcCmd.AddCommand(validateconfig.NewValidateConfigCmd(
		"GC",
		validateconfig.ValidateConfigFunc(
			func(configPath string) error {
				_, err := bundle.ParseConfig(configPath, true /* strict */)
				return err
			}),
	))
}
