package xpflag

import (
	"github.com/spf13/pflag"
)

type Func struct {
	value string
	parse func(string) error
}

// Set implements pflag.Value.
func (o *Func) Set(value string) error {
	err := o.parse(value)
	if err != nil {
		return err
	}
	o.value = value
	return nil
}

// String implements pflag.Value.
func (o *Func) String() string {
	return o.value
}

// Type implements pflag.Value.
func (o *Func) Type() string {
	return "string"
}

func NewFunc(parse func(string) error) *Func {
	return &Func{parse: parse}
}

var _ pflag.Value = (*Func)(nil)
