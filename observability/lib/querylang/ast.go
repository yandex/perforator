package querylang

import (
	"math/big"
	"strconv"
	"time"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

type LogicalOperator int

const (
	AND LogicalOperator = iota
	OR
)

// Selector represents Solomon language "selectors", e.g. {x = "a|*", y != "b|c", z =~ ".*"}.
// Logically, Selector is just a conjunction of multiple expressions, here called matchers.
// However, since the right side of a matcher may have multiple values,
// We must have 2 levels of nesting: Matcher and Condition.
type Selector struct {
	Matchers []*Matcher
}

// Matcher represents one key-operator-values expression.
// Operator is usually an OR, if the expression operator is not inverse (e.g. x = "a|*"),
// and it is usually an AND if the expression operator is inverse (e.g. y != "b|c").
type Matcher struct {
	Field      string
	Operator   LogicalOperator
	Conditions []*Condition
}

type Condition struct {
	Operator operator.Operator
	Inverse  bool
	Value    Value
}

type Value interface {
	Repr() string

	// To seal the set of implementors.
	unexported()
}

type Empty struct {
	valueStub
}

type String struct {
	Value string
	valueStub
}

type Int struct {
	Value *big.Int
	valueStub
}

type Float struct {
	Value float64
	valueStub
}

type Duration struct {
	Value time.Duration
	valueStub
}

func (v Empty) Repr() string {
	return "empty_value"
}

func (v String) Repr() string {
	return strconv.Quote(v.Value)
}

func (v Int) Repr() string {
	return v.Value.Text(10)
}

func (v Float) Repr() string {
	// TODO: distinguish int and float repr.
	return strconv.FormatFloat(v.Value, 'g', 15, 64)
}

func (v Duration) Repr() string {
	return v.Value.String()
}

type valueStub struct{}

func (v valueStub) unexported() {}
