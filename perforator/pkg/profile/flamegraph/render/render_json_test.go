package render

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render/format"
)

func TestRenderJSON(t *testing.T) {
	raw, err := os.ReadFile(yatest.WorkPath("stacks/stacks-large.txt"))
	if err != nil {
		panic(err)
	}

	profile, err := collapsed.Decode(bytes.NewBuffer(raw))

	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	bufferWriter := bufio.NewWriter(&buf)
	fg := NewFlameGraph()
	fg.SetFormat(JSONFormat)

	err = fg.RenderCollapsed(profile, bufferWriter)

	if err != nil {
		panic(err)
	}

	var data format.ProfileData

	err = json.Unmarshal(buf.Bytes(), &data)

	if err != nil {
		panic(err)
	}

	t.Run("prev index must be less than max index in prev row", func(t *testing.T) {
		for h, row := range data.Nodes {
			if h == 0 {
				continue
			}
			prevRowMax := len(data.Nodes[h-1]) - 1
			for _, node := range row {
				if node.ParentIndex > prevRowMax {
					t.Error("ParentIndex is greater than prevRowMax")
				}
			}
		}
	})

	t.Run("event sum for each row must be less than or equal of the prev row", func(t *testing.T) {
		var prevEventSum float64
		for h, row := range data.Nodes {
			if h == 0 {
				prevEventSum = row[0].EventCount
				continue
			}

			var eventSum float64
			for _, node := range row {
				eventSum += node.EventCount
			}
			if eventSum > prevEventSum {
				t.Error("EventSum is greater than prevEventSum")
			}
			prevEventSum = eventSum
		}
	})

	t.Run("check for every function in stack that it is in the flame json", func(t *testing.T) {
		for _, sample := range profile.Samples {
			for i, frame := range sample.Stack {
				found := false
			outer:
				for h, row := range data.Nodes {
					for _, node := range row {
						// h-1 off-by-one due to artificial root in the flamegraph
						if data.Strings[node.TextID] == frame && (h-1) == i {

							found = true
							break outer
						}
					}
				}
				if !found {
					t.Errorf("Function %s not found in flame json on position %d", frame, i)
				}
			}
		}
	})
}
