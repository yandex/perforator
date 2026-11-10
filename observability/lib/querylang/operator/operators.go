package operator

import "github.com/yandex/perforator/observability/lib/querylang/condition"

// Operator is a compatibility alias for condition.ConditionType.
// Prefer importing observability/lib/querylang/condition in new code.
type Operator = condition.ConditionType

const (
	Eq         = condition.Eq
	LT         = condition.LT
	LTE        = condition.LTE
	GT         = condition.GT
	GTE        = condition.GTE
	Regex      = condition.Regex
	Exists     = condition.Exists
	Glob       = condition.Glob
	ISubstring = condition.ISubstring
	Grep       = condition.Grep
)

func Repr(op Operator, inverse bool) string {
	return condition.Repr(op, inverse)
}
