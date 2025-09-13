package querylang

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

// Repr returns string representation of Selector.
// It should only be used for tests and debug purposes.
func (f *Selector) Repr() string {
	if f == nil {
		return "nil_selector"
	}

	var parts []string
	for _, m := range f.Matchers {
		parts = append(parts, matcherRepr(m))
	}

	return strings.Join(parts, " AND ")
}

func matcherRepr(m *Matcher) string {
	var conditions []string
	for _, cond := range m.Conditions {
		conditions = append(conditions, conditionRepr(m.Field, cond))
	}

	if m.Operator == AND {
		return strings.Join(conditions, " AND ")
	}
	repr := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		return "(" + repr + ")"
	}
	return repr
}

func conditionRepr(field string, cond *Condition) string {
	if cond.Operator == operator.Exists {
		return fmt.Sprintf(`%s %s`, strconv.Quote(field), operator.Repr(cond.Operator, cond.Inverse))
	}

	value := cond.Value.Repr()
	return fmt.Sprintf(`%s %s %s`, strconv.Quote(field), operator.Repr(cond.Operator, cond.Inverse), value)
}
