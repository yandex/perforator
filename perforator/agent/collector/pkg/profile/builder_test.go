package profile

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

func TestInterpreterLocationsPreserveIdentity(t *testing.T) {
	// Language IDs are opaque to the builder.
	identities := []struct {
		language  models.Language
		startTime uint64
		mapping   string
	}{
		{1, 100, PythonSpecialMapping},
		{1, 200, PythonSpecialMapping},
		{2, 100, PHPSpecialMapping},
		{3, 100, LuaSpecialMapping},
	}
	b := NewBuilderWithCaches(NewProcessCaches()).AddSampleType("cpu", "cycles")
	for range 2 {
		for _, identity := range identities {
			sample := b.Add(linux.ProcessKey{Pid: 42, ProcessStartTime: identity.startTime}).AddValue(1)
			loc := sample.AddInterpreterLocation(&InterpreterLocationKey{
				ObjectAddress: 0xabc,
				Linestart:     10,
				Language:      identity.language,
			})
			loc.SetMapping().SetPath(identity.mapping).Finish()
			loc.AddFrame().
				SetName(fmt.Sprintf("%s-process-%d", identity.mapping, identity.startTime)).
				SetFilename(identity.mapping + "-source").
				SetStartLine(10).
				Finish()
			loc.Finish()
			sample.Finish()
		}
	}

	p := b.FinishRaw()
	require.Len(t, p.Sample, 2*len(identities))
	require.Len(t, p.Location, len(identities))
	for i, identity := range identities {
		loc := p.Sample[i].Location[0]
		require.Equal(t, identity.mapping, loc.Mapping.File)
		require.Len(t, loc.Line, 1)
		require.Equal(t, fmt.Sprintf("%s-process-%d", identity.mapping, identity.startTime), loc.Line[0].Function.Name)
		require.Equal(t, identity.mapping+"-source", loc.Line[0].Function.Filename)
		require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
		require.Same(t, loc, p.Sample[i+len(identities)].Location[0])
	}
}

func TestInterpreterLocationKey_LineDedup(t *testing.T) {
	caches := NewProcessCaches()
	b := NewBuilderWithCaches(caches).AddSampleType("cpu", "cycles")

	lines := []int32{20, 30, 20}
	for _, line := range lines {
		sb := b.Add(linux.ProcessKey{Pid: 1}).AddValue(1)
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

func TestNativeLocationsPreserveProcessLifetime(t *testing.T) {
	b := NewBuilderWithCaches(NewProcessCaches()).AddSampleType("cpu", "cycles")
	processes := []linux.ProcessKey{
		{Pid: 42, ProcessStartTime: 100},
		{Pid: 42, ProcessStartTime: 200},
		{Pid: 43, ProcessStartTime: 100},
		{Pid: 42, ProcessStartTime: 100},
	}
	for _, process := range processes {
		sample := b.AddTimestampedSample(process, time.Unix(1, 0)).AddValue(1)
		loc := sample.AddNativeLocation(0x1000)
		loc.SetMapping().SetBegin(0x1000).SetEnd(0x2000).
			SetPath(fmt.Sprintf("process-%d-%d", process.Pid, process.ProcessStartTime)).Finish()
		loc.AddFrame().SetName(fmt.Sprintf("symbol-%d-%d", process.Pid, process.ProcessStartTime)).Finish()
		loc.Finish()
		sample.Finish()
	}
	p := b.FinishRaw()
	require.Len(t, p.Location, 3)
	require.Len(t, p.Mapping, 3)
	for i, process := range processes {
		loc := p.Sample[i].Location[0]
		require.Equal(t, fmt.Sprintf("process-%d-%d", process.Pid, process.ProcessStartTime), loc.Mapping.File)
		require.Equal(t, fmt.Sprintf("symbol-%d-%d", process.Pid, process.ProcessStartTime), loc.Line[0].Function.Name)
	}
	require.Same(t, p.Sample[0].Location[0], p.Sample[3].Location[0])
}
