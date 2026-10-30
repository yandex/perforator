package aggregated

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Masterminds/squirrel"

	pb "github.com/yandex/perforator/perforator/proto/perforator"
)

var ErrUnsupportedShape = errors.New("unsupported Cluster Top query shape")

type dimension = pb.ClusterTopDimension

var columns = map[dimension]string{
	pb.ClusterTopDimension_Function:     "function",
	pb.ClusterTopDimension_Service:      "service",
	pb.ClusterTopDimension_BuildID:      "build_id",
	pb.ClusterTopDimension_CommitID:     "commit_id",
	pb.ClusterTopDimension_BinaryPath:   "binary_path",
	pb.ClusterTopDimension_Filename:     "source_file",
	pb.ClusterTopDimension_SourceLine:   "source_line",
	pb.ClusterTopDimension_SourceColumn: "source_column",
}

var stringFilterColumns = map[pb.ClusterTopStringFilter_FieldName]string{
	pb.ClusterTopStringFilter_Function:   "function",
	pb.ClusterTopStringFilter_Service:    "service",
	pb.ClusterTopStringFilter_BuildID:    "build_id",
	pb.ClusterTopStringFilter_CommitID:   "commit_id",
	pb.ClusterTopStringFilter_BinaryPath: "binary_path",
	pb.ClusterTopStringFilter_Filename:   "source_file",
}

// ValidateQuery rejects malformed and unsupported requests before any storage I/O.
func ValidateQuery(req *pb.ClusterTopQueryRequest) error {
	_, err := prepareQuery(req)
	return err
}

func queryLimit(req *pb.ClusterTopQueryRequest) uint64 {
	if limit := req.GetPagination().GetLimit(); limit != 0 {
		return limit
	}
	return DefaultPageSize
}

func prepareQuery(req *pb.ClusterTopQueryRequest) (squirrel.SelectBuilder, error) {
	scope := req.GetQuery()
	if scope.GetGeneration() == 0 {
		return squirrel.SelectBuilder{}, errors.New("generation must be nonzero")
	}
	limit, offset := queryLimit(req), req.GetPagination().GetOffset()
	if limit > 1000 || offset > 10000 {
		return squirrel.SelectBuilder{}, errors.New("pagination exceeds limit 1000 or offset 10000")
	}
	if len(req.GroupBy) == 0 || len(req.GroupBy) > len(columns) {
		return squirrel.SelectBuilder{}, errors.New("GroupBy must contain between one and eight dimensions")
	}
	groups := slices.Clone(req.GroupBy)
	slices.Sort(groups)
	for i, field := range groups {
		if _, ok := columns[field]; !ok {
			return squirrel.SelectBuilder{}, errors.New("unknown group dimension")
		}
		if i > 0 && groups[i-1] == field {
			return squirrel.SelectBuilder{}, errors.New("duplicate group dimension")
		}
	}
	supported := true
	builder := squirrel.Select().From("cluster_top_v3 AS src").
		Where(squirrel.Eq{"src.generation": scope.Generation, "src.event_type": "cpu.cycles"})
	functionFilters := 0
	for _, fieldFilter := range scope.Filters {
		var filter *pb.ClusterTopStringFilter
		switch typed := fieldFilter.GetFilter().(type) {
		case *pb.ClusterTopFieldFilter_String_:
			if typed == nil || typed.String_ == nil {
				return squirrel.SelectBuilder{}, errors.New("missing string filter")
			}
			filter = typed.String_
		case *pb.ClusterTopFieldFilter_Integer:
			if typed == nil {
				return squirrel.SelectBuilder{}, errors.New("missing integer filter")
			}
			if err := validateIntegerFilter(typed.Integer); err != nil {
				return squirrel.SelectBuilder{}, err
			}
			supported = false
			continue
		default:
			return squirrel.SelectBuilder{}, errors.New("missing filter type")
		}
		column, ok := stringFilterColumns[filter.Field]
		if !ok {
			return squirrel.SelectBuilder{}, errors.New("unknown string filter field")
		}
		if filter.Field == pb.ClusterTopStringFilter_Function {
			functionFilters++
		}
		var values []string
		switch condition := filter.Condition.(type) {
		case *pb.ClusterTopStringFilter_Exact:
			values = []string{condition.Exact}
		case *pb.ClusterTopStringFilter_In:
			values = condition.In.GetValues()
			maxValues := 5
			if filter.Field == pb.ClusterTopStringFilter_Function {
				maxValues = 10
			}
			if len(values) == 0 || len(values) > maxValues {
				return squirrel.SelectBuilder{}, fmt.Errorf("In requires 1-%d distinct values", maxValues)
			}
		case *pb.ClusterTopStringFilter_Prefix:
			values = []string{condition.Prefix}
			if filter.Field == pb.ClusterTopStringFilter_Function {
				supported = false
			}
		case *pb.ClusterTopStringFilter_Contains:
			values = []string{condition.Contains}
			if filter.Field == pb.ClusterTopStringFilter_Function {
				supported = false
			}
		default:
			return squirrel.SelectBuilder{}, errors.New("missing filter condition")
		}
		seen := make(map[string]bool, len(values))
		for _, value := range values {
			if len(value) > 4096 {
				return squirrel.SelectBuilder{}, errors.New("filter value exceeds 4096 bytes")
			}
			if value == "" && filter.Field == pb.ClusterTopStringFilter_Function {
				return squirrel.SelectBuilder{}, errors.New("function must be nonempty")
			}
			if seen[value] {
				return squirrel.SelectBuilder{}, errors.New("duplicate filter value")
			}
			seen[value] = true
		}
		switch filter.Condition.(type) {
		case *pb.ClusterTopStringFilter_Prefix:
			builder = builder.Where("startsWith(src."+column+", ?)", values[0])
		case *pb.ClusterTopStringFilter_Contains:
			builder = builder.Where("position(src."+column+", ?) > 0", values[0])
		default:
			builder = builder.Where(squirrel.Eq{"src." + column: values})
		}
	}
	if functionFilters != 1 {
		supported = false
	}

	// Always scan a fixed row shape; unselected dimensions have sentinel values.
	for field := pb.ClusterTopDimension_Function; field <= pb.ClusterTopDimension_SourceColumn; field++ {
		column := columns[field]
		expression := "src." + column + " AS " + column
		if !slices.Contains(groups, field) {
			expression = "'' AS " + column
			if field >= pb.ClusterTopDimension_SourceLine {
				expression = "toUInt32(0) AS " + column
			}
		}
		builder = builder.Column(expression)
	}
	builder = builder.Columns("sum(src.self_cycles) AS self_cycles", "sum(src.cumulative_cycles) AS cumulative_cycles")
	for _, field := range groups {
		builder = builder.GroupBy("src." + columns[field])
	}

	ordered := make(map[string]bool)
	for _, order := range req.OrderBy {
		if order == nil || (order.Direction != pb.ClusterTopQueryOrder_Ascending && order.Direction != pb.ClusterTopQueryOrder_Descending) {
			return squirrel.SelectBuilder{}, errors.New("ordering requires an explicit direction")
		}
		var column string
		switch key := order.Key.(type) {
		case *pb.ClusterTopQueryOrder_Dimension:
			if !slices.Contains(groups, key.Dimension) {
				return squirrel.SelectBuilder{}, errors.New("ordering dimension must belong to GroupBy")
			}
			column = columns[key.Dimension]
		case *pb.ClusterTopQueryOrder_Metric:
			switch key.Metric {
			case pb.ClusterTopMetric_SelfCPU:
				column = "self_cycles"
			case pb.ClusterTopMetric_CumulativeCPU:
				column = "cumulative_cycles"
			default:
				return squirrel.SelectBuilder{}, errors.New("unknown ordering metric")
			}
		default:
			return squirrel.SelectBuilder{}, errors.New("missing ordering key")
		}
		if ordered[column] {
			return squirrel.SelectBuilder{}, errors.New("duplicate ordering key")
		}
		ordered[column] = true
		direction := " ASC"
		if order.Direction == pb.ClusterTopQueryOrder_Descending {
			direction = " DESC"
		}
		builder = builder.OrderBy(column + direction)
	}
	for _, field := range groups {
		if !ordered[columns[field]] {
			builder = builder.OrderBy(columns[field] + " ASC")
		}
	}
	if !supported {
		return squirrel.SelectBuilder{}, ErrUnsupportedShape
	}
	return builder.Limit(limit + 1).Offset(offset), nil
}

func validateIntegerFilter(filter *pb.ClusterTopIntegerFieldFilter) error {
	if filter == nil {
		return errors.New("missing integer filter")
	}
	switch filter.Field {
	case pb.ClusterTopIntegerFieldFilter_SourceLine, pb.ClusterTopIntegerFieldFilter_SourceColumn:
	default:
		return errors.New("unknown integer filter field")
	}
	condition, ok := filter.Condition.(*pb.ClusterTopIntegerFieldFilter_Range)
	if !ok || condition == nil {
		return errors.New("missing integer filter condition")
	}
	bounds := condition.Range
	if bounds == nil || (bounds.Min == nil && bounds.Max == nil) {
		return errors.New("range requires at least one bound")
	}
	if bounds.Min != nil && bounds.Max != nil && *bounds.Min > *bounds.Max {
		return errors.New("range minimum exceeds maximum")
	}
	return nil
}
