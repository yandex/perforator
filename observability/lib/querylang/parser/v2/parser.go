package parserv2

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/parser/v2/generated"
)

type parserImpl struct{}

func NewParser() querylang.Parser {
	return &parserImpl{}
}

func (p *parserImpl) ParseSelector(query string) (*querylang.Selector, error) {
	l := newListener()

	is := antlr.NewInputStream(query)
	lexer := parser.NewSolomonLexer(is)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(l)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	ssp := parser.NewSolomonSelectorParser(stream)
	ssp.RemoveErrorListeners()
	ssp.AddErrorListener(l)

	antlr.ParseTreeWalkerDefault.Walk(l, ssp.Selectors())

	return l.root, l.Err()
}

var _ querylang.Parser = (*parserImpl)(nil)
