package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"gaze/internal/application/repository"
	"gaze/internal/config"
	"gaze/internal/logger"
)

const (
	retrievalAuditPurgeInterval     = 24 * time.Hour
	retrievalAuditPurgeStartupDelay = 10 * time.Minute
)

// RetrievalAuditRetentionRunner deletes retrieval_audit_logs older than
// the configured retention window. It mirrors AuditLogRetentionRunner:
// daily sweep, disabled when retentionDays <= 0, stoppable on shutdown.
type RetrievalAuditRetentionRunner struct {
	repo          repository.RetrievalAuditRepository
	retentionDays int
	interval      time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
	started   atomic.Bool
}

func NewRetrievalAuditRetentionRunner(
	cfg *config.Config, repo repository.RetrievalAuditRepository,
) *RetrievalAuditRetentionRunner {
	retentionDays := 30
	if cfg != nil && cfg.RetrievalAudit != nil {
		retentionDays = cfg.RetrievalAudit.RetentionDays
	}
	return &RetrievalAuditRetentionRunner{
		repo:          repo,
		retentionDays: retentionDays,
		interval:      retrievalAuditPurgeInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

func (r *RetrievalAuditRetentionRunner) Start(ctx context.Context) {
	if r == nil || r.repo == nil {
		return
	}
	r.startOnce.Do(func() {
		r.started.Store(true)
		if r.retentionDays <= 0 {
			logger.Infof(ctx, "[retrieval-audit-retention] disabled (retention_days=%d)", r.retentionDays)
			close(r.doneCh)
			return
		}
		logger.Infof(ctx, "[retrieval-audit-retention] starting daily sweep: retention_days=%d interval=%s",
			r.retentionDays, r.interval)
		go r.loop()
	})
}

func (r *RetrievalAuditRetentionRunner) Stop() {
	if r == nil || !r.started.Load() {
		return
	}
	r.stopOnce.Do(func() { close(r.stopCh) })
	<-r.doneCh
}

func (r *RetrievalAuditRetentionRunner) loop() {
	defer close(r.doneCh)
	timer := time.NewTimer(retrievalAuditPurgeStartupDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
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

func (r *RetrievalAuditRetentionRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cutoff := time.Now().Add(-time.Duration(r.retentionDays) * 24 * time.Hour)
	deleted, err := r.repo.DeleteBefore(ctx, cutoff)
	if err != nil {
		logger.Warnf(ctx, "[retrieval-audit-retention] sweep failed: retention_days=%d err=%v", r.retentionDays, err)
		return
	}
	if deleted > 0 {
		logger.Infof(ctx, "[retrieval-audit-retention] sweep complete: deleted=%d retention_days=%d", deleted, r.retentionDays)
	}
}
