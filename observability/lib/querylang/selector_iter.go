package querylang

import (
	"fmt"
	"slices"
	"strings"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

func (f *Selector) Clone() *Selector {
	matchers := make([]*Matcher, 0, len(f.Matchers))
	for _, m := range f.Matchers {
		conditions := make([]*Condition, 0, len(m.Conditions))
		for _, c := range m.Conditions {
			conditions = append(conditions, &Condition{
				Operator: c.Operator,
				Inverse:  c.Inverse,
				Value:    c.Value.clone(),
			})
		}
		matchers = append(matchers, &Matcher{
			Field:      m.Field,
			Operator:   m.Operator,
			Conditions: conditions,
		})
	}
	return &Selector{
		Matchers: matchers,
	}
}

func (f *Selector) SortMatchers(highPriorityFields ...string) {
	slices.SortFunc(f.Matchers, func(a, b *Matcher) int {
		ai := slices.Index(highPriorityFields, a.Field)
		bi := slices.Index(highPriorityFields, b.Field)

		if ai != -1 && bi != -1 {
			return ai - bi
		} else if ai != -1 {
			return -1
		} else if bi != -1 {
			return 1
		}
		return strings.Compare(a.Field, b.Field)
	})
}

func (f *Selector) IsEmpty() bool {
	return len(f.Matchers) == 0
}

func (f *Selector) AllMentionedFields() []string {
	fields := make(map[string]struct{})
	for _, m := range f.Matchers {
		fields[m.Field] = struct{}{}
	}
	return mapKeys(fields)
}

func (f *Selector) UniqueFieldValues(field string) []Value {
	values := make(map[Value]struct{})
	for _, m := range f.Matchers {
		if m.Field == field {
			for _, cond := range m.Conditions {
				values[cond.Value] = struct{}{}
			}
		}
	}
	return mapKeys(values)
}

// StrictMap returns a field-value mapping of Selector
// if there are only strict equality operators are used
// and any field corresponds to exactly 1 string value.
func (f *Selector) StrictMap() (map[string]string, error) {
	result := make(map[string]string, len(f.Matchers))
	for _, m := range f.Matchers {
		for _, c := range m.Conditions {
			if !c.IsStrictEq() {
				return nil, fmt.Errorf("field `%s` is involved in non-strict-equality comparison", m.Field)
			}
			if _, present := result[m.Field]; present {
				return nil, fmt.Errorf("found multiple values correspond to field `%s`", m.Field)
			}
			if sv, ok := c.Value.(String); !ok {
				return nil, fmt.Errorf("found non-string literal comparison with field `%s`", m.Field)
			} else {
				result[m.Field] = sv.Value
			}
		}
	}
	return result, nil
}

func (f *Selector) ReplaceConditionValue(field string, oldValue Value, newValues []Value) {
	for _, matcher := range f.Matchers {
		if matcher.Field == field {
			exists := hasCondition(matcher.Conditions, func(item *Condition) bool {
				return item.Value == oldValue
			})

			if !exists {
				continue
			}

			var newConditions []*Condition
			for _, condition := range matcher.Conditions {
				if condition.Value == oldValue {
					for _, newValue := range newValues {
						newConditions = append(newConditions, &Condition{
							Operator: condition.Operator,
							Inverse:  condition.Inverse,
							Value:    newValue,
						})
					}
				}
			}

			matcher.Conditions = filterConditions(matcher.Conditions, func(item *Condition, _ int) bool {
				return item.Value != oldValue
			})

			matcher.Conditions = append(matcher.Conditions, newConditions...)
		}
	}
}

func (f *Selector) RemoveFieldsMatchers(fields ...string) {
	filtered := make([]*Matcher, 0, len(f.Matchers))
	for _, m := range f.Matchers {
		if !slices.Contains(fields, m.Field) {
			filtered = append(filtered, m)
		}
	}
	f.Matchers = filtered
}

func (f *Selector) AddMatchers(matchers ...*Matcher) {
	f.Matchers = append(f.Matchers, matchers...)
}

// CandidateValues returns a superset of values for each field in Selector.
// If result[field] is nil then the function failed to find a superset of
// values.
// If result[field] is empty slice, then the answer is empty set.
// If result[field] not exists, then there's no such field in Selector.
func (f *Selector) CandidateValues() map[string][]Value {
	result := make(map[string][]Value)

	// Find starting set
	for _, m := range f.Matchers {
		if len(result[m.Field]) != 0 {
			continue
		}
		result[m.Field] = extractValuesFromMatcher(m)
	}

	// Filter values
	for _, m := range f.Matchers {
		if len(result[m.Field]) == 0 {
			continue
		}
		filtered := make([]Value, 0, len(result[m.Field]))
		for _, v := range result[m.Field] {
			if matches(v, m) {
				filtered = append(filtered, v)
			}
		}
		result[m.Field] = filtered
	}

	return result
}

func matches(value Value, m *Matcher) bool {
	processed := 0
	matched := 0
	notMatched := 0
	for _, cond := range m.Conditions {
		if cond.Operator != operator.Eq {
			continue
		}
		processed++
		isMatching := value.Repr() == cond.Value.Repr()
		if cond.Inverse {
			isMatching = !isMatching
		}
		if isMatching {
			matched++
		} else {
			notMatched++
		}
	}

	switch m.Operator {
	case AND:
		return notMatched == 0
	case OR:
		return matched > 0 || processed == 0
	}
	return false
}

func extractValuesFromMatcher(m *Matcher) (candidates []Value) {
	for _, cond := range m.Conditions {
		if !cond.IsStrictEq() {
			return nil
		}
		candidates = append(candidates, cond.Value)
	}

	return
}
