package collapsed_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
)

func TestCollapsedParsing(t *testing.T) {
	for i, test := range []struct {
		raw         string
		expected    *string
		profile     *collapsed.Profile
		err         bool
		noroundtrip bool
	}{{
		raw: `printf;malloc;memcpy 42`,
		profile: &collapsed.Profile{
			Samples: []collapsed.Sample{{
				Stack: []string{"printf", "malloc", "memcpy"},
				Value: 42,
			}},
		},
	}, {
		raw: `aaa aaa 1


std::__v1::__unordered_map_base<std::__v1::__unordered_map_derived_base_direct_virtual_holder_1<std::__v1::basic_string_without_cow 1099511627776`,
		profile: &collapsed.Profile{
			Samples: []collapsed.Sample{{
				Stack: []string{"aaa aaa"},
				Value: 1,
			}, {
				Stack: []string{"std::__v1::__unordered_map_base<std::__v1::__unordered_map_derived_base_direct_virtual_holder_1<std::__v1::basic_string_without_cow"},
				Value: 1099511627776,
			}},
		},
		noroundtrip: true,
	}, {
		raw: `hex;count 0xdeadbeef`,
		profile: &collapsed.Profile{
			Samples: []collapsed.Sample{{
				Stack: []string{"hex", "count"},
				Value: 3735928559,
			}},
		},
		expected: ptr.String(`hex;count 3735928559`),
	}, {
		raw: `abc`,
		err: true,
	}, {
		raw: `i love c++`,
		err: true,
	}} {
		t.Run(fmt.Sprintf("collapsed/%d", i), func(t *testing.T) {
			profile, err := collapsed.Unmarshal([]byte(test.raw))
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, profile, test.profile)

				raw, err := collapsed.Marshal(profile)
				require.NoError(t, err)
				if !test.noroundtrip {
					if test.expected != nil {
						require.Equal(t, strings.TrimSpace(string(raw)), *test.expected)
					} else {
						require.Equal(t, strings.TrimSpace(string(raw)), test.raw)
					}
				}
			}
		})
	}
}
