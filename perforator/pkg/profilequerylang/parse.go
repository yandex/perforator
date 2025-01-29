package profilequerylang

import (
	"errors"
	"fmt"
	"time"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	parserv2 "github.com/yandex/perforator/observability/lib/querylang/parser/v2"
	"github.com/yandex/perforator/perforator/pkg/humantime"
)

func ValueRepr(value querylang.Value) string {
	switch value := value.(type) {
	case querylang.String:
		return value.Value
	case *querylang.String:
		return value.Value
	default:
		return value.Repr()
	}
}

type TimeInterval struct {
	From *time.Time
	To   *time.Time
}

func ParseTimeInterval(selector *querylang.Selector) (*TimeInterval, error) {
	interval := TimeInterval{}

	for _, matcher := range selector.Matchers {
		if matcher.Field != TimestampLabel {
			continue
		}

		if matcher.Operator == querylang.OR && len(matcher.Conditions) > 1 {
			return nil, errors.New("unexpected OR matcher for timestamp label")
		}

		for _, condition := range matcher.Conditions {
			repr := ValueRepr(condition.Value)
			ts, err := humantime.Parse(repr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse one of timestamps values %s: %w", repr, err)
			}

			switch condition.Operator {
			case operator.LT, operator.LTE:
				if interval.To == nil || interval.To.After(ts) {
					interval.To = &ts
				}
			case operator.GT, operator.GTE:
				if interval.From == nil || interval.From.Before(ts) {
					interval.From = &ts
				}
			}
		}
	}

	return &interval, nil
}

func ParseSelector(selector string) (*querylang.Selector, error) {
	parser := parserv2.NewParser()
	return parser.ParseSelector(selector)
}
