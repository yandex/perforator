package sqlbuilder

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type OrderBy struct {
	Columns    []string
	Descending bool
}

type Dialect string

const (
	// for yt.TabletClient
	YTDynTableDialect Dialect = "YT_DYN_TABLE"
	DefaultDialect    Dialect = "DEFAULT"
)

type SelectQueryBuilder struct {
	fromTable       string
	selectValues    string
	whereClauses    []string
	orderBy         *OrderBy
	limit           *uint64
	offset          *uint64
	groupByClause   string
	havingByClauses []string
	settings        []string
	dialect         Dialect
}

func Select() *SelectQueryBuilder {
	return &SelectQueryBuilder{
		whereClauses:    make([]string, 0),
		havingByClauses: make([]string, 0),
	}
}

func Escape(str string) string {
	return strings.Replace(str, "'", "''", -1)
}

func BuildQuotedList(vals []string) string {
	res := ""

	for i, val := range vals {
		if i > 0 {
			res += ", "
		}

		res += fmt.Sprintf("'%s'", val)
	}

	return res
}

func (b *SelectQueryBuilder) Query() (string, error) {

	var query string

	// dyn table interface yt.TabletClient use methods as verbs
	// so the query would be used like tx.SelectRows(ctx, query, options)
	if b.dialect == YTDynTableDialect {
		query = fmt.Sprintf("%s FROM [%s]", b.selectValues, b.fromTable)
	} else {
		query = fmt.Sprintf("SELECT %s FROM %s", b.selectValues, b.fromTable)
	}

	if len(b.whereClauses) != 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(b.whereClauses, " AND "))
	}

	if b.groupByClause != "" {
		query += fmt.Sprintf(" GROUP BY %s", b.groupByClause)
	}

	if len(b.havingByClauses) > 0 && b.groupByClause == "" {
		return "", errors.New("having cannot be set unless group by clause is set")
	}

	if len(b.havingByClauses) > 0 {
		query += fmt.Sprintf(" HAVING %s", strings.Join(b.havingByClauses, " AND "))
	}

	if (b.limit != nil || b.offset != nil) && b.orderBy == nil {
		return "", errors.New("order by is required for LIMIT or OFFSET")
	}

	if b.orderBy != nil {
		query += fmt.Sprintf(" ORDER BY %s", strings.Join(b.orderBy.Columns, ", "))
		if b.orderBy.Descending {
			query += " DESC"
		}
	}

	offsetLimitSection := make([]string, 0, 2)

	if b.limit != nil {
		offsetLimitSection = append(offsetLimitSection, fmt.Sprintf(" LIMIT %d", *b.limit))
	}

	if b.offset != nil {
		offsetLimitSection = append(offsetLimitSection, fmt.Sprintf(" OFFSET %d", *b.offset))
	}

	if b.dialect == YTDynTableDialect {
		slices.Reverse(offsetLimitSection)
	}

	if len(offsetLimitSection) != 0 {
		query += strings.Join(offsetLimitSection, "")
	}

	if len(b.settings) > 0 {
		query += fmt.Sprintf(" SETTINGS %s", strings.Join(b.settings, ", "))
	}

	return query, nil
}

func (b *SelectQueryBuilder) Where(clause string) *SelectQueryBuilder {
	b.whereClauses = append(b.whereClauses, clause)
	return b
}

func (b *SelectQueryBuilder) WithDialect(dialect Dialect) *SelectQueryBuilder {
	b.dialect = dialect
	return b
}

func (b *SelectQueryBuilder) GroupBy(clause string) *SelectQueryBuilder {
	b.groupByClause = clause
	return b
}

func (b *SelectQueryBuilder) Having(clause string) *SelectQueryBuilder {
	b.havingByClauses = append(b.havingByClauses, clause)
	return b
}

func (b *SelectQueryBuilder) Values(values string) *SelectQueryBuilder {
	b.selectValues = values
	return b
}

func (b *SelectQueryBuilder) From(table string) *SelectQueryBuilder {
	b.fromTable = table
	return b
}

func (b *SelectQueryBuilder) Offset(offset uint64) *SelectQueryBuilder {
	b.offset = &offset
	return b
}

func (b *SelectQueryBuilder) Limit(limit uint64) *SelectQueryBuilder {
	b.limit = &limit
	return b
}

func (b *SelectQueryBuilder) OrderBy(clause *OrderBy) *SelectQueryBuilder {
	if clause != nil && len(clause.Columns) > 0 {
		b.orderBy = clause
	} else {
		b.orderBy = nil
	}
	return b
}

func (b *SelectQueryBuilder) OrderByColumn(column string) *SelectQueryBuilder {
	return b.OrderBy(&OrderBy{Columns: []string{column}})
}

func (b *SelectQueryBuilder) Settings(setting string) *SelectQueryBuilder {
	b.settings = append(b.settings, setting)
	return b
}
