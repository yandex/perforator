package render_test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"testing"

	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
)

func BenchmarkFlamegraphRender(b *testing.B) {
	raw, err := os.ReadFile(yatest.WorkPath("stacks/stacks-large.txt"))
	if err != nil {
		panic(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		profile, err := collapsed.Decode(bytes.NewBuffer(raw))
		if err != nil {
			b.Fatal(err)
		}

		fg := render.NewFlameGraph()
		err = fg.RenderCollapsed(profile, io.Discard)
		if err != nil {
			panic(err)
		}
	}
}
