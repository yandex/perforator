package profile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterpreterLocationKey_LineDedup(t *testing.T) {
	caches := NewProcessCaches()
	b := NewBuilderWithCaches(caches).AddSampleType("cpu", "cycles")

	lines := []int32{20, 30, 20}
	for _, line := range lines {
		sb := b.Add(1).AddValue(1)
		loc := sb.AddInterpreterLocation(&InterpreterLocationKey{
			ObjectAddress: 0xabc,
			Linestart:     10,
			Line:          line,
		})
		loc.SetMapping().SetPath(string(PythonSpecialMapping)).Finish()
		loc.AddFrame().
			SetName("foo").
			SetFilename("busyloop.py").
			SetStartLine(10).
			SetLine(int64(line)).
			Finish()
		loc.Finish()
		sb.Finish()
	}

	// FinishRaw avoids pprof sample merge so we can assert per-sample locations.
	p := b.FinishRaw()
	require.Len(t, p.Sample, 3)

	loc0 := p.Sample[0].Location[0]
	loc1 := p.Sample[1].Location[0]
	loc2 := p.Sample[2].Location[0]

	require.Equal(t, int64(20), loc0.Line[0].Line)
	require.Equal(t, int64(30), loc1.Line[0].Line)
	require.Equal(t, int64(20), loc2.Line[0].Line)

	// Same ObjectAddress+Linestart+Line → same location object (dedup).
	require.NotEqual(t, loc0.ID, loc1.ID)
	require.Equal(t, loc0.ID, loc2.ID)
}
