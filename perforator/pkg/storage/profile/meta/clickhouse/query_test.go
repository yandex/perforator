package clickhouse

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestFormatFieldForInsert(t *testing.T) {
	var builder strings.Builder
	timestamp := time.Date(2025, 6, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string field",
			value:    "test string",
			expected: "'test string'",
		},
		{
			name:     "string with quotes",
			value:    "test'string",
			expected: "'test\\'string'",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},
		{
			name:     "string slice",
			value:    []string{"a", "b", "c"},
			expected: "['a', 'b', 'c']",
		},
		{
			name:     "empty string slice",
			value:    []string{},
			expected: "[]",
		},
		{
			name:     "string map",
			value:    map[string]string{"key1": "value1", "key2": "value2"},
			expected: "",
		},
		{
			name:     "empty string map",
			value:    map[string]string{},
			expected: "{}",
		},
		{
			name:     "timestamp",
			value:    timestamp,
			expected: fmt.Sprintf("%d", timestamp.UnixMilli()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.Reset()
			field := reflect.ValueOf(tt.value)

			err := formatFieldForInsert(&builder, field)
			require.NoError(t, err)

			result := builder.String()
			if tt.name == "string map" {
				// we should not rely on map traversal order
				require.True(t, strings.Contains(result, "'key1': 'value1'"))
				require.True(t, strings.Contains(result, "'key2': 'value2'"))
				require.True(t, strings.HasPrefix(result, "{"))
				require.True(t, strings.HasSuffix(result, "}"))
				require.Contains(t, result, ", ")
			} else {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEscapeStringToBuilder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "simple",
			expected: "simple",
		},
		{
			name:     "with single quote",
			input:    "with'quote",
			expected: "with\\'quote",
		},
		{
			name:     "with backslash",
			input:    "with\\backslash",
			expected: "with\\\\backslash",
		},
		{
			name:     "with newline",
			input:    "with\nline",
			expected: "with\\nline",
		},
		{
			name:     "with carriage return",
			input:    "with\rreturn",
			expected: "with\\rreturn",
		},
		{
			name:     "with tab",
			input:    "with\ttab",
			expected: "with\\ttab",
		},
		{
			name:     "with null byte",
			input:    "with\x00null",
			expected: "with\\0null",
		},
		{
			name:     "with backspace",
			input:    "with\bbackspace",
			expected: "with\\bbackspace",
		},
		{
			name:     "with form feed",
			input:    "with\ffeed",
			expected: "with\\ffeed",
		},
		{
			name:     "with bell",
			input:    "with\abell",
			expected: "with\\abell",
		},
		{
			name:     "with vertical tab",
			input:    "with\vtab",
			expected: "with\\vtab",
		},
		{
			name:     "multiple special chars",
			input:    "test'quote\\back\nline\ttab",
			expected: "test\\'quote\\\\back\\nline\\ttab",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			escapeStringToBuilder(&builder, tt.input)
			result := builder.String()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildInsertQuery(t *testing.T) {
	timestamp := time.Date(2025, 6, 9, 12, 0, 0, 0, time.UTC)

	// Do not use map of more than 1 element because serialization order is not guaranteed for maps
	tests := []struct {
		name        string
		rows        []*ProfileRow
		expected    string
		expectError bool
	}{
		{
			name:     "empty rows",
			rows:     []*ProfileRow{},
			expected: "",
		},
		{
			name: "single row with basic fields",
			rows: []*ProfileRow{
				{
					ID:            "test-id-1",
					System:        "test-system",
					MainEventType: "cpu.cycles",
					AllEventTypes: []string{"cpu.cycles", "wall.seconds"},
					Cluster:       "test-cluster",
					Service:       "test-service",
					PodID:         "test-pod",
					NodeID:        "test-node",
					Timestamp:     timestamp,
					BuildIDs:      []string{"build1", "build2"},
					Attributes:    map[string]string{"cpu": "Intel"},
					Expired:       false,
					Envs:          []string{"KEY=value", "ENV=prod"},
				},
			},
			expected: fmt.Sprintf("INSERT INTO profiles (%s) SETTINGS async_insert=1, wait_for_async_insert=1 VALUES ('test-id-1', 'test-system', 'cpu.cycles', ['cpu.cycles', 'wall.seconds'], 'test-cluster', 'test-service', 'test-pod', 'test-node', %d, ['build1', 'build2'], {'cpu': 'Intel'}, false, ['KEY=value', 'ENV=prod'])", AllColumns, timestamp.UnixMilli()),
		},
		{
			name: "single row with empty collections",
			rows: []*ProfileRow{
				{
					ID:            "test-id-2",
					System:        "system2",
					MainEventType: "wall.seconds",
					AllEventTypes: []string{},
					Cluster:       "cluster2",
					Service:       "service2",
					PodID:         "pod2",
					NodeID:        "node2",
					Timestamp:     timestamp,
					BuildIDs:      []string{},
					Attributes:    map[string]string{},
					Expired:       true,
					Envs:          []string{},
				},
			},
			expected: fmt.Sprintf("INSERT INTO profiles (%s) SETTINGS async_insert=1, wait_for_async_insert=1 VALUES ('test-id-2', 'system2', 'wall.seconds', [], 'cluster2', 'service2', 'pod2', 'node2', %d, [], {}, true, [])", AllColumns, timestamp.UnixMilli()),
		},
		{
			name: "multiple rows",
			rows: []*ProfileRow{
				{
					ID:            "id1",
					System:        "sys1",
					MainEventType: "cpu.cycles",
					AllEventTypes: []string{"cpu.cycles"},
					Cluster:       "cluster1",
					Service:       "service1",
					PodID:         "pod1",
					NodeID:        "node1",
					Timestamp:     timestamp,
					BuildIDs:      []string{"build1"},
					Attributes:    map[string]string{"key": "value"},
					Expired:       false,
					Envs:          []string{"ENV=test"},
				},
				{
					ID:            "id2",
					System:        "sys2",
					MainEventType: "wall.seconds",
					AllEventTypes: []string{"wall.seconds"},
					Cluster:       "cluster2",
					Service:       "service2",
					PodID:         "pod2",
					NodeID:        "node2",
					Timestamp:     timestamp,
					BuildIDs:      []string{"build2"},
					Attributes:    map[string]string{"cpu": "AMD"},
					Expired:       true,
					Envs:          []string{"ENV=prod"},
				},
			},
			expected: fmt.Sprintf("INSERT INTO profiles (%s) SETTINGS async_insert=1, wait_for_async_insert=1 VALUES ('id1', 'sys1', 'cpu.cycles', ['cpu.cycles'], 'cluster1', 'service1', 'pod1', 'node1', %d, ['build1'], {'key': 'value'}, false, ['ENV=test']), ('id2', 'sys2', 'wall.seconds', ['wall.seconds'], 'cluster2', 'service2', 'pod2', 'node2', %d, ['build2'], {'cpu': 'AMD'}, true, ['ENV=prod'])", AllColumns, timestamp.UnixMilli(), timestamp.UnixMilli()),
		},
		{
			name: "row with special characters in strings",
			rows: []*ProfileRow{
				{
					ID:            "test'id",
					System:        "sys\\tem",
					MainEventType: "cpu.cycles",
					AllEventTypes: []string{"cpu.cycles"},
					Cluster:       "cluster\nname",
					Service:       "service\ttab",
					PodID:         "pod\rid",
					NodeID:        "node\x00null",
					Timestamp:     timestamp,
					BuildIDs:      []string{"build'1", "build\\2"},
					Attributes:    map[string]string{"ke'y": "val'ue"},
					Expired:       false,
					Envs:          []string{"KEY='value'", "ENV=test\\path"},
				},
			},
			expected: fmt.Sprintf("INSERT INTO profiles (%s) SETTINGS async_insert=1, wait_for_async_insert=1 VALUES ('test\\'id', 'sys\\\\tem', 'cpu.cycles', ['cpu.cycles'], 'cluster\\nname', 'service\\ttab', 'pod\\rid', 'node\\0null', %d, ['build\\'1', 'build\\\\2'], {'ke\\'y': 'val\\'ue'}, false, ['KEY=\\'value\\'', 'ENV=test\\\\path'])", AllColumns, timestamp.UnixMilli()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildInsertQuery(tt.rows)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, "", result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
