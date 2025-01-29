package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/must"
	storageserver "github.com/yandex/perforator/perforator/pkg/storage/server"
)

var (
	storageConfigForValidationPath string

	storageValidateConfigCmd = &cobra.Command{
		Use:   "validate-config",
		Short: "Validate storage config",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := storageserver.ParseConfig(storageConfigForValidationPath, true /* strict */)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%#v\n", config)
		},
	}
)

func init() {
	storageValidateConfigCmd.Flags().StringVar(&storageConfigForValidationPath, "config", "", "Path to the config file")
	must.Must(storageValidateConfigCmd.MarkFlagRequired("config"))
	storageCmd.AddCommand(storageValidateConfigCmd)
}
