package profilequerylang

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

const (
	exampleTimestampRFC3339 = "2023-02-03T15:04:05Z"
)

func TestSelector_ParseTimestampComparisons(t *testing.T) {
	selectorString := `{ts>"2024-03-15", ts<="now"}`

	selector, err := ParseSelector(selectorString)
	require.NoError(t, err)

	lowerBound := false
	upperBound := true

	for _, matcher := range selector.Matchers {
		if matcher.Field == "ts" {
			for _, condition := range matcher.Conditions {
				if condition.Operator == operator.LTE {
					upperBound = true
				} else if condition.Operator == operator.GT {
					lowerBound = true
				} else {
					t.Fatalf("unexpected operator %v", condition.Operator)
				}
			}
		}
	}

	require.True(t, lowerBound)
	require.True(t, upperBound)
}

func TestSelector_DumpSelector(t *testing.T) {
	tests := map[*querylang.Selector]string{
		&querylang.Selector{
			Matchers: []*querylang.Matcher{
				{
					Field: "service",
					Conditions: []*querylang.Condition{
						{
							Operator: operator.Eq,
							Value:    querylang.String{Value: "perforator"},
						},
					},
				},
				{
					Field:    "build_ids",
					Operator: querylang.OR,
					Conditions: []*querylang.Condition{
						{
							Operator: operator.Eq,
							Value:    querylang.String{Value: "a"},
						},
						{
							Operator: operator.Eq,
							Value:    querylang.String{Value: "b"},
						},
					},
				},
				{
					Field:    "ts",
					Operator: querylang.AND,
					Conditions: []*querylang.Condition{
						{
							Operator: operator.LTE,
							Value:    querylang.String{Value: "now"},
						},
						{
							Operator: operator.GT,
							Value:    querylang.String{Value: "2024-03-15"},
						},
					},
				},
			},
		}: `{"service"="perforator","build_ids"="a|b","ts"<="now","ts">"2024-03-15"}`,
		&querylang.Selector{
			Matchers: []*querylang.Matcher{},
		}: `{}`,
	}

	for selector, expectedDump := range tests {
		t.Run(expectedDump, func(t *testing.T) {
			selectorString, err := SelectorToString(selector)
			require.NoError(t, err)
			require.Equal(t, expectedDump, selectorString)
		})
	}
}

func TestSelector_DumpErrors(t *testing.T) {
	tests := map[*querylang.Selector]string{
		&querylang.Selector{
			Matchers: []*querylang.Matcher{
				{
					Field:    "service",
					Operator: querylang.OR,
					Conditions: []*querylang.Condition{
						{
							Operator: operator.Eq,
							Inverse:  true,
							Value:    querylang.String{Value: "perforator"},
						},
						{
							Operator: operator.Eq,
							Value:    querylang.String{Value: "web-search"},
						},
					},
				},
			},
		}: "multiple comparison operators for OR",
		&querylang.Selector{
			Matchers: []*querylang.Matcher{
				{
					Field:    "service",
					Operator: querylang.LogicalOperator(-1),
					Conditions: []*querylang.Condition{
						{
							Operator: operator.Eq,
							Inverse:  true,
							Value:    querylang.String{Value: "perforator"},
						},
						{
							Operator: operator.Eq,
							Value:    querylang.String{Value: "web-search"},
						},
					},
				},
			},
		}: "unknown logical operator",
	}

	for selector, description := range tests {
		t.Run(description, func(t *testing.T) {
			_, err := SelectorToString(selector)
			require.Error(t, err)
		})
	}
}

func TestSelector_ParseTimeInterval(t *testing.T) {
	tests := map[*querylang.Selector]*TimeInterval{
		&querylang.Selector{
			Matchers: []*querylang.Matcher{
				{
					Field:    TimestampLabel,
					Operator: querylang.AND,
					Conditions: []*querylang.Condition{
						{
							Operator: operator.LT,
							Value:    querylang.String{Value: "1717426513"},
						},
						{
							Operator: operator.GT,
							Value:    querylang.String{Value: "1717426431"},
						},
						{
							Operator: operator.GTE,
							Value:    querylang.String{Value: "1717426436"},
						},
						{
							Operator: operator.LTE,
							Value:    querylang.String{Value: "1717426517"},
						},
					},
				},
				{
					Field:    TimestampLabel,
					Operator: querylang.AND,
					Conditions: []*querylang.Condition{
						{
							Operator: operator.LT,
							Value:    querylang.String{Value: "1717426520"},
						},
					},
				},
			},
		}: &TimeInterval{
			From: ptr.Time(time.Unix(1717426436, 0)),
			To:   ptr.Time(time.Unix(1717426513, 0)),
		},
		&querylang.Selector{
			Matchers: []*querylang.Matcher{
				{
					Field:    TimestampLabel,
					Operator: querylang.AND,
					Conditions: []*querylang.Condition{

						{
							Operator: operator.GT,
							Value:    querylang.String{Value: "1717426431"},
						},
						{
							Operator: operator.GTE,
							Value:    querylang.String{Value: "1717426436"},
						},
					},
				},
			},
		}: &TimeInterval{
			From: ptr.Time(time.Unix(1717426436, 0)),
		},
	}

	for selector, expectedInterval := range tests {
		t.Run(selector.Repr(), func(t *testing.T) {
			interval, err := ParseTimeInterval(selector)
			require.NoError(t, err)
			require.Equal(t, expectedInterval.From, interval.From)
			require.Equal(t, expectedInterval.To, interval.To)
		})
	}
}
