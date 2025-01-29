package convert_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/convert"
)

func TestPProfConvert(t *testing.T) {
	for i, test := range []struct {
		raw string
	}{{
		raw: `printf;malloc;memcpy 42
`,
	}, {
		raw: `printf;malloc;memcpy 42
kek;ek2;copy 1
aaaaaa ; aaaaaa 123
`,
	}} {
		t.Run(fmt.Sprintf("roundtrip/%d", i), func(t *testing.T) {
			folded, err := collapsed.Unmarshal([]byte(test.raw))
			require.NoError(t, err)
			pprof, err := convert.CollapsedToPProf(folded)
			require.NoError(t, err)
			folded2, err := convert.PProfToCollapsed(pprof)
			require.NoError(t, err)
			raw, err := collapsed.Marshal(folded2)
			require.NoError(t, err)

			require.Equal(t, test.raw, string(raw))
		})
	}
}
