package parserv2

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	parser "github.com/yandex/perforator/observability/lib/querylang/parser/v2/generated"
)

type selectorListener struct {
	errorListener
	parser.BaseSolomonParserListener

	root *querylang.Selector
}

func newSelectorListener() *selectorListener {
	var root querylang.Selector
	l := &selectorListener{
		root: &root,
	}
	return l
}

var _ parser.SolomonParserListener = (*selectorListener)(nil)

func (l *selectorListener) currentMatcher() *querylang.Matcher {
	return l.root.Matchers[len(l.root.Matchers)-1]
}

func (l *selectorListener) appendCondition(cond *querylang.Condition) {
	l.currentMatcher().Conditions = append(l.currentMatcher().Conditions, cond)
}

func (l *selectorListener) VisitErrorNode(node antlr.ErrorNode) {
	l.errorListener.VisitErrorNode(node)
}

func (l *selectorListener) EnterSelector(c *parser.SelectorContext) {
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

func (l *selectorListener) handleStringSelector(opCtx parser.ISelectorOpStringContext, right string) {
	if opCtx.EQ() != nil || opCtx.NOT_EQUIV() != nil {
		l.handleStringExactSelector(opCtx, right)
		return
	}

	if opCtx.REGEX() != nil || opCtx.NOT_REGEX() != nil {
		l.handleStringRegexSelector(opCtx, right)
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

func (l *selectorListener) handleStringExactSelector(opCtx parser.ISelectorOpStringContext, right string) {
	l.appendCondition(&querylang.Condition{
		Operator: operator.Eq,
		Inverse:  opCtx.NOT_EQUIV() != nil,
		Value:    querylang.String{Value: unquote(right)},
	})
}

func (l *selectorListener) handleStringRegexSelector(opCtx parser.ISelectorOpStringContext, right string) {
	l.appendCondition(&querylang.Condition{
		Operator: operator.Regex,
		Inverse:  opCtx.NOT_REGEX() != nil,
		Value:    querylang.String{Value: unquote(right)},
	})
}

func (l *selectorListener) handleNumericSelector(opCtx parser.ISelectorOpNumberContext, right string) {
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

func (l *selectorListener) handleNotExists() {
	l.appendCondition(&querylang.Condition{
		Operator: operator.Exists,
		Inverse:  true,
		Value:    querylang.Empty{},
	})
}

func (l *selectorListener) handleDurationSelector(opCtx parser.ISelectorOpDurationContext, right string) {
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
