package gateway

import (
	"context"
	"time"

	"github.com/cometline/cometmind/internal/jobs"
)

const jobHeartbeatInterval = 5 * time.Minute

func heartbeatJobOnce(ctx context.Context, svc *jobs.Service, sessionID string) {
	if svc == nil || sessionID == "" {
		return
	}
	job, ok, err := svc.JobForSession(ctx, sessionID)
	if err != nil || !ok {
		return
	}
	_ = svc.Heartbeat(ctx, job.ID, sessionID)
}

// startJobHeartbeatDuringTurn extends the lease while a gateway turn is running.
func startJobHeartbeatDuringTurn(ctx context.Context, svc *jobs.Service, sessionID string) func() {
	heartbeatJobOnce(ctx, svc, sessionID)
	if svc == nil || sessionID == "" {
		return func() {}
	}
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(jobHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				heartbeatJobOnce(ctx, svc, sessionID)
			}
		}
	}()
	return func() { close(stop) }
}
