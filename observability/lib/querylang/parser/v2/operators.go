package parserv2

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

type basicOperator interface {
	antlr.ParseTree
	ASSIGNMENT() antlr.TerminalNode // =
	NE() antlr.TerminalNode         // !=
	EQ() antlr.TerminalNode         // ==
	NOT_EQUIV() antlr.TerminalNode  // !==
}

type numericOperator interface {
	basicOperator
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GE() antlr.TerminalNode
	LE() antlr.TerminalNode
}

type stringOperator interface {
	numericOperator
	REGEX() antlr.TerminalNode
	NOT_REGEX() antlr.TerminalNode
}

type durationOperator interface {
	antlr.ParseTree
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GE() antlr.TerminalNode
	LE() antlr.TerminalNode
}

type operatorCond struct {
	operator operator.Operator
	inverse  bool
}

func (c operatorCond) isEq() bool {
	return c.operator == operator.Eq
}

func convertBasicOperator(op basicOperator) (result operatorCond, err error) {
	switch {
	case op.ASSIGNMENT() != nil || op.EQ() != nil:
		result.operator = operator.Eq
	case op.NE() != nil || op.NOT_EQUIV() != nil:
		result.operator = operator.Eq
		result.inverse = true
	default:
		err = fmt.Errorf("unsupported operator: '%s'", op.GetText())
	}
	return
}

func convertNumericOperator(op numericOperator) (result operatorCond, err error) {
	switch {
	case op.GT() != nil:
		result.operator = operator.GT
	case op.LT() != nil:
		result.operator = operator.LT
	case op.GE() != nil:
		result.operator = operator.GTE
	case op.LE() != nil:
		result.operator = operator.LTE
	default:
		return convertBasicOperator(op)
	}
	return
}

func convertStringOperator(op stringOperator) (result operatorCond, err error) {
	switch {
	case op.REGEX() != nil:
		result.operator = operator.Regex
	case op.NOT_REGEX() != nil:
		result.operator = operator.Regex
		result.inverse = true
	default:
		return convertNumericOperator(op)
	}
	return
}

func convertDurationOperator(op durationOperator) (result operatorCond, err error) {
	switch {
	case op.GT() != nil:
		result.operator = operator.GT
	case op.LT() != nil:
		result.operator = operator.LT
	case op.GE() != nil:
		result.operator = operator.GTE
	case op.LE() != nil:
		result.operator = operator.LTE
	default:
		err = fmt.Errorf("unsupported operator: '%s'", op.GetText())
	}
	return
}
