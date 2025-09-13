// Package add_generate_directive makes sure output places the go:generate command as a directive in the generated code.
package add_generate_directive

type Message struct {
	Text string
}

type Foo interface {
	Bar(channels []string, message chan<- Message)
}
