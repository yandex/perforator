//go:build !linux
// +build !linux

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func makeRecordCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "record",
		Short: "record local profile (Linux only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("unfortunately, perforator record is only available on Linux")
		},
	}
}

func init() {
	rootCmd.AddCommand(makeRecordCommand())
}
