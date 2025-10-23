package main

import (
	"github.com/yandex/perforator/perforator/internal/agent_gateway/server"
	"github.com/yandex/perforator/perforator/pkg/validateconfig"
)

func init() {
	storageCmd.AddCommand(validateconfig.NewValidateConfigCmd(
		"storage",
		validateconfig.ValidateConfigFunc(
			func(configPath string) error {
				_, err := server.ParseConfig(configPath, true /* strict */)
				return err
			},
		),
	),
	)
}
