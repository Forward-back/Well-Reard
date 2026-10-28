package service

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gaze/internal/application/repository"
	"gaze/internal/logger"
)

// PortalSessionRetentionRunner sweeps portal_sessions once a day, deleting
// sessions whose last activity is older than PORTAL_RETENTION_DAYS
// (§2.13 P3-7 合规留存). Same shape as AuditLogRetentionRunner: a bare
// time.Ticker goroutine — no cron, no asynq — because retention needs
// "approximately daily, eventually", not wall-clock alignment.
//
// PORTAL_RETENTION_DAYS default 90; <= 0 disables retention entirely
// (configured opt-out for personal deployments).
type PortalSessionRetentionRunner struct {
	repo          *repository.PortalSessionRepository
	retentionDays int
	interval      time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
	started   atomic.Bool
}

const portalRetentionDefaultDays = 90
const portalRetentionInterval = 24 * time.Hour
const portalRetentionStartupDelay = 10 * time.Minute

// NewPortalSessionRetentionRunner constructs the runner; nothing fires
// until Start is called.
func NewPortalSessionRetentionRunner(
	repo *repository.PortalSessionRepository,
) *PortalSessionRetentionRunner {
	days := portalRetentionDefaultDays
	if raw := strings.TrimSpace(os.Getenv("PORTAL_RETENTION_DAYS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			days = n
		}
	}
	return &PortalSessionRetentionRunner{
		repo:          repo,
		retentionDays: days,
		interval:      portalRetentionInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start spins up the background goroutine. Idempotent; retentionDays <= 0
// keeps the runner dormant.
func (r *PortalSessionRetentionRunner) Start(ctx context.Context) {
	if r == nil || r.repo == nil {
		return
	}
	r.startOnce.Do(func() {
		r.started.Store(true)
		if r.retentionDays <= 0 {
			logger.Infof(ctx,
				"[portal-retention] disabled (retention_days=%d)", r.retentionDays)
			close(r.doneCh)
			return
		}
		logger.Infof(ctx,
			"[portal-retention] starting daily sweep: retention_days=%d", r.retentionDays)
		go r.loop()
	})
}

// Stop signals the loop to exit and blocks until it returns. Idempotent;
// returns immediately if Start was never called.
func (r *PortalSessionRetentionRunner) Stop() {
	if r == nil || !r.started.Load() {
		return
	}
	r.stopOnce.Do(func() { close(r.stopCh) })
	<-r.doneCh
}

func (r *PortalSessionRetentionRunner) loop() {
	defer close(r.doneCh)

	startupTimer := time.NewTimer(portalRetentionStartupDelay)
	defer startupTimer.Stop()
	select {
	case <-startupTimer.C:
	case <-r.stopCh:
		return
	}

	r.runOnce()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.runOnce()
		case <-r.stopCh:
			return
		}
	}
}

func (r *PortalSessionRetentionRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cutoff := time.Now().AddDate(0, 0, -r.retentionDays)
	deleted, err := r.repo.DeleteSessionsOlderThan(ctx, cutoff)
	if err != nil {
		logger.Warnf(ctx,
			"[portal-retention] sweep failed: retention_days=%d err=%v",
			r.retentionDays, err)
		return
	}
	if deleted > 0 {
		logger.Infof(ctx,
			"[portal-retention] sweep complete: deleted=%d retention_days=%d",
			deleted, r.retentionDays)
	} else {
		logger.Debugf(ctx,
			"[portal-retention] sweep complete: deleted=0 retention_days=%d",
			r.retentionDays)
	}
}
