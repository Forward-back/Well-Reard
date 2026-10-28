package service

import (
	"context"
	"os"
	"strings"
	"sync"

	"gaze/internal/engine"
	"gaze/internal/logger"
	"gaze/internal/types/interfaces"
	"github.com/robfig/cron/v3"
)

// YLSWReconcileService runs the daily dual-write reconciliation (§2.2 ③):
// every day at 03:00 it diffs app-side knowledges against YLSW
// GET /documents/ids — missing documents are re-pushed (convert → push),
// stale ones are deleted from the engine. Both sides treat the operations as
// idempotent/no-op, so a failed run is simply retried on the next sweep.
type YLSWReconcileService struct {
	mu          sync.Mutex
	started     bool
	reconciling bool
	cron        *cron.Cron
	svc         interfaces.KnowledgeService
	engine      engine.RAGEngine
}

// NewYLSWReconcileService creates the reconciliation service.
func NewYLSWReconcileService(
	svc interfaces.KnowledgeService, eng engine.RAGEngine,
) *YLSWReconcileService {
	return &YLSWReconcileService{
		svc:    svc,
		engine: eng,
		cron: cron.New(cron.WithSeconds(), cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
	}
}

// Start registers the daily 03:00 sweep. Idempotent; disabled via
// GAZE_YLSW_RECONCILE_ENABLED=0/false/off/no.
func (r *YLSWReconcileService) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	if !ylswReconcileEnabled() {
		logger.Infof(ctx, "[YLSWReconcile] disabled via GAZE_YLSW_RECONCILE_ENABLED=false")
		return nil
	}
	if _, err := r.cron.AddFunc("0 0 3 * * *", func() {
		r.reconcileOnce(context.Background())
	}); err != nil {
		return err
	}
	r.cron.Start()
	r.started = true
	logger.Infof(ctx, "[YLSWReconcile] started (daily 03:00)")
	return nil
}

// Stop halts the cron scheduler. Idempotent.
func (r *YLSWReconcileService) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		return
	}
	c := r.cron.Stop()
	<-c.Done()
	r.started = false
}

// ReconcileOnce triggers a manual full diff sweep (Admin console endpoint).
// Runs asynchronously with a background context so the HTTP request never
// blocks on document conversion; concurrent triggers while a sweep is in
// flight are rejected (returns false).
func (r *YLSWReconcileService) ReconcileOnce(ctx context.Context) bool {
	if r.svc == nil || r.engine == nil {
		return false
	}
	r.mu.Lock()
	if r.reconciling {
		r.mu.Unlock()
		return false
	}
	r.reconciling = true
	r.mu.Unlock()
	go func() {
		defer func() {
			r.mu.Lock()
			r.reconciling = false
			r.mu.Unlock()
		}()
		r.reconcileOnce(context.Background())
	}()
	logger.Infof(ctx, "[YLSWReconcile] manual trigger accepted")
	return true
}

// reconcileOnce performs one full diff. All failures are logged and swallowed:
// the next daily sweep (or the asynq retry task for fresh pushes) covers them.
func (r *YLSWReconcileService) reconcileOnce(ctx context.Context) {
	if r.svc == nil || r.engine == nil {
		return
	}

	ylswIDs, err := r.engine.ListDocumentIDs(ctx)
	if err != nil {
		logger.Warnf(ctx, "[YLSWReconcile] list engine documents failed: %v", err)
		return
	}
	repo := r.svc.GetRepository()
	if repo == nil {
		return
	}
	rows, err := repo.ListKnowledgeIDs(ctx)
	if err != nil {
		logger.Warnf(ctx, "[YLSWReconcile] list app knowledges failed: %v", err)
		return
	}

	ylswSet := make(map[string]struct{}, len(ylswIDs))
	for _, id := range ylswIDs {
		ylswSet[id] = struct{}{}
	}
	appSet := make(map[string]struct{}, len(rows))
	for _, k := range rows {
		if k != nil {
			appSet[k.ID] = struct{}{}
		}
	}

	// 差集一:app 有、引擎无 → 重推(非 completed 由 RetryEnginePush 自行跳过)。
	var rePushed int
	for _, k := range rows {
		if k == nil {
			continue
		}
		if _, ok := ylswSet[k.ID]; ok {
			continue
		}
		if err := r.svc.RetryEnginePush(ctx, k.TenantID, k.ID, k.KnowledgeBaseID); err != nil {
			logger.Warnf(ctx, "[YLSWReconcile] re-push knowledge %s failed: %v", k.ID, err)
			continue
		}
		rePushed++
	}

	// 差集二:引擎有、app 无 → 删(删除链路失败残留,no-op 语义已确认)。
	var deleted int
	for _, id := range ylswIDs {
		if _, ok := appSet[id]; ok {
			continue
		}
		if err := r.engine.DeleteDocument(ctx, id); err != nil {
			logger.Warnf(ctx, "[YLSWReconcile] delete stale engine document %s failed: %v", id, err)
			continue
		}
		deleted++
	}

	if rePushed > 0 || deleted > 0 {
		logger.Infof(ctx, "[YLSWReconcile] done: re-pushed %d, deleted %d stale", rePushed, deleted)
	}
}

// ylswReconcileEnabled mirrors housekeepingEnabled(): default on, opt-out via
// GAZE_YLSW_RECONCILE_ENABLED=0/false/off/no.
func ylswReconcileEnabled() bool {
	v := strings.TrimSpace(os.Getenv("GAZE_YLSW_RECONCILE_ENABLED"))
	if v == "" {
		return true
	}
	switch strings.ToLower(v) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}
