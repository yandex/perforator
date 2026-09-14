package parserexp

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	parser "github.com/yandex/perforator/observability/lib/querylang/parser/exp/generated"
)

type expressionListener struct {
	errorListener
	parser.BaseSolomonParserListener

	selectorListener *selectorListener

	stack     []*querylang.Expression
	pipeInput *querylang.Expression
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

	fc := &querylang.FunctionCall{
		Identifier: querylang.Identifier(c.IDENT().GetText()),
	}
	if isPipeRhsCall(c) {
		l.pipeInput = detachExpressionValue(l.top())
	}
	l.top().FunctionCall = fc
}

func (l *expressionListener) ExitCall(c *parser.CallContext) {
	if l.hasErrors() || l.pipeInput == nil {
		return
	}
	if !isPipeRhsCall(c) {
		return
	}

	insertPipeArg(l.top().FunctionCall, l.pipeInput)
	l.pipeInput = nil
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
	if e.FunctionCall != nil {
		return false
	}
	return e.Lambda == nil && e.Selector == nil && e.Identifier == "" && e.Value == nil
}

func isPipeRhsCall(c *parser.CallContext) bool {
	parent, ok := c.GetParent().(antlr.ParserRuleContext)
	if !ok {
		return false
	}
	callWithBy, ok := asCallWithByContext(parent)
	if !ok {
		return false
	}
	_, ok = callWithBy.GetParent().(*parser.ExprPipeContext)
	return ok
}

func asCallWithByContext(ctx antlr.ParserRuleContext) (*parser.CallWithByContext, bool) {
	switch p := ctx.(type) {
	case *parser.CallWithByContext:
		return p, true
	case *parser.AtomCallContext:
		return &p.CallWithByContext, true
	case *parser.AtomCallByDurationContext:
		return &p.CallWithByContext, true
	case *parser.AtomCallByLabelContext:
		return &p.CallWithByContext, true
	case *parser.AtomCallByLabelsContext:
		return &p.CallWithByContext, true
	default:
		return nil, false
	}
}

func detachExpressionValue(e *querylang.Expression) *querylang.Expression {
	piped := &querylang.Expression{}
	switch {
	case e.FunctionCall != nil:
		piped.FunctionCall = e.FunctionCall
		e.FunctionCall = nil
	case e.Selector != nil:
		piped.Selector = e.Selector
		e.Selector = nil
	case e.Identifier != "":
		piped.Identifier = e.Identifier
		e.Identifier = ""
	case e.Lambda != nil:
		piped.Lambda = e.Lambda
		e.Lambda = nil
	case e.Value != nil:
		piped.Value = e.Value
		e.Value = nil
	}
	return piped
}

func insertPipeArg(fc *querylang.FunctionCall, piped *querylang.Expression) {
	fc.Arguments = append(fc.Arguments, nil)
	copy(fc.Arguments[1:], fc.Arguments[0:])
	fc.Arguments[0] = piped
}

func pipeSelectorArgIndex(funcName string, explicitArgCount int) int {
	switch funcName {
	case "alias", "relabel":
		return 0
	default:
		return explicitArgCount
	}
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
