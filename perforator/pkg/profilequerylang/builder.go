package profilequerylang

import (
	"fmt"
	"math/big"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

func BuildValue(value interface{}) (querylang.Value, error) {
	switch value := value.(type) {
	case string:
		return querylang.String{
			Value: value,
		}, nil
	case int:
		return querylang.Int{
			Value: big.NewInt(int64(value)),
		}, nil
	case int64:
		return querylang.Int{
			Value: big.NewInt(value),
		}, nil
	default:
		return nil, fmt.Errorf("type %T cannot be converted to querylang.Value", value)
	}
}

func BuildMatcher[T string | int64 | int](
	field string,
	logicalOperator querylang.LogicalOperator,
	conditionBase querylang.Condition,
	values []T,
) *querylang.Matcher {
	matcher := &querylang.Matcher{
		Field:      field,
		Operator:   logicalOperator,
		Conditions: make([]*querylang.Condition, 0, len(values)),
	}

	for _, value := range values {
		condition := conditionBase
		condition.Value, _ = BuildValue(value)

		matcher.Conditions = append(matcher.Conditions, &condition)
	}
	return matcher
}

// Simple selector builder for *querylang.Selector construction
type Builder struct {
	services         []string
	buildIDs         []string
	nodeIDs          []string
	podIDs           []string
	cpus             []string
	profilerVersions []string
	profileIDs       []string
	timestampFrom    *time.Time
	timestampTo      *time.Time
	clusters         []string
}

func NewBuilder() *Builder {
	return &Builder{
		services:         []string{},
		buildIDs:         []string{},
		nodeIDs:          []string{},
		podIDs:           []string{},
		cpus:             []string{},
		profilerVersions: []string{},
		profileIDs:       []string{},
		clusters:         []string{},
	}
}

func (b *Builder) Services(services ...string) *Builder {
	b.services = append(b.services, services...)
	return b
}

func (b *Builder) BuildIDs(buildIDs ...string) *Builder {
	b.buildIDs = append(b.buildIDs, buildIDs...)
	return b
}

func (b *Builder) Clusters(clusters ...string) *Builder {
	b.clusters = append(b.clusters, clusters...)
	return b
}

func (b *Builder) NodeIDs(nodeIDs ...string) *Builder {
	b.nodeIDs = append(b.nodeIDs, nodeIDs...)
	return b
}

func (b *Builder) PodIDs(podIDs ...string) *Builder {
	b.podIDs = append(b.podIDs, podIDs...)
	return b
}

func (b *Builder) CPUs(cpus ...string) *Builder {
	b.cpus = append(b.cpus, cpus...)
	return b
}

func (b *Builder) ProfilerVersions(versions ...string) *Builder {
	b.profilerVersions = append(b.profilerVersions, versions...)
	return b
}

func (b *Builder) ProfileIDs(ids ...string) *Builder {
	b.profileIDs = append(b.profileIDs, ids...)
	return b
}

func (b *Builder) From(ts time.Time) *Builder {
	b.timestampFrom = &ts
	return b
}

func (b *Builder) To(ts time.Time) *Builder {
	b.timestampTo = &ts
	return b
}

func (b *Builder) AddMatcher(selector *querylang.Selector, label string, values []string) *Builder {
	if len(values) > 0 {
		selector.Matchers = append(
			selector.Matchers,
			BuildMatcher(
				label,
				querylang.OR,
				querylang.Condition{Operator: operator.Eq},
				values,
			),
		)
	}
	return b
}

func (b *Builder) AddTimestampMatcher(
	selector *querylang.Selector,
	oper operator.Operator,
	value *time.Time,
) *Builder {
	if value != nil && !value.IsZero() {
		selector.Matchers = append(
			selector.Matchers,
			BuildMatcher(
				TimestampLabel,
				querylang.AND,
				querylang.Condition{Operator: oper},
				[]string{value.Format(time.RFC3339Nano)},
			),
		)
	}
	return b
}

func (b *Builder) Build() *querylang.Selector {
	selector := &querylang.Selector{
		Matchers: []*querylang.Matcher{},
	}

	b.
		AddMatcher(selector, BuildIDLabel, b.buildIDs).
		AddMatcher(selector, CPULabel, b.cpus).
		AddMatcher(selector, PodIDLabel, b.podIDs).
		AddMatcher(selector, NodeIDLabel, b.nodeIDs).
		AddMatcher(selector, ProfileIDLabel, b.profileIDs).
		AddMatcher(selector, ProfilerVersionLabel, b.profilerVersions).
		AddMatcher(selector, ServiceLabel, b.services).
		AddMatcher(selector, ClusterLabel, b.clusters)

	b.
		AddTimestampMatcher(selector, operator.GTE, b.timestampFrom).
		AddTimestampMatcher(selector, operator.LTE, b.timestampTo)

	return selector
}

func TimeFromProto(timestamp *timestamppb.Timestamp) time.Time {
	if timestamp != nil && timestamp.IsValid() {
		return timestamp.AsTime()
	}
	return time.Time{}
}
