package validateconfig

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/must"
)

type ValidateConfigFunc func(configPath string) error

func NewValidateConfigCmd(
	componentName string,
	validateFunc ValidateConfigFunc,
) *cobra.Command {
	var configPath string
	validateConfigCmd := &cobra.Command{
		Use:   "validate-config",
		Short: fmt.Sprintf("Validate %s config", componentName),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			err := validateFunc(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
				os.Exit(1)
			}
		},
	}

	validateConfigCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to the config file")
	must.Must(validateConfigCmd.MarkFlagRequired("config"))
	return validateConfigCmd
}
