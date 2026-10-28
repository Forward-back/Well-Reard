package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"gaze/internal/engine"
	"gaze/internal/types"
)

type fakeRetrievalAuditRepo struct {
	mu      sync.Mutex
	batches [][]*types.RetrievalAuditLog
	cutoff  time.Time
	deleted int64
	notify  chan int
}

func (f *fakeRetrievalAuditRepo) Create(context.Context, *types.RetrievalAuditLog) error { return nil }

func (f *fakeRetrievalAuditRepo) CreateBatch(_ context.Context, entries []*types.RetrievalAuditLog) error {
	f.mu.Lock()
	f.batches = append(f.batches, entries)
	f.mu.Unlock()
	if f.notify != nil {
		f.notify <- len(entries)
	}
	return nil
}

func (f *fakeRetrievalAuditRepo) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	f.cutoff = cutoff
	f.mu.Unlock()
	return f.deleted, nil
}

func TestRetrievalAuditBatchesOnSize(t *testing.T) {
	repo := &fakeRetrievalAuditRepo{notify: make(chan int, 1)}
	svc := newRetrievalAuditService(repo, 2, time.Hour, 10)

	svc.LogRetrieval(context.Background(), engine.RetrievalAuditEvent{TenantID: 1, Query: "a"})
	svc.LogRetrieval(context.Background(), engine.RetrievalAuditEvent{TenantID: 1, Query: "b"})

	select {
	case n := <-repo.notify:
		if n != 2 {
			t.Fatalf("batch size = %d, want 2", n)
		}
	case <-time.After(time.Second):
		t.Fatal("batch was not flushed")
	}
}

func TestRetrievalAuditFlushesOnTimer(t *testing.T) {
	repo := &fakeRetrievalAuditRepo{notify: make(chan int, 1)}
	svc := newRetrievalAuditService(repo, 50, 10*time.Millisecond, 10)

	svc.LogRetrieval(context.Background(), engine.RetrievalAuditEvent{TenantID: 1, Query: "a"})

	select {
	case n := <-repo.notify:
		if n != 1 {
			t.Fatalf("batch size = %d, want 1", n)
		}
	case <-time.After(time.Second):
		t.Fatal("timer flush did not run")
	}
}

func TestRetrievalAuditDropsWhenBufferFull(t *testing.T) {
	repo := &fakeRetrievalAuditRepo{notify: make(chan int, 1)}
	svc := newRetrievalAuditService(repo, 50, 10*time.Millisecond, 1)

	svc.LogRetrieval(context.Background(), engine.RetrievalAuditEvent{TenantID: 1, Query: "a"})
	svc.LogRetrieval(context.Background(), engine.RetrievalAuditEvent{TenantID: 1, Query: "b"})

	select {
	case n := <-repo.notify:
		if n != 1 {
			t.Fatalf("batch size = %d, want 1 (second event dropped)", n)
		}
	case <-time.After(time.Second):
		t.Fatal("timer flush did not run")
	}
}

func TestRetrievalAuditRetentionUsesCutoff(t *testing.T) {
	repo := &fakeRetrievalAuditRepo{deleted: 4}
	runner := &RetrievalAuditRetentionRunner{repo: repo, retentionDays: 30}

	runner.runOnce()

	if repo.deleted != 4 {
		t.Fatalf("deleted = %d, want 4", repo.deleted)
	}
	if delta := time.Since(repo.cutoff); delta < 29*24*time.Hour || delta > 31*24*time.Hour {
		t.Fatalf("cutoff delta = %s, want about 30 days", delta)
	}
}
