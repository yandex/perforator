package cluster_top

import "time"

const ExecutionStatsVersionV1 = "v1"

type JobExecutionStats struct {
	Version        string        `json:"version"`
	Duration       time.Duration `json:"duration"`
	QueueWait      time.Duration `json:"queue_wait,omitempty"`
	ProcessingWall time.Duration `json:"processing_wall,omitempty"`
	Stages         JobStages     `json:"stages"`
	Metrics        JobMetrics    `json:"metrics"`
	Error          string        `json:"error,omitempty"`
}

type JobStages struct {
	QueryMetadata     time.Duration `json:"query_metadata"`
	DownloadGsym      time.Duration `json:"download_gsym"`
	FetchProfile      time.Duration `json:"fetch_profile"`
	ParseProfile      time.Duration `json:"parse_profile"`
	AggregateProfiles time.Duration `json:"aggregate_profiles"`
	MergeExtract      time.Duration `json:"merge_extract"`
	SaveTop           time.Duration `json:"save_top"`
}

type JobMetrics struct {
	ProfilesDiscovered int `json:"profiles_discovered"`
	ProfilesProcessed  int `json:"profiles_processed"`
	BuildIDs           int `json:"build_ids"`
	Functions          int `json:"functions"`
	Batches            int `json:"batches"`
}

func newJobExecutionStats() *JobExecutionStats {
	return &JobExecutionStats{
		Version: ExecutionStatsVersionV1,
	}
}
