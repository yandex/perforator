package profileresult

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
)

func TestParsePprofErrors(t *testing.T) {
	for _, r := range []*Result{nil, {}, {Bundle: bundle.NewBundle(nil, nil)}, {Bundle: bundle.NewYaprofBundle([]byte("invalid"))}} {
		_, err := r.ParsePprof()
		require.Error(t, err)
	}
}
