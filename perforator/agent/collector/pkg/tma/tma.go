package tma

// TMASampleTypeUnit is the unit of TMA metric sample types in profiles.
const TMASampleTypeUnit = "_tma_"

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

type EvaluationRule struct {
	Numerator   string
	Denominator string
}

// MetricDef describes a single TMA metric
type MetricDef struct {
	SampleType     string
	EvaluationRule *EvaluationRule
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
			EvaluationRule: &EvaluationRule{
				Numerator:   "retiring",
				Denominator: "slots",
			},
		},
		{
			SampleType: "tma:bad-speculation",
			EvaluationRule: &EvaluationRule{
				Numerator:   "bad-speculation",
				Denominator: "slots",
			},
		},
		{
			SampleType: "tma:frontend-bound",
			EvaluationRule: &EvaluationRule{
				Numerator:   "frontend-bound",
				Denominator: "slots",
			},
		},
		{
			SampleType: "tma:backend-bound",
			EvaluationRule: &EvaluationRule{
				Numerator:   "backend-bound",
				Denominator: "slots",
			},
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
