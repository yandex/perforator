package cluster_top

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/lease"
)

func (t *ClusterTop) runSelectedJob(ctx context.Context, selected *SelectedJob, action func(context.Context, string) error) error {
	var actionErr error
	err := lease.LockAndRun(ctx, t.jobLogger(selected.Job), selected.Lease(),
		func(jobCtx context.Context, token string) {
			startedAt, err := selected.MarkRunning(jobCtx, token)
			if errors.Is(err, sql.ErrNoRows) {
				// Eligibility or ownership changed after selecting this candidate.
				// Do not compute or overwrite terminal state.
				actionErr = context.Cause(jobCtx)
				return
			}
			if err != nil {
				actionErr = errors.Join(err, context.Cause(jobCtx))
				return
			}
			selected.Job.StartedAt = startedAt
			actionErr = action(jobCtx, token)
		}, lease.WithTTL(t.leaseConf.TTL),
		lease.WithRenewInterval(t.leaseConf.HeartbeatInterval),
		lease.WithOperationTimeout(t.leaseConf.OperationTimeout),
		lease.WithReleaseTimeout(t.leaseConf.OperationTimeout),
		lease.WithWaitForLease(false), lease.WithMaxAcquireRetryAttempts(0),
		lease.WithRenewalErrors(t.metrics.heartbeatErrors))
	if err == nil {
		err = actionErr
	}
	if errors.Is(err, ErrJobLeaseLost) {
		t.metrics.leaseLost.Inc()
		t.l.Warn(ctx, "Lost job lease", log.Int64("job_id", selected.Job.ID))
	}
	return err
}
