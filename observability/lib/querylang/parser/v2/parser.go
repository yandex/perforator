package parserv2

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/parser/v2/generated"
)

type parserImpl struct{}

func NewParser() querylang.Parser {
	return &parserImpl{}
}

func (p *parserImpl) ParseSelector(query string) (*querylang.Selector, error) {
	return parseSelector(query, false)
}

// ParseSelectorStrict requires the input to contain only one selector.
func ParseSelectorStrict(query string) (*querylang.Selector, error) {
	return parseSelector(query, true)
}

func parseSelector(query string, requireEOF bool) (*querylang.Selector, error) {
	l := newSelectorListener()

	is := antlr.NewInputStream(query)
	lexer := parser.NewSolomonLexer(is)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(l)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	ssp := parser.NewSolomonParser(stream)
	ssp.RemoveErrorListeners()
	ssp.AddErrorListener(l)

	antlr.ParseTreeWalkerDefault.Walk(l, ssp.Selectors())

	if requireEOF && !l.hasErrors() {
		if tok := stream.LT(1); tok != nil && tok.GetTokenType() != antlr.TokenEOF {
			l.onSyntaxError(fmt.Errorf(
				"unexpected token '%s' (at line %d, column %d)",
				tok.GetText(), tok.GetLine(), tok.GetColumn(),
			))
		}
	}

	return l.root, l.getError()
}

func (p *parserImpl) ParseExpression(query string) (*querylang.Expression, error) {
	l := newExpressionListener()

	is := antlr.NewInputStream(query)
	lexer := parser.NewSolomonLexer(is)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(l)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	ssp := parser.NewSolomonParser(stream)
	ssp.RemoveErrorListeners()
	ssp.AddErrorListener(l)

	antlr.ParseTreeWalkerDefault.Walk(l, ssp.Expression())

	if !l.hasErrors() {
		if tok := stream.LT(1); tok != nil && tok.GetTokenType() != antlr.TokenEOF {
			l.onSyntaxError(fmt.Errorf(
				"unexpected token '%s' (at line %d, column %d)",
				tok.GetText(), tok.GetLine(), tok.GetColumn(),
			))
		}
	}

	return l.getRoot(), l.getError()
}

var _ querylang.Parser = (*parserImpl)(nil)
