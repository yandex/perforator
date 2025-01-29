package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
)

var (
	rootCmd = &cobra.Command{
		Use:           "offline_processing",
		Short:         "Process binaries offline",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	cobrabuildinfo.Init(rootCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
