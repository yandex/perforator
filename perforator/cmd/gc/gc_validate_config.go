package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
)

var (
	gcConfigForValidationPath string

	gcValidateConfigCmd = &cobra.Command{
		Use:   "validate-config",
		Short: "Validate GC config",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := bundle.ParseConfig(gcConfigForValidationPath, true /* strict */)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%#v\n", config)
		},
	}
)

func init() {
	gcValidateConfigCmd.Flags().StringVar(&gcConfigForValidationPath, "config", "", "Path to the config file")
	must.Must(gcValidateConfigCmd.MarkFlagRequired("config"))
	gcCmd.AddCommand(gcValidateConfigCmd)
}
