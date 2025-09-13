package querylang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

var (
	errUnknownConditionOperator = fmt.Errorf("unknown condition operator")
	errImpossibleOperatorsForOr = fmt.Errorf("impossible operators for OR")
)

// ToString returns string representation of Selector.
// It should be used for logging and analytics purposes.
// It uses more cpu time than Repr.
// It returns a string depending on data only, not on order.
// It may return error if such a Selector is impossible in the grammar.
func (f *Selector) ToString() (string, error) {
	if f == nil {
		return "{}", nil
	}

	var result []string

	var (
		currentField      string
		currentConditions []string
	)
	flush := func() {
		sort.Strings(currentConditions)
		result = append(result, currentConditions...)
		currentConditions = currentConditions[:0]
	}

	for _, m := range f.Matchers {
		if m.Field != currentField {
			flush()
		}
		cond, err := matcherReprStable(m)
		if err != nil {
			return "", err
		}
		currentConditions = append(currentConditions, cond...)
		currentField = m.Field
	}
	flush()

	return fmt.Sprintf("{%s}", strings.Join(result, ", ")), nil
}

func matcherReprStable(m *Matcher) ([]string, error) {
	if m.Operator == AND || len(m.Conditions) == 1 {
		var conditions []string
		for _, cond := range m.Conditions {
			c, err := fieldWithConditionReprStable(m.Field, cond)
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, c)
		}
		return conditions, nil
	}

	// OR

	if !matcherCouldBeProcessedAsOr(m) {
		return nil, errImpossibleOperatorsForOr
	}

	var values []string

	for _, cond := range m.Conditions {
		v := cond.Value.raw()
		if cond.Operator == operator.Exists {
			if cond.Inverse {
				v = "-"
			} else {
				v = "*"
			}
		}
		values = append(values, v)
	}

	sort.Strings(values)

	valueStr := strings.Join(values, "|")

	field, err := smartquote(m.Field)
	if err != nil {
		return nil, err
	}
	v, err := smartquote(valueStr)
	if err != nil {
		return nil, err
	}

	return []string{fmt.Sprintf("%s = %s", field, v)}, nil
}

func matcherCouldBeProcessedAsOr(m *Matcher) bool {
	for _, c := range m.Conditions {
		if c.Operator == operator.Regex || c.Operator == operator.ISubstring {
			return false
		}
		if c.Operator != operator.Exists && c.Inverse {
			return false
		}
	}

	return true
}

func fieldWithConditionReprStable(field string, cond *Condition) (string, error) {
	field, err := smartquote(field)
	if err != nil {
		return "", err
	}

	condStr, err := conditionReprStable(cond)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`%s %s`, field, condStr), nil
}

func conditionReprStable(cond *Condition) (string, error) {
	var op string
	switch cond.Operator {
	case operator.Exists:
		if cond.Inverse {
			return `= "-"`, nil
		}
		return `= "*"`, nil
	case operator.LT:
		op = "<"
	case operator.LTE:
		op = "<="
	case operator.GT:
		op = ">"
	case operator.GTE:
		op = ">="
	case operator.Regex:
		if cond.Inverse {
			op = "!~"
		} else {
			op = "=~"
		}
	case operator.Glob:
		if cond.Inverse {
			op = "!="
		} else {
			op = "="
		}
	case operator.Eq:
		op = "="
		if cond.Inverse {
			op = "!" + op
		}
		if hasSpecialSymbols(cond.Value.raw()) {
			op += "="
		}
	case operator.ISubstring:
		if cond.Inverse {
			op = "!=*"
		} else {
			op = "=*"
		}
	default:
		return "", errUnknownConditionOperator
	}

	value, err := cond.Value.ToString()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s", op, value), nil
}

func hasSpecialSymbols(value string) bool {
	return strings.ContainsAny(value, "*?|") || value == "-"
}
