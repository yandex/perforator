package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/xelf"
)

var (
	rootCmd = &cobra.Command{
		Use:           "buildid",
		Short:         "Calculate buildids",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args[0])
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func run(path string) error {
	id, err := xelf.GetBuildID(path)
	if err != nil {
		return err
	}

	fmt.Printf("Found buildid %s\n", id)

	return nil
}
