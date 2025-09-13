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
	ToString() (string, error)

	raw() string
	clone() Value
}

type Empty struct {
}

type String struct {
	Value string
}

type Int struct {
	Value *big.Int
}

type Float struct {
	Value float64
}

type Duration struct {
	Value time.Duration
}

func (v Empty) Repr() string {
	return "empty_value"
}

func (v Empty) ToString() (string, error) {
	return v.raw(), nil
}

func (v Empty) raw() string {
	return ""
}

func (v Empty) clone() Value {
	return v
}

func (v String) Repr() string {
	return strconv.Quote(v.raw())
}

func (v String) ToString() (string, error) {
	return smartquote(v.raw())
}

func (v String) raw() string {
	return v.Value
}

func (v String) clone() Value {
	return String{Value: v.Value}
}

func (v Int) Repr() string {
	return v.raw()
}

func (v Int) ToString() (string, error) {
	return v.raw(), nil
}

func (v Int) raw() string {
	return v.Value.Text(10)
}

func (v Int) clone() Value {
	return Int{Value: new(big.Int).Set(v.Value)}
}

func (v Float) Repr() string {
	return v.raw()
}

func (v Float) ToString() (string, error) {
	return v.raw(), nil
}

func (v Float) raw() string {
	// TODO: distinguish int and float repr.
	return strconv.FormatFloat(v.Value, 'g', 15, 64)
}

func (v Float) clone() Value {
	return Float{Value: v.Value}
}

func (v Duration) Repr() string {
	return v.raw()
}

func (v Duration) ToString() (string, error) {
	return v.raw(), nil
}

func (v Duration) raw() string {
	return v.Value.String()
}

func (v Duration) clone() Value {
	return Duration{Value: v.Value}
}
