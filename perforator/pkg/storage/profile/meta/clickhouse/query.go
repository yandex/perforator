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

func forEachCHField(row interface{}, callback func(fieldIndex int, structField reflect.StructField, fieldValue *reflect.Value) error) error {
	t := reflect.TypeOf(row)
	var v reflect.Value

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = reflect.ValueOf(row).Elem()
	} else {
		v = reflect.ValueOf(row)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := field.Tag.Lookup("ch"); ok {
			fieldValue := v.Field(i)
			if err := callback(i, field, &fieldValue); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateAllColumns(row interface{}) string {
	var columns []string

	err := forEachCHField(row, func(fieldIndex int, structField reflect.StructField, fieldValue *reflect.Value) error {
		if tag, ok := structField.Tag.Lookup("ch"); ok {
			columns = append(columns, tag)
		}
		return nil
	})

	if err != nil {
		// This shouldn't happen in generateAllColumns, but handle it gracefully
		panic(fmt.Sprintf("unexpected error in generateAllColumns: %v", err))
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

	envsColumn = "envs"
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
	return buildMultiValueWhereClause(matcher.Operator, envsColumn, []string{fmt.Sprintf("'%s'", sqlbuilder.Escape(concatenatedEnv))}), nil
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
) (*sqlbuilder.SelectQueryBuilder, error) {
	builder := sqlbuilder.Select().
		Where("expired = false").
		Values(AllColumns).
		From("profiles")

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
	builder, err := makeSelectProfilesQueryBuilder(query)
	if err != nil {
		return "", err
	}
	return builder.Query()
}

func makeOrderBy(order *util.SortOrder) *sqlbuilder.OrderBy {
	return &sqlbuilder.OrderBy{Columns: order.Columns, Descending: order.Descending}
}

func buildInsertQuery(rows []*ProfileRow) (string, error) {
	if len(rows) == 0 {
		return "", nil
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO profiles (")
	queryBuilder.WriteString(AllColumns)
	queryBuilder.WriteString(") SETTINGS async_insert=1, wait_for_async_insert=1 VALUES ")

	for i, row := range rows {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		err := formatRowForInsert(&queryBuilder, row)
		if err != nil {
			return "", fmt.Errorf("failed to format row: %w", err)
		}
	}

	return queryBuilder.String(), nil
}

func formatRowForInsert(builder *strings.Builder, row *ProfileRow) error {
	builder.WriteByte('(')

	firstField := true
	err := forEachCHField(row, func(fieldIndex int, structField reflect.StructField, fieldValue *reflect.Value) error {
		if !firstField {
			builder.WriteString(", ")
		}
		firstField = false

		if err := formatFieldForInsert(builder, *fieldValue); err != nil {
			return fmt.Errorf("failed to format field %s: %w", structField.Name, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	builder.WriteByte(')')
	return nil
}

func formatFieldForInsert(builder *strings.Builder, field reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		builder.WriteByte('\'')
		escapeStringToBuilder(builder, field.String())
		builder.WriteByte('\'')

	case reflect.Bool:
		if field.Bool() {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}

	case reflect.Slice:
		switch field.Type().Elem().Kind() {
		case reflect.String:
			formatStringSliceForInsert(builder, field)
		default:
			return fmt.Errorf("unsupported slice type: %v", field.Type())
		}

	case reflect.Map:
		if field.Type().Key().Kind() == reflect.String && field.Type().Elem().Kind() == reflect.String {
			formatStringMapForInsert(builder, field)
		} else {
			return fmt.Errorf("unsupported map type: %v", field.Type())
		}

	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			timestamp := field.Interface().(time.Time)
			milliseconds := timestamp.UnixMilli()
			builder.WriteString(fmt.Sprintf("%d", milliseconds))
		} else {
			return fmt.Errorf("unsupported struct type: %v", field.Type())
		}

	default:
		return fmt.Errorf("unsupported field type: %v", field.Type())
	}

	return nil
}

func formatStringSliceForInsert(builder *strings.Builder, field reflect.Value) {
	if field.Len() == 0 {
		builder.WriteString("[]")
		return
	}

	builder.WriteByte('[')
	for i := 0; i < field.Len(); i++ {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteByte('\'')
		escapeStringToBuilder(builder, field.Index(i).String())
		builder.WriteByte('\'')
	}
	builder.WriteByte(']')
}

func formatStringMapForInsert(builder *strings.Builder, field reflect.Value) {
	builder.WriteByte('{')
	first := true
	iter := field.MapRange()
	for iter.Next() {
		if !first {
			builder.WriteString(", ")
		}
		first = false

		builder.WriteByte('\'')
		escapeStringToBuilder(builder, iter.Key().String())
		builder.WriteString("': '")
		escapeStringToBuilder(builder, iter.Value().String())
		builder.WriteByte('\'')
	}
	builder.WriteByte('}')
}

func escapeStringToBuilder(builder *strings.Builder, str string) {
	for _, r := range str {
		switch r {
		case '\b':
			builder.WriteString("\\b")
		case '\f':
			builder.WriteString("\\f")
		case '\r':
			builder.WriteString("\\r")
		case '\n':
			builder.WriteString("\\n")
		case '\t':
			builder.WriteString("\\t")
		case '\x00':
			builder.WriteString("\\0")
		case '\a':
			builder.WriteString("\\a")
		case '\v':
			builder.WriteString("\\v")
		case '\\':
			builder.WriteString("\\\\")
		case '\'':
			builder.WriteString("\\'")
		default:
			builder.WriteRune(r)
		}
	}
}
