package parserv2

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	parser "github.com/yandex/perforator/observability/lib/querylang/parser/v2/generated"
)

type expressionListener struct {
	errorListener
	parser.BaseSolomonParserListener

	selectorListener *selectorListener

	stack []*querylang.Expression
}

func newExpressionListener() *expressionListener {
	l := &expressionListener{}
	l.stack = append(l.stack, &querylang.Expression{
		FunctionCall: &querylang.FunctionCall{},
	})
	return l
}

func (l *expressionListener) getRoot() *querylang.Expression {
	if len(l.stack) != 1 {
		return nil
	}
	args := l.stack[0].FunctionCall.Arguments
	if len(args) != 1 {
		return nil
	}
	return args[0]
}

func (l *expressionListener) VisitErrorNode(node antlr.ErrorNode) {
	l.errorListener.VisitErrorNode(node)
}

// --- Supported expressions ---

func (l *expressionListener) EnterExpression(c *parser.ExpressionContext) {
	if l.hasErrors() {
		return
	}

	arg := &querylang.Expression{}

	if l.top().FunctionCall != nil {
		l.top().FunctionCall.Arguments = append(l.top().FunctionCall.Arguments, arg)
	} else if l.top().Lambda != nil {
		l.top().Lambda.Expression = arg
	}

	l.push(arg)
}

func (l *expressionListener) ExitExpression(c *parser.ExpressionContext) {
	if l.hasErrors() {
		return
	}

	if isZeroExpression(l.top()) {
		l.onSemanticError(fmt.Errorf("empty expression"))
	} else {
		l.pop()
	}
}

func (l *expressionListener) EnterCall(c *parser.CallContext) {
	if l.hasErrors() {
		return
	}

	l.top().FunctionCall = &querylang.FunctionCall{
		Identifier: querylang.Identifier(c.IDENT().GetText()),
	}
}

func (l *expressionListener) EnterSelectors(c *parser.SelectorsContext) {
	if l.hasErrors() {
		return
	}

	l.selectorListener = newSelectorListener()
}

func (l *expressionListener) EnterSelector(c *parser.SelectorContext) {
	if l.hasErrors() {
		return
	}

	l.selectorListener.EnterSelector(c)
}

func (l *expressionListener) ExitSelectors(c *parser.SelectorsContext) {
	if l.hasErrors() {
		return
	}

	if l.selectorListener.hasErrors() {
		l.parseErrors = l.selectorListener.parseErrors
		return
	}

	l.top().Selector = l.selectorListener.root
	l.selectorListener = nil
}

func (l *expressionListener) EnterLambda(c *parser.LambdaContext) {
	if l.hasErrors() {
		return
	}

	l.top().Lambda = &querylang.Lambda{}
	if c.IDENT() != nil {
		l.top().Lambda.Arguments = append(l.top().Lambda.Arguments, querylang.Identifier(c.IDENT().GetText()))
	} else {
		for _, id := range c.Arglist().AllIDENT() {
			l.top().Lambda.Arguments = append(l.top().Lambda.Arguments, querylang.Identifier(id.GetText()))
		}
	}
}

func (l *expressionListener) EnterAtomNumber(c *parser.AtomNumberContext) {
	if l.hasErrors() {
		return
	}

	value, err := convertNumber(c.NUMBER().GetText())
	if err != nil {
		l.onSyntaxError(err)
		return
	}
	l.top().Value = value
}

func (l *expressionListener) EnterAtomString(c *parser.AtomStringContext) {
	if l.hasErrors() {
		return
	}

	l.top().Value = querylang.String{Value: unquote(c.STRING().GetText())}
}

func (l *expressionListener) EnterAtomIdent(c *parser.AtomIdentContext) {
	if l.hasErrors() {
		return
	}

	l.top().Identifier = querylang.Identifier(c.IDENT().GetText())
}

// --- Helpers ---

func (l *expressionListener) pop() {
	l.stack = l.stack[:len(l.stack)-1]
}

func (l *expressionListener) push(e *querylang.Expression) {
	l.stack = append(l.stack, e)
}

func (l *expressionListener) top() *querylang.Expression {
	if len(l.stack) > 0 {
		return l.stack[len(l.stack)-1]
	}
	return nil
}

func isZeroExpression(e *querylang.Expression) bool {
	if e == nil {
		return true
	}
	return e.FunctionCall == nil && e.Lambda == nil && e.Selector == nil && e.Identifier == "" && e.Value == nil
}

// --- Ensure no operators on expressions are used ---

func (l *expressionListener) EnterExprUnary(c *parser.ExprUnaryContext) {
	if c.MINUS() != nil || c.PLUS() != nil {
		l.onSemanticError(fmt.Errorf("unexpected arithmetic expression"))
	}
}

func (l *expressionListener) EnterExprTerm(c *parser.ExprTermContext) {
	if len(c.AllExprUnary()) > 1 {
		l.onSemanticError(fmt.Errorf("unexpected arithmetic expression"))
	}
}

func (l *expressionListener) EnterExprArith(c *parser.ExprArithContext) {
	if len(c.AllExprTerm()) > 1 {
		l.onSemanticError(fmt.Errorf("unexpected arithmetic expression"))
	}
}

func (l *expressionListener) EnterExprComp(c *parser.ExprCompContext) {
	if len(c.AllExprArith()) > 1 {
		l.onSemanticError(fmt.Errorf("unexpected comparison expression"))
	}
}

func (l *expressionListener) EnterExprNot(c *parser.ExprNotContext) {
	if c.NOT() != nil {
		l.onSemanticError(fmt.Errorf("unexpected logical expression"))
	}
}

func (l *expressionListener) EnterExprAnd(c *parser.ExprAndContext) {
	if len(c.AllAND()) > 0 {
		l.onSemanticError(fmt.Errorf("unexpected logical expression"))
	}
}

func (l *expressionListener) EnterExprOr(c *parser.ExprOrContext) {
	if len(c.AllOR()) > 0 {
		l.onSemanticError(fmt.Errorf("unexpected logical expression"))
	}
}

// --- Unsupported yet expressions ---

func (l *expressionListener) EnterAtomVector(c *parser.AtomVectorContext) {
	l.onSemanticError(fmt.Errorf("unexpected vector value"))
}

func (l *expressionListener) EnterAtomDuration(c *parser.AtomDurationContext) {
	l.onSemanticError(fmt.Errorf("unexpected duration value"))
}

func (l *expressionListener) EnterAtomCallByDuration(c *parser.AtomCallByDurationContext) {
	l.onSemanticError(fmt.Errorf("unexpected call by duration"))
}

func (l *expressionListener) EnterAtomCallByLabel(c *parser.AtomCallByLabelContext) {
	l.onSemanticError(fmt.Errorf("unexpected call by label"))
}

func (l *expressionListener) EnterAtomCallByLabels(c *parser.AtomCallByLabelsContext) {
	l.onSemanticError(fmt.Errorf("unexpected call by labels"))
}
