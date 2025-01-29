package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
)

func buildBlocksCollapsed(profile *collapsed.Profile, maxDepth int) []*block {
	fg := NewFlameGraph()
	fg.SetDepthLimit(maxDepth)
	_ = fg.AddCollapsedProfile(profile)
	return fg.bb.Finish(0.0)
}

func TestBlocksBuilder(t *testing.T) {
	type expectedBlocks struct {
		name    string
		level   int
		offset  float64
		samples float64
	}

	tests := []struct {
		raw      string
		maxDepth int
		expected []expectedBlocks
	}{
		{
			raw: "",
			expected: []expectedBlocks{
				{name: "all", level: 0, offset: 0., samples: 0},
			},
		},
		{
			raw: "foo 2\nboo 1",
			expected: []expectedBlocks{
				{name: "all", level: 0, offset: 0. / 3., samples: 3},
				{name: "boo", level: 1, offset: 0. / 3., samples: 1},
				{name: "foo", level: 1, offset: 1. / 3., samples: 2},
			},
		},
		{
			raw: "foo;boo 5\nfoo;bar;baz 1\nbar;baz 10\nbar 7",
			expected: []expectedBlocks{
				{name: "all", level: 0, offset: 0., samples: 23},
				{name: "bar", level: 1, offset: 0., samples: 17},
				{name: "baz", level: 2, offset: 0., samples: 10},
				{name: "foo", level: 1, offset: 17. / 23., samples: 6},
				{name: "bar", level: 2, offset: 17. / 23., samples: 1},
				{name: "baz", level: 3, offset: 17. / 23., samples: 1},
				{name: "boo", level: 2, offset: 18. / 23., samples: 5},
			},
		},
		{
			raw:      "1;2;3;4;5;6;7;8;9;10 1\na;b;c 1\nf1;f2;f3;f4 5",
			maxDepth: 3,
			expected: []expectedBlocks{
				{name: "all", level: 0, offset: 0., samples: 7},

				{name: "1", level: 1, offset: 0., samples: 1},
				{name: "2", level: 2, offset: 0., samples: 1},
				{name: "(truncated stack)", level: 3, offset: 0., samples: 1},

				{name: "a", level: 1, offset: 1. / 7., samples: 1},
				{name: "b", level: 2, offset: 1. / 7., samples: 1},
				{name: "c", level: 3, offset: 1. / 7., samples: 1},

				{name: "f1", level: 1, offset: 2. / 7., samples: 5},
				{name: "f2", level: 2, offset: 2. / 7., samples: 5},
				{name: "(truncated stack)", level: 3, offset: 2. / 7., samples: 5},
			},
		},
	}

	for i := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			raw := tests[i].raw
			expected := tests[i].expected

			profile, err := collapsed.Decode(bytes.NewBufferString(raw))
			require.NoError(t, err)

			blocks := buildBlocksCollapsed(profile, tests[i].maxDepth)
			slices.SortFunc(blocks, func(lhs, rhs *block) int {
				if lhs.offset != rhs.offset {
					if lhs.offset < rhs.offset {
						return -1
					} else if lhs.offset > rhs.offset {
						return 1
					}
					return 0
				}
				if lhs.level < rhs.level {
					return -1
				} else if lhs.level > rhs.level {
					return 1
				}
				return 0
			})

			require.Equal(t, len(blocks), len(expected))
			for i, expected := range expected {
				require.Equal(t, blocks[i].name, expected.name)
				require.Equal(t, blocks[i].level, expected.level)
				require.InDelta(t, blocks[i].offset, expected.offset, 1e-6)
				require.InDelta(t, blocks[i].nextCount.events, expected.samples, 1e-6)
			}
		})
	}
}

func BenchmarkBlocksBuilder_Large(b *testing.B) {
	raw, err := os.ReadFile(yatest.WorkPath("stacks/stacks-large.txt"))
	if err != nil {
		panic(err)
	}

	profile, err := collapsed.Decode(bytes.NewBuffer(raw))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buildBlocksCollapsed(profile, 0)
	}
}

func BenchmarkCollapsedDecode(b *testing.B) {
	raw, err := os.ReadFile(yatest.WorkPath("stacks/stacks-large.txt"))
	if err != nil {
		panic(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := collapsed.Decode(bytes.NewBuffer(raw))
		if err != nil {
			b.Fatal(err)
		}
	}
}
