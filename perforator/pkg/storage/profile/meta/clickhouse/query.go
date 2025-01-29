package clickhouse

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/humantime"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sqlbuilder"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/tls"
)

const (
	// https://github.com/clickhouse/clickhouse/issues/33592#issuecomment-1013620382
	MinimalAllowedFilteringTimestamp = 1000000
)

var (
	AllColumns string = ""
)

func generateAllColumns(row interface{}) string {
	t := reflect.TypeOf(row)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag, ok := field.Tag.Lookup("ch"); ok {
			columns = append(columns, tag)
		}
	}

	return strings.Join(columns, ", ")
}

func init() {
	AllColumns = generateAllColumns(ProfileRow{})
}

var (
	// multiple columns may correspond to single label if slow migration is in progress (one column migrates to another)
	labelsToColumns = map[string][]string{
		profilequerylang.CPULabel:             []string{"attributes['cpu']"},
		profilequerylang.ProfilerVersionLabel: []string{"attributes['profiler_version']"},
		profilequerylang.BuildIDLabel:         []string{"build_ids"},
		profilequerylang.ServiceLabel:         []string{"service"},
		profilequerylang.PodIDLabel:           []string{"pod_id"},
		profilequerylang.NodeIDLabel:          []string{"node_id"},
		profilequerylang.ProfileIDLabel:       []string{"id"},
		profilequerylang.EventTypeLabel:       []string{"event_type"},
		profilequerylang.SystemNameLabel:      []string{"system_name"},
		profilequerylang.TimestampLabel:       []string{"timestamp"},
		profilequerylang.ClusterLabel:         []string{"cluster"},
	}

	arrayColumns = map[string]bool{
		"build_ids": true,
	}

	nonStringColumns = map[string]bool{
		"build_ids": true,
		"id":        true,
		"timestamp": true,
	}
)

func getTimestampFraction(ts time.Time) float64 {
	return float64(ts.UnixNano()) / 1e9
}

func buildTimestampValueRepr(value querylang.Value) (string, error) {
	tsFraction := float64(0)

	switch value := value.(type) {
	case querylang.String:
		ts, err := humantime.Parse(value.Value)
		if err != nil {
			return "", err
		}
		tsFraction = getTimestampFraction(ts)
	case querylang.Int:
		tsFraction = getTimestampFraction(time.Unix(0, value.Value.Int64()))
	default:
		return "", errors.New("unrecognized querylang.Value type for timestamp field")
	}

	if tsFraction < float64(MinimalAllowedFilteringTimestamp) {
		tsFraction = float64(MinimalAllowedFilteringTimestamp)
	}

	return fmt.Sprintf("%.3f", tsFraction), nil
}

func buildValueRepr(field string, value querylang.Value) (string, error) {
	if field == "timestamp" {
		return buildTimestampValueRepr(value)
	}

	switch value := value.(type) {
	case querylang.String:
		return fmt.Sprintf("'%s'", sqlbuilder.Escape(value.Value)), nil
	case querylang.Int:
		return value.Value.String(), nil
	default:
		return value.Repr(), nil
	}
}

func buildConditionString(column string, condition *querylang.Condition) (string, error) {
	prefix := ""
	if condition.Inverse {
		prefix = "NOT "
	}

	valueRepr, err := buildValueRepr(column, condition.Value)
	if err != nil {
		return "", fmt.Errorf("failed to build value repr: %w", err)
	}

	switch condition.Operator {
	case operator.Eq:
		return fmt.Sprintf("%s%s = %s", prefix, column, valueRepr), nil
	case operator.Regex:
		return fmt.Sprintf("%smatch(%s, %s)", prefix, column, valueRepr), nil
	case operator.LTE:
		return fmt.Sprintf("%s%s <= %s", prefix, column, valueRepr), nil
	case operator.LT:
		return fmt.Sprintf("%s%s < %s", prefix, column, valueRepr), nil
	case operator.GTE:
		return fmt.Sprintf("%s%s >= %s", prefix, column, valueRepr), nil
	case operator.GT:
		return fmt.Sprintf("%s%s > %s", prefix, column, valueRepr), nil
	default:
		return "", fmt.Errorf("querylang operator %v is not supported for column %s", condition.Operator, column)
	}
}

var (
	logicalOperatorToFuncName = map[querylang.LogicalOperator]string{
		querylang.AND: "hasAll",
		querylang.OR:  "hasAny",
	}
)

func buildMultiValueWhereClause(op querylang.LogicalOperator, column string, values []string) string {
	return fmt.Sprintf("%s(%s, [%s])", logicalOperatorToFuncName[op], column, strings.Join(values, ", "))
}

// only support equality checks for array fields
func buildArrayColumnWhereClause(column string, matcher *querylang.Matcher) (string, error) {
	values := make([]string, 0, len(matcher.Conditions))

	for _, condition := range matcher.Conditions {
		if condition.Operator != operator.Eq {
			return "", fmt.Errorf("unsupported operator %v for array column %s", condition.Operator, column)
		}

		if condition.Inverse {
			return "", fmt.Errorf("inverse operators are not supported for array column: %s", column)
		}

		valueRepr, err := buildValueRepr(matcher.Field, condition.Value)
		if err != nil {
			return "", err
		}

		values = append(values, valueRepr)
	}

	return buildMultiValueWhereClause(matcher.Operator, column, values), nil
}

func buildEnvWhereClause(matcher *querylang.Matcher) (string, error) {
	envKey, ok := env.BuildEnvKeyFromMatcherField(matcher.Field)
	if !ok {
		return "", fmt.Errorf("failed to build env key from matcher field: %v", matcher.Field)
	}

	val, err := profilequerylang.ExtractEqualityMatch(matcher)
	if err != nil {
		return "", fmt.Errorf("failed to build where clause for env %v: %w", matcher.Field, err)
	}

	concatenatedEnv := env.BuildConcatenatedEnv(envKey, val)
	return buildMultiValueWhereClause(matcher.Operator, "envs", []string{fmt.Sprintf("'%s'", sqlbuilder.Escape(concatenatedEnv))}), nil
}

func buildSingleValueColumnWhereClause(column string, matcher *querylang.Matcher) (string, error) {
	conditions := make([]string, 0, len(matcher.Conditions))

	for _, condition := range matcher.Conditions {
		condition, err := buildConditionString(column, condition)
		if err != nil {
			return "", err
		}
		conditions = append(conditions, condition)
	}

	separator := " AND "
	if matcher.Operator == querylang.OR {
		separator = " OR "
	}

	if len(conditions) == 0 {
		return "", errors.New("empty where clause for matcher")
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}

	return "(" + strings.Join(conditions, separator) + ")", nil
}

func buildMatcherWhereClause(matcher *querylang.Matcher) (string, error) {
	if env.IsEnvMatcherField(matcher.Field) {
		return buildEnvWhereClause(matcher)
	}

	clauses := make([]string, 0, len(labelsToColumns[matcher.Field]))
	for _, column := range labelsToColumns[matcher.Field] {
		var clause string
		var err error
		if arrayColumns[column] {
			clause, err = buildArrayColumnWhereClause(column, matcher)
		} else {
			clause, err = buildSingleValueColumnWhereClause(column, matcher)
		}
		if err != nil {
			return "", fmt.Errorf("failed to build column `%s` where clause: %w", clause, err)
		}
		clauses = append(clauses, clause)
	}

	if len(clauses) == 0 {
		return "", errors.New("no where clauses are build for querylang.Matcher")
	}

	if len(clauses) == 1 {
		return clauses[0], nil
	}

	return "(" + strings.Join(clauses, " OR ") + ")", nil
}

func makeSelectProfilesQueryBuilder(
	query *meta.ProfileQuery,
	excludeExpired bool,
) (*sqlbuilder.SelectQueryBuilder, error) {
	builder := sqlbuilder.Select().
		Values(AllColumns).
		From("profiles")

	if excludeExpired {
		builder.Where("expired = false")
	}

	for _, matcher := range query.Selector.Matchers {
		if tls.IsTLSMatcherField(matcher.Field) {
			continue
		}

		clause, err := buildMatcherWhereClause(matcher)
		if err != nil {
			return nil, fmt.Errorf("failed to build matcher `%s` where clause: %w", matcher.Field, err)
		}

		builder.Where(clause)
	}

	if query.MaxSamples != 0 {
		if len(query.SortOrder.Columns) != 0 {
			return nil, fmt.Errorf("cannot combine sort order with max samples")
		}

		builder.OrderByColumn("farmHash64(id)")
		builder.Limit(query.MaxSamples)
	} else {
		if query.Pagination.Offset != 0 {
			builder.Offset(query.Pagination.Offset)
		}
		if query.Pagination.Limit != 0 {
			builder.Limit(query.Pagination.Limit)
		}

		if len(query.SortOrder.Columns) == 0 {
			builder.OrderByColumn("timestamp")
		} else {
			builder.OrderBy(makeOrderBy(&query.SortOrder))
		}
	}

	return builder, nil
}

func buildSelectProfilesQuery(query *meta.ProfileQuery) (string, error) {
	builder, err := makeSelectProfilesQueryBuilder(query, true)
	if err != nil {
		return "", err
	}
	return builder.Query()
}

func makeOrderBy(order *util.SortOrder) *sqlbuilder.OrderBy {
	return &sqlbuilder.OrderBy{Columns: order.Columns, Descending: order.Descending}
}
