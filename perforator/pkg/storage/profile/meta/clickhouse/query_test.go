package clickhouse

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

func TestQueryBuild(t *testing.T) {
	queryBase := meta.ProfileQuery{
		Pagination: util.Pagination{
			Limit: 10,
		},
	}

	makeQuery := func(selector string) *meta.ProfileQuery {
		query := queryBase
		parsedSelector, err := profilequerylang.ParseSelector(selector)
		require.NoError(t, err)
		query.Selector = parsedSelector
		return &query
	}
	queries := map[string]string{
		`{service="perforator|web-search", build_ids="a|b"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND (service = 'perforator' OR service = 'web-search')
				AND hasAny(build_ids, ['a', 'b'])
			ORDER BY`,
			AllColumns,
		),
		`{"service"="perforator.storage-production"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND service = 'perforator.storage-production'
			ORDER BY`,
			AllColumns,
		),
		`{"cpu"="Intel", profiler_version="12341|12|156", build_ids="a"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND attributes['cpu'] = 'Intel'
				AND (attributes['profiler_version'] = '12341' OR attributes['profiler_version'] = '12' OR attributes['profiler_version'] = '156')
				AND hasAny(build_ids, ['a'])
			ORDER BY`,
			AllColumns,
		),
		`{}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
			ORDER BY`,
			AllColumns,
		),
		`{id="a|b|y"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND (id = 'a' OR id = 'b' OR id = 'y')
			ORDER BY`,
			AllColumns,
		),
		`{id="a|b|y", tls.KEY="value"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND (id = 'a' OR id = 'b' OR id = 'y')
			ORDER BY`,
			AllColumns,
		),
		`{id="a|b|y", env.KEY="value", env.KEY2="value2"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles
			WHERE expired = false
				AND (id = 'a' OR id = 'b' OR id = 'y')
				AND hasAny(envs, ['KEY=value'])
				AND hasAny(envs, ['KEY2=value2'])
			ORDER BY`,
			AllColumns,
		),
		`{event_type="cpu.cycles"}`: fmt.Sprintf(`
			SELECT %s 
			FROM profiles
			WHERE expired=false
				AND event_type='cpu.cycles'
			ORDER BY`,
			AllColumns,
		),
		`{event_type="wall.seconds", service="perforator.storage-prestable"}`: fmt.Sprintf(`
			SELECT %s
			FROM profiles 
			WHERE expired=false
				AND event_type='wall.seconds'
				AND service='perforator.storage-prestable'
			ORDER BY
			`,
			AllColumns,
		),
	}

	for selector, expectedSQLprefix := range queries {
		t.Run(selector, func(t *testing.T) {
			sql, err := buildSelectProfilesQuery(makeQuery(selector))
			require.NoError(t, err)

			expectedSQLprefix = strings.ReplaceAll(expectedSQLprefix, "\n", "")
			expectedSQLprefix = strings.ReplaceAll(expectedSQLprefix, "\t", "")
			expectedSQLprefix = strings.ReplaceAll(expectedSQLprefix, " ", "")
			sql = strings.ReplaceAll(sql, " ", "")

			require.True(
				t,
				strings.HasPrefix(sql, expectedSQLprefix),
				"%s does not have prefix %s",
				sql,
				expectedSQLprefix,
			)
		})
	}
}
