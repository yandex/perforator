package profilequerylang

import (
	"fmt"
	"strings"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

func ValueToString(value querylang.Value) string {
	switch value := value.(type) {
	case querylang.String:
		return value.Value
	case *querylang.String:
		return value.Value
	default:
		return value.Repr()
	}
}

func ConditionToString(field string, condition *querylang.Condition) string {
	return fmt.Sprintf(`"%s"%s"%s"`, field, operator.Repr(condition.Operator, condition.Inverse), ValueToString(condition.Value))
}

func MatcherToString(matcher *querylang.Matcher) (string, error) {
	if len(matcher.Conditions) == 0 {
		return "", nil
	}

	switch matcher.Operator {
	case querylang.OR:
		firstOperator := operator.Repr(matcher.Conditions[0].Operator, matcher.Conditions[0].Inverse)
		sameOperator := true

		for _, condition := range matcher.Conditions {
			if operator.Repr(condition.Operator, condition.Inverse) != firstOperator {
				sameOperator = false
			}
		}

		if !sameOperator {
			return "", fmt.Errorf(
				"cannot use multiple conditions with different operators with logical operator OR for field %s",
				matcher.Field,
			)
		}

		values := make([]string, 0, len(matcher.Conditions))
		for _, condition := range matcher.Conditions {
			values = append(values, ValueToString(condition.Value))
		}

		return fmt.Sprintf(`"%s"%s"%s"`, matcher.Field, firstOperator, strings.Join(values, "|")), nil
	default:
		conditions := make([]string, 0, len(matcher.Conditions))

		for _, condition := range matcher.Conditions {
			conditions = append(conditions, ConditionToString(matcher.Field, condition))
		}

		if len(conditions) > 1 && matcher.Operator != querylang.AND {
			return "", fmt.Errorf("unexpected logical operator for field %s", matcher.Field)
		}

		return strings.Join(conditions, ","), nil
	}
}

func ExtractEqualityMatch(matcher *querylang.Matcher) (string, error) {
	if len(matcher.Conditions) != 1 {
		return "", fmt.Errorf("only one condition is allowed")
	}
	if matcher.Conditions[0].Operator != operator.Eq {
		return "", fmt.Errorf("only operator '=' is supported")
	}
	if matcher.Conditions[0].Inverse {
		return "", fmt.Errorf("'!=' sign is not supported")
	}
	switch v := matcher.Conditions[0].Value.(type) {
	case querylang.String:
		return v.Value, nil
	case *querylang.String:
		return v.Value, nil
	default:
		return "", fmt.Errorf("failed to extract string value from %s", v.Repr())
	}
}

func SelectorToString(selector *querylang.Selector) (string, error) {
	results := make([]string, 0, len(selector.Matchers))

	for _, matcher := range selector.Matchers {
		matcherStr, err := MatcherToString(matcher)
		if err != nil {
			return "", err
		}

		if matcherStr != "" {
			results = append(results, matcherStr)
		}
	}

	return fmt.Sprintf("{%s}", strings.Join(results, ",")), nil
}
