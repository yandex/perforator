package xpflag

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type OneOf struct {
	allowed []string
	value   string
}

// Set implements pflag.Value.
func (o *OneOf) Set(value string) error {
	if !slices.Contains(o.allowed, value) {
		return fmt.Errorf("unexpected value %q, expected one of [%v]", value, o.Variants())
	}
	o.value = value
	return nil
}

// String implements pflag.Value.
func (o *OneOf) String() string {
	return o.value
}

// Type implements pflag.Value.
func (o *OneOf) Type() string {
	return "string"
}

func (o *OneOf) Variants() string {
	return strings.Join(o.allowed, ", ")
}

func NewOneOf(defaul string, allowed ...string) *OneOf {
	return &OneOf{allowed, defaul}
}

// Allow to use OneOf flags in the cobra autocompletion framework.
// For example:
//
// loglevel := xpflag.New("info", "debug", "info", "warn", "error")
// cmd.Flags().Var(loglevel, "log-level", "log level, one of "+loglevel.Variants())
// cmd.RegisterFlagCompletionFunc(flagName, loglevel.Complete)
func (o *OneOf) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return o.allowed, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
}

var _ pflag.Value = (*OneOf)(nil)
