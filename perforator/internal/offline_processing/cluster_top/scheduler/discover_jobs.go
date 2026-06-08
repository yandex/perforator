package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/yandex/perforator/perforator/pkg/sampletype"
)

// If any profile in the group has pod_id, the job is pod-scoped; otherwise it is a host-agent job (node_id only).
const discoverJobsQueryTemplate = `
SELECT
    service,
    if(pod_profiles_count > 0, workload_key, '') AS pod_id,
    if(pod_profiles_count > 0, '', workload_key) AS node_id,
    profiles_count
FROM (
    SELECT
        service,
        workload_key,
        count() AS profiles_count,
        countIf(pod_id != '') AS pod_profiles_count
    FROM (
        SELECT
            service,
            pod_id,
            coalesce(nullIf(pod_id, ''), node_id) AS workload_key
        FROM profiles
        WHERE system_name = '%s'
          AND event_type = '%s'
          AND %s
          AND timestamp >= ?
          AND timestamp < ?
    )
    GROUP BY service, workload_key
)
`

var discoverJobsQuery = fmt.Sprintf(
	discoverJobsQueryTemplate,
	clusterTopSystemName,
	sampletype.SampleTypeCPUCycles,
	discoverJobsProfileFiltersWhere,
)

type discoveredJob struct {
	service       string
	podID         string
	nodeID        string
	profilesCount uint64
}

func (s *Scheduler) discoverJobs(ctx context.Context, start, end time.Time) ([]discoveredJob, error) {
	rows, err := s.storage.DBs.ClickhouseConn.Query(ctx, discoverJobsQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to discover jobs: %w", err)
	}
	defer rows.Close()

	var jobs []discoveredJob
	for rows.Next() {
		var job discoveredJob
		if err := rows.Scan(
			&job.service,
			&job.podID,
			&job.nodeID,
			&job.profilesCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan discovered job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read discovered jobs: %w", err)
	}

	return jobs, nil
}

func countUniqueServices(jobs []discoveredJob) int {
	seen := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		seen[job.service] = struct{}{}
	}
	return len(seen)
}
