package filter

import (
	"sort"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
)

type eventType int

const (
	microscopeStart eventType = iota
	microscopeEnd
)

type timeEvent struct {
	ts    time.Time
	event eventType
}

type MapLabelFilter struct {
	eventsPerValue map[string][]timeEvent

	tp MicroscopeType
}

func NewMapLabelFilter(filter MicroscopeType) *MapLabelFilter {
	return &MapLabelFilter{
		eventsPerValue: map[string][]timeEvent{},
		tp:             filter,
	}
}

func (f *MapLabelFilter) AddValue(value string, fromTS time.Time, toTS time.Time) {
	f.eventsPerValue[value] = append(
		f.eventsPerValue[value],
		timeEvent{fromTS, microscopeStart},
		timeEvent{toTS, microscopeEnd},
	)
}

// Unite intersecting segments and provide new array of events
// Example:
// [{1, microscopeStart}, {2, microscopeStart}, {3, microscopeEnd}, {4, microscopeEnd}]
// -> [{1, microscopeStart}, {4, microscopeEnd}]
func unionEvents(events []timeEvent) []timeEvent {
	startedMicroscopes := 0
	ptr := 0

	for _, event := range events {
		switch event.event {
		case microscopeStart:
			if startedMicroscopes == 0 {
				events[ptr] = event
				ptr++
			}
			startedMicroscopes++
		case microscopeEnd:
			startedMicroscopes--
			if startedMicroscopes == 0 {
				events[ptr] = event
				ptr++
			}
		}
	}

	return events[:ptr]
}

// check if point is covered by some microscope
func checkPoint(ts time.Time, events []timeEvent) bool {
	ind := sort.Search(len(events), func(i int) bool {
		return events[i].ts.After(ts)
	})
	if ind == 0 {
		return false
	}

	return events[ind-1].event == microscopeStart || (events[ind-1].ts == ts)
}

func (f *MapLabelFilter) Finalize() {
	for val, events := range f.eventsPerValue {
		sort.Slice(events, func(i, j int) bool {
			if events[i].ts != events[j].ts {
				return events[i].ts.Before(events[j].ts)
			}

			return events[i].event < events[j].event
		})

		f.eventsPerValue[val] = unionEvents(events)
	}
}

func (f *MapLabelFilter) Filter(meta *meta.ProfileMetadata) bool {
	switch f.tp {
	case PodFilter:
		return checkPoint(meta.Timestamp, f.eventsPerValue[meta.PodID])
	case NodeFilter:
		return checkPoint(meta.Timestamp, f.eventsPerValue[meta.NodeID])
	case ServiceFilter:
		return checkPoint(meta.Timestamp, f.eventsPerValue[meta.Service])
	default:
		return false
	}
}
