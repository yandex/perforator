package parserv2

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	parser "github.com/yandex/perforator/observability/lib/querylang/parser/v2/generated"
)

type listener struct {
	parser.BaseSolomonSelectorParserListener
	antlr.DefaultErrorListener

	root           *querylang.Selector
	syntaxErrors   []error
	semanticErrors []error
}

func newListener() *listener {
	var root querylang.Selector
	l := &listener{
		root: &root,
	}
	return l
}

var _ parser.SolomonSelectorParserListener = (*listener)(nil)

func (l *listener) hasErrors() bool {
	return len(l.syntaxErrors)+len(l.semanticErrors) > 0
}

func (l *listener) Err() error {
	if len(l.syntaxErrors) > 0 {
		return fmt.Errorf("syntax errors: %s", errors.Join(l.syntaxErrors...))
	}
	if len(l.semanticErrors) > 0 {
		return fmt.Errorf("semantic errors: %s", errors.Join(l.semanticErrors...))
	}
	return nil
}

func (l *listener) onSyntaxError(err error) {
	l.syntaxErrors = append(l.syntaxErrors, err)
}

func (l *listener) onSemanticError(err error) {
	l.semanticErrors = append(l.semanticErrors, err)
}

func (l *listener) currentMatcher() *querylang.Matcher {
	return l.root.Matchers[len(l.root.Matchers)-1]
}

func (l *listener) appendCondition(cond *querylang.Condition) {
	l.currentMatcher().Conditions = append(l.currentMatcher().Conditions, cond)
}

func (l *listener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	l.onSyntaxError(fmt.Errorf("%s (at line %d, column %d)", msg, line, column))
}

func (l *listener) VisitErrorNode(node antlr.ErrorNode) {
	if l.hasErrors() {
		return
	}
	if parent, ok := node.GetParent().GetPayload().(antlr.ParseTree); ok {
		l.onSyntaxError(fmt.Errorf(
			"syntax error at '%s', in token '%s'",
			parent.GetText(),
			node.GetText(),
		))
	} else {
		l.onSyntaxError(fmt.Errorf(
			"syntax error at '%s'",
			node.GetText(),
		))
	}
}

func (l *listener) EnterSelector(c *parser.SelectorContext) {
	if l.hasErrors() {
		return
	}

	left := c.SelectorLeftOperand().GetText()

	matcher := querylang.Matcher{
		Field:    unquote(left),
		Operator: evalMatcherOperator(c),
	}
	l.root.Matchers = append(l.root.Matchers, &matcher)

	switch {
	case c.SelectorOpString() != nil:
		right := firstNotNilTerminal(c.IDENT_WITH_DOTS(), c.IdentOrString())
		l.handleStringSelector(c.SelectorOpString(), right)
	case c.SelectorOpNumber() != nil:
		right := firstNotNilTerminal(c.NumberUnary())
		l.handleNumericSelector(c.SelectorOpNumber(), right)
	case c.SelectorOpDuration() != nil:
		l.handleDurationSelector(c.SelectorOpDuration(), c.DURATION().GetText())
	case c.ASSIGNMENT() != nil:
		l.handleNotExists()
	}
}

func evalMatcherOperator(c *parser.SelectorContext) querylang.LogicalOperator {
	if c.SelectorOpString() != nil {
		op, err := convertStringOperator(c.SelectorOpString())
		if err == nil && op.inverse {
			return querylang.AND
		}
	}
	if c.SelectorOpNumber() != nil {
		op, err := convertNumericOperator(c.SelectorOpNumber())
		if err == nil && op.inverse {
			return querylang.AND
		}
	}
	return querylang.OR
}

func firstNotNilTerminal(terms ...antlr.ParseTree) string {
	for _, t := range terms {
		if t != nil {
			return t.GetText()
		}
	}
	return ""
}

func unquote(s string) string {
	const singleQuote = "'"
	const doubleQuote = "\""
	if strings.HasPrefix(s, singleQuote) && strings.HasSuffix(s, singleQuote) {
		return strings.Trim(s, singleQuote)
	}
	if strings.HasPrefix(s, doubleQuote) && strings.HasSuffix(s, doubleQuote) {
		return strings.Trim(s, doubleQuote)
	}
	return s
}

var numberSuffixes = map[string]int64{"k": 1e3, "M": 1e6, "G": 1e9, "T": 1e12, "P": 1e15, "E": 1e18}

func findSuffix(v string) string {
	for suffix := range numberSuffixes {
		if strings.HasSuffix(v, suffix) {
			return suffix
		}
	}
	return ""
}

func convertNumber(v string) (querylang.Value, error) {
	suffix := findSuffix(v)
	v = strings.TrimSuffix(v, suffix)
	mul := numberSuffixes[suffix]
	if suffix == "" {
		mul = 1
	}

	var bigInt big.Int
	if err := bigInt.UnmarshalText([]byte(v)); err == nil {
		bigInt.Mul(&bigInt, big.NewInt(mul))
		return querylang.Int{Value: &bigInt}, nil
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return querylang.Empty{}, err
	}
	f *= float64(mul)
	return querylang.Float{Value: f}, nil
}

func (l *listener) handleStringSelector(opCtx parser.ISelectorOpStringContext, right string) {
	if opCtx.EQ() != nil || opCtx.NOT_EQUIV() != nil {
		l.handleStringExactSelector(opCtx, right)
		return
	}

	right = unquote(right)
	rightVariants := strings.Split(right, "|")

	op, err := convertStringOperator(opCtx)
	if err != nil {
		l.onSemanticError(err)
		return
	}

	newCond := func() *querylang.Condition {
		var cond querylang.Condition
		cond.Operator = op.operator
		cond.Inverse = op.inverse
		cond.Value = querylang.Empty{}
		return &cond
	}

	for _, v := range rightVariants {
		cond := newCond()
		cond.Value = querylang.String{Value: strings.ReplaceAll(v, "\\n", "\n")}

		if op.isEq() {
			if v == "-" {
				cond.Operator = operator.Exists
				cond.Inverse = !cond.Inverse
				cond.Value = querylang.Empty{}
			} else if v == "*" {
				cond.Operator = operator.Exists
				cond.Value = querylang.Empty{}
			} else if isGlobPatternRegex.MatchString(v) {
				cond.Operator = operator.Glob
			}
		}
		l.appendCondition(cond)
	}
}

func (l *listener) handleStringExactSelector(opCtx parser.ISelectorOpStringContext, right string) {
	l.appendCondition(&querylang.Condition{
		Operator: operator.Eq,
		Inverse:  opCtx.NOT_EQUIV() != nil,
		Value:    querylang.String{Value: unquote(right)},
	})
}

func (l *listener) handleNumericSelector(opCtx parser.ISelectorOpNumberContext, right string) {
	op, err := convertNumericOperator(opCtx)
	if err != nil {
		l.onSemanticError(err)
		return
	}

	value, err := convertNumber(unquote(right))
	if err != nil {
		l.onSyntaxError(err)
		return
	}

	l.appendCondition(&querylang.Condition{
		Operator: op.operator,
		Inverse:  op.inverse,
		Value:    value,
	})
}

func (l *listener) handleNotExists() {
	l.appendCondition(&querylang.Condition{
		Operator: operator.Exists,
		Inverse:  true,
		Value:    querylang.Empty{},
	})
}

func (l *listener) handleDurationSelector(opCtx parser.ISelectorOpDurationContext, right string) {
	op, err := convertDurationOperator(opCtx)
	if err != nil {
		l.onSemanticError(err)
		return
	}

	d, err := ParseSolomonDuration(right)
	if err != nil {
		l.onSyntaxError(err)
		return
	}
	l.appendCondition(&querylang.Condition{
		Operator: op.operator,
		Inverse:  op.inverse,
		Value:    querylang.Duration{Value: d},
	})
}

var isGlobPatternRegex = regexp.MustCompile(`(?m)(?:[^\\]|^)[*?]`)
