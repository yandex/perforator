package cobrabuildinfo

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/buildinfo"
)

func make() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildinfo.Dump(os.Stdout)
		},
	}
}

func Init(cmd *cobra.Command) {
	cmd.AddCommand(make())
}
