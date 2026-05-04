package profile_event

import "time"

type SignalProfileMessage struct {
	PartitionKey string
	Event        *SignalProfileEvent
}

type SignalProfileEvent struct {
	ProfileID   string    `json:"profile_id"`
	Service     string    `json:"service"`
	Cluster     string    `json:"cluster"`
	NodeID      string    `json:"node_id"`
	PodID       string    `json:"pod_id"`
	Timestamp   time.Time `json:"timestamp"`
	BuildIDs    []string  `json:"build_ids"`
	MainEvent   string    `json:"main_event"`
	SignalTypes []string  `json:"signal_types"`
}
