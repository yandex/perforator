package filter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

func TestDeduct(t *testing.T) {
	tests := map[string]DeductResult{
		`{pod_id="abc"}`:                    DeductResult{PodFilter, "abc"},
		`{node_id="abc"}`:                   DeductResult{NodeFilter, "abc"},
		`{service="bc"}`:                    DeductResult{ServiceFilter, "bc"},
		`{node_id!="abc"}`:                  DeductResult{AbstractFilter, ""},
		`{}`:                                DeductResult{AbstractFilter, ""},
		`{build_id="a", service="abacaba"}`: DeductResult{AbstractFilter, ""},
		`{pod_id="abc|abacaba"}`:            DeductResult{AbstractFilter, ""},
		`{pod_id="abc", timestamp<"now"}`:   DeductResult{PodFilter, "abc"},
		`{timestamp>"now-1h", pod_id="abc", timestamp<"now"}`: DeductResult{PodFilter, "abc"},
	}

	for selectorString, deductResult := range tests {
		t.Run(selectorString, func(t *testing.T) {
			selector, err := profilequerylang.ParseSelector(selectorString)
			require.NoError(t, err)

			res := DeductMicroscope(selector)
			require.Equal(t, deductResult, *res)
		})
	}
}
