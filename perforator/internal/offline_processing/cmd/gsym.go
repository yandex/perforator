package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/offline_processing/gsym"
	"github.com/yandex/perforator/perforator/pkg/must"
)

var (
	input             string
	output            string
	convertNumThreads uint32

	dwarfToGSYMCommand = &cobra.Command{
		Use:   "dwarf-to-gsym",
		Short: "Convert DWARF to GSYM",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gsym.ConvertDWARFToGsym(input, output, convertNumThreads)
		},
	}
)

func init() {
	dwarfToGSYMCommand.Flags().StringVarP(
		&input,
		"input",
		"i",
		"",
		"path to input binary",
	)
	must.Must(dwarfToGSYMCommand.MarkFlagFilename("input"))

	dwarfToGSYMCommand.Flags().StringVarP(
		&output,
		"output",
		"o",
		"",
		"path to output gsym file",
	)
	must.Must(dwarfToGSYMCommand.MarkFlagFilename("output"))

	dwarfToGSYMCommand.Flags().Uint32VarP(
		&convertNumThreads,
		"num-threads",
		"n",
		4,
		"number of simultaneous threads to use when converting files to GSYM",
	)

	must.Must(dwarfToGSYMCommand.MarkFlagRequired("input"))
	must.Must(dwarfToGSYMCommand.MarkFlagRequired("output"))

	rootCmd.AddCommand(dwarfToGSYMCommand)
}
