package tma

import "github.com/yandex/perforator/perforator/agent/collector/pkg/profile"

// SampleTypeUnit is the unit of TMA metric sample types in profiles.
const SampleTypeUnit = "percents"

// SampleTypePrefix is the prefix of all TMA metric sample types in profiles.
const SampleTypePrefix = "tma:"

// ProfileCommentKey should be used when embedding HierarchyInfo into profile.
const ProfileCommentKey = "perforator-tma-hierarchy-info"

type HierarchyInfo struct {
	Children []*HierarchyNode `json:"children"`
}

// HierarchyNode is a node in the TMA metric tree
type HierarchyNode struct {
	DisplayName string `json:"display_name"`
	Metric      string `json:"metric"`
}

// Event is a perf event which is used to calculate TMA metrics
type Event struct {
	Name string
	// Config is a configuration for perf_event_open
	Config uint64
}

// MetricDef describes a single TMA metric
type MetricDef struct {
	SampleType string
}

// MachineProfile describes how TMA metrics should be calculated
// for a particular machine.
type MachineProfile struct {
	// Events lists the perf events needed to calculate the metrics.
	Events    []*Event
	Metrics   []*MetricDef
	Hierarchy *HierarchyInfo
}

var intelL1 = &MachineProfile{
	Metrics: []*MetricDef{
		{
			SampleType: "tma:retiring",
		},
		{
			SampleType: "tma:bad-speculation",
		},
		{
			SampleType: "tma:frontend-bound",
		},
		{
			SampleType: "tma:backend-bound",
		},
	},

	Events: []*Event{
		{
			Name:   "slots",
			Config: 0x0400,
		},
		{
			Name:   "retiring",
			Config: 0x8000,
		},
		{
			Name:   "bad-speculation",
			Config: 0x8100,
		},
		{
			Name:   "frontend-bound",
			Config: 0x8200,
		},
		{
			Name:   "backend-bound",
			Config: 0x8300,
		},
	},
	Hierarchy: &HierarchyInfo{
		Children: []*HierarchyNode{
			{
				DisplayName: "Retiring",
				Metric:      "retiring",
			},
			{
				DisplayName: "Bad speculation",
				Metric:      "bad-speculation",
			},
			{
				DisplayName: "Frontend bound",
				Metric:      "frontend-bound",
			},
			{
				DisplayName: "Backend bound",
				Metric:      "backend-bound",
			},
		},
	},
}

// Current returns MachineProfile for a local machine.
func Current() (*MachineProfile, error) {
	return intelL1, nil
}

func SampleTypes(h *HierarchyInfo) []profile.SampleType {
	res := make([]profile.SampleType, 0, len(h.Children))
	for _, node := range h.Children {
		res = append(res, profile.SampleType{
			Kind: SampleTypePrefix + node.Metric,
			Unit: SampleTypeUnit,
		})
	}
	return res
}
