package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gaze/internal/application/access"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gazeGuardTaskQueue is a TaskEnqueuer stub for tests that only need the
// enqueue call to succeed.
type gazeGuardTaskQueue struct{}

func (q *gazeGuardTaskQueue) Enqueue(_ *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	return &asynq.TaskInfo{ID: "t1"}, nil
}

// Moving a document out of a gaze-enabled KB must reconcile that KB's gaze
// state the same way a delete does. gaze_pages hold source_refs back to the
// knowledge and are what both the folder tree and the gaze graph are rendered
// from, so skipping the cleanup leaves the source KB showing pages for a
// document it no longer owns.

type moveGazeKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	knowledge *types.Knowledge
	commitErr error
}

func (r *moveGazeKnowledgeRepo) GetKnowledgeByID(
	_ context.Context, _ uint64, _ string,
) (*types.Knowledge, error) {
	clone := *r.knowledge
	return &clone, nil
}

func (r *moveGazeKnowledgeRepo) UpdateKnowledge(_ context.Context, k *types.Knowledge) error {
	clone := *k
	r.knowledge = &clone
	return nil
}

func (r *moveGazeKnowledgeRepo) DeleteKnowledgeTagRelations(_ context.Context, _ string) error {
	return nil
}

type moveGazePageRepo struct {
	interfaces.GazePageRepository
	listedKBs  []string
	listErr    error
	beforeList func()
	pages      []*types.GazePage
}

func (r *moveGazePageRepo) ListBySourceRef(
	_ context.Context, kbID, _ string,
) ([]*types.GazePage, error) {
	r.listedKBs = append(r.listedKBs, kbID)
	if r.beforeList != nil {
		r.beforeList()
	}
	return r.pages, r.listErr
}

type moveGazePendingRepo struct {
	interfaces.TaskPendingOpsRepository
	ops    []*types.TaskPendingOp
	err    error
	failKB string
}

func (r *moveGazePendingRepo) Enqueue(_ context.Context, op *types.TaskPendingOp) error {
	if r.err != nil && (r.failKB == "" || r.failKB == op.ScopeID) {
		return r.err
	}
	r.ops = append(r.ops, op)
	return nil
}

func (r *moveGazePendingRepo) DeleteByDedupKey(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

type moveGazeChunkRepo struct {
	interfaces.ChunkRepository
	movedToKB string
	moves     int
	moveErr   error
}

func (r *moveGazeChunkRepo) ListChunksByKnowledgeID(
	_ context.Context, _ uint64, _ string,
) ([]*types.Chunk, error) {
	return nil, nil
}

func (r *moveGazeChunkRepo) MoveChunksByKnowledgeID(
	_ context.Context, _ uint64, _ string, targetKBID string,
) error {
	r.moves++
	if r.moveErr != nil {
		return r.moveErr
	}
	r.movedToKB = targetKBID
	return nil
}

func gazeEnabledKB(id string) *types.KnowledgeBase {
	return &types.KnowledgeBase{
		ID:               id,
		TenantID:         1,
		IndexingStrategy: types.IndexingStrategy{GazeEnabled: true},
	}
}

// opsFor returns the pending ops enqueued against a given KB scope.
func opsFor(ops []*types.TaskPendingOp, kbID string) []*types.TaskPendingOp {
	var out []*types.TaskPendingOp
	for _, op := range ops {
		if op.ScopeID == kbID {
			out = append(out, op)
		}
	}
	return out
}

func newMoveGazeService(t *testing.T) (
	*knowledgeService, *moveGazePageRepo, *moveGazePendingRepo, *moveGazeChunkRepo,
) {
	t.Helper()
	gazeRepo := &moveGazePageRepo{}
	pendingRepo := &moveGazePendingRepo{}
	chunkRepo := &moveGazeChunkRepo{}
	svc := &knowledgeService{
		repo: &moveGazeKnowledgeRepo{knowledge: &types.Knowledge{
			ID:              "kn-1",
			TenantID:        1,
			Title:           "Doc",
			KnowledgeBaseID: "kb-src",
			ParseStatus:     types.ParseStatusCompleted,
		}},
		gazeRepo:        gazeRepo,
		taskPendingRepo: pendingRepo,
		task:            &gazeGuardTaskQueue{},
		chunkRepo:       chunkRepo,
	}
	return svc, gazeRepo, pendingRepo, chunkRepo
}

func moveGazeCtx() context.Context {
	ctx, _ := access.WithKBTransferTask(
		context.Background(),
		gazeEnabledKB("kb-src"),
		gazeEnabledKB("kb-dst"),
		1,
		access.KBTransferMove,
		"move-task",
		false,
	)
	return ctx
}

func TestMoveOneKnowledgeRejectsUnknownModeBeforeGazeCleanup(t *testing.T) {
	svc, gazeRepo, pendingRepo, _ := newMoveGazeService(t)
	err := svc.moveOneKnowledge(moveGazeCtx(), "kn-1", gazeEnabledKB("kb-src"), gazeEnabledKB("kb-dst"), "bogus")
	require.ErrorContains(t, err, "unknown move mode")
	require.Empty(t, gazeRepo.listedKBs)
	require.Empty(t, pendingRepo.ops)
	require.Equal(t, types.ParseStatusCompleted, svc.repo.(*moveGazeKnowledgeRepo).knowledge.ParseStatus)
}

func TestMoveOneKnowledgeReuseVectorsIngestsIntoTargetKB(t *testing.T) {
	// reuse_vectors keeps the existing chunks and never re-enters the parse
	// pipeline, so nothing else would tell the target KB to build gaze pages.
	svc, _, pendingRepo, chunkRepo := newMoveGazeService(t)

	err := svc.moveOneKnowledge(moveGazeCtx(), "kn-1",
		gazeEnabledKB("kb-src"), gazeEnabledKB("kb-dst"), "reuse_vectors")

	require.NoError(t, err)
	assert.Equal(t, "kb-dst", chunkRepo.movedToKB)

	dstOps := opsFor(pendingRepo.ops, "kb-dst")
	require.Len(t, dstOps, 1)
	assert.Equal(t, GazeOpIngest, dstOps[0].Op)
	assert.Equal(t, "kn-1", dstOps[0].DedupKey)
}

func TestMoveOneKnowledgeSkipsGazeWorkForNonGazeKBs(t *testing.T) {
	svc, gazeRepo, pendingRepo, _ := newMoveGazeService(t)

	err := svc.moveOneKnowledge(
		moveGazeCtx(),
		"kn-1",
		&types.KnowledgeBase{
			ID:       "kb-src",
			TenantID: 1,
		},
		&types.KnowledgeBase{ID: "kb-dst", TenantID: 1},
		"reuse_vectors",
	)

	require.NoError(t, err)
	assert.Empty(t, gazeRepo.listedKBs)
	assert.Empty(t, pendingRepo.ops)
}

func (r *moveGazeKnowledgeRepo) UpdateKnowledgeForTransfer(
	ctx context.Context,
	_ *types.Knowledge,
	after *types.Knowledge,
) error {
	if after.KnowledgeBaseID == "kb-dst" && r.commitErr != nil {
		return r.commitErr
	}
	return r.UpdateKnowledge(ctx, after)
}

func (r *moveGazeChunkRepo) ListAllChunksByKnowledgeID(context.Context, uint64, string) ([]*types.Chunk, error) {
	return nil, nil
}

type moveGazeMetaService struct {
	interfaces.GazePageService
	updated []*types.GazePage
}

func (s *moveGazeMetaService) UpdatePageMeta(_ context.Context, page *types.GazePage) error {
	s.updated = append(s.updated, page)
	return nil
}

func TestMoveGazeCleanupRunsOnlyAfterOwnershipCommit(t *testing.T) {
	for _, stage := range []string{"chunks", "CAS"} {
		t.Run(stage, func(t *testing.T) {
			svc, gaze, pending, chunks := newMoveGazeService(t)
			repo := svc.repo.(*moveGazeKnowledgeRepo)
			failure := errors.New("transfer failed")
			if stage == "chunks" {
				chunks.moveErr = failure
			} else {
				repo.commitErr = failure
			}
			err := svc.moveOneKnowledge(
				moveGazeCtx(),
				"kn-1",
				gazeEnabledKB("kb-src"),
				gazeEnabledKB("kb-dst"),
				"reuse_vectors",
			)
			require.ErrorIs(t, err, failure)
			require.Equal(t, "kb-src", repo.knowledge.KnowledgeBaseID)
			require.Empty(t, gaze.listedKBs)
			require.Empty(t, pending.ops)
		})
	}
}

func TestMoveGazeCleanupFailureRetriesWithoutMovingAgain(t *testing.T) {
	for _, stage := range []string{"lookup", "enqueue", "target enqueue"} {
		t.Run(stage, func(t *testing.T) {
			svc, gaze, pending, chunks := newMoveGazeService(t)
			repo := svc.repo.(*moveGazeKnowledgeRepo)
			failure := errors.New("gaze unavailable")
			if stage == "lookup" {
				gaze.listErr = failure
			} else {
				pending.err = failure
				if stage == "target enqueue" {
					pending.failKB = "kb-dst"
				}
			}
			gaze.beforeList = func() { require.Equal(t, "kb-dst", repo.knowledge.KnowledgeBaseID) }
			err := svc.moveOneKnowledge(
				moveGazeCtx(),
				"kn-1",
				gazeEnabledKB("kb-src"),
				gazeEnabledKB("kb-dst"),
				"reuse_vectors",
			)
			require.ErrorIs(t, err, failure)
			require.Equal(t, "kb-dst", repo.knowledge.KnowledgeBaseID)
			require.Equal(t, []string{"kb-src"}, gaze.listedKBs)
			require.Empty(t, opsFor(pending.ops, "kb-dst"))
			gaze.listErr = nil
			pending.err = nil
			require.NoError(
				t,
				svc.moveOneKnowledge(
					moveGazeCtx(),
					"kn-1",
					gazeEnabledKB("kb-src"),
					gazeEnabledKB("kb-dst"),
					"reuse_vectors",
				),
			)
			require.Equal(t, 1, chunks.moves)
			require.Equal(t, []string{"kb-src", "kb-src"}, gaze.listedKBs)
			require.NotEmpty(t, opsFor(pending.ops, "kb-dst"))
		})
	}
}

func TestMoveReparseGazeUsesDurableSourceRefsAfterChunksDeleted(t *testing.T) {
	f := transferFixture(t, access.KBTransferMove)
	f.kbs.values["kb"].IndexingStrategy.GazeEnabled = true
	row, err := f.repo.GetKnowledgeByID(f.ctx, 7, "doc")
	require.NoError(t, err)
	require.NoError(
		t,
		row.SetManualMetadata(types.NewManualKnowledgeMetadata("# content", types.ManualKnowledgeStatusPublish, 1)),
	)
	row.Description = "source summary"
	require.NoError(t, f.repo.UpdateKnowledge(f.ctx, row))
	page := &types.GazePage{
		ID:              "page",
		KnowledgeBaseID: "kb",
		SourceRefs:      types.StringArray{"doc", "other-doc"},
		ChunkRefs:       types.StringArray{"chunk", "other-chunk"},
	}
	gaze := &moveGazePageRepo{pages: []*types.GazePage{page}}
	meta := &moveGazeMetaService{}
	pending := &moveGazePendingRepo{}
	gaze.beforeList = func() {
		saved, err := f.repo.GetKnowledgeByID(f.ctx, 7, "doc")
		require.NoError(t, err)
		require.Equal(t, "other", saved.KnowledgeBaseID)
		chunks, err := f.chunkRepo.ListAllChunksByKnowledgeID(f.ctx, 7, "doc")
		require.NoError(t, err)
		require.Empty(t, chunks)
	}
	f.svc.gazeRepo = gaze
	f.svc.gazeService = meta
	f.svc.taskPendingRepo = pending
	f.svc.task = &transferQueue{}
	require.NoError(t, f.svc.ProcessKnowledgeMove(context.Background(), moveTask(t, []string{"doc"}, "reparse")))
	require.Equal(t, []string{"kb"}, gaze.listedKBs)
	require.Len(t, meta.updated, 1)
	require.Equal(t, types.StringArray{"other-doc"}, meta.updated[0].SourceRefs)
	require.Equal(t, types.StringArray{"other-chunk"}, meta.updated[0].ChunkRefs)
	ops := opsFor(pending.ops, "kb")
	require.Len(t, ops, 1)
	var payload GazePendingOp
	require.NoError(t, json.Unmarshal(ops[0].Payload, &payload))
	require.Equal(t, "source summary", payload.DocSummary)
}

func TestMoveGazeRetractionIsDurableBeforeRemovingPageSources(t *testing.T) {
	svc, gaze, pending, chunks := newMoveGazeService(t)
	page := &types.GazePage{ID: "page", Slug: "concept/source", SourceRefs: types.StringArray{"kn-1", "other"}}
	gaze.pages = []*types.GazePage{page}
	meta := &moveGazeMetaService{}
	svc.gazeService = meta
	pending.err = errors.New("enqueue unavailable")
	err := svc.moveOneKnowledge(
		moveGazeCtx(),
		"kn-1",
		gazeEnabledKB("kb-src"),
		gazeEnabledKB("kb-dst"),
		"reuse_vectors",
	)
	require.ErrorIs(t, err, pending.err)
	require.Empty(t, meta.updated)
	require.Contains(t, page.SourceRefs, "kn-1")
	pending.err = nil
	require.NoError(
		t,
		svc.moveOneKnowledge(moveGazeCtx(), "kn-1", gazeEnabledKB("kb-src"), gazeEnabledKB("kb-dst"), "reuse_vectors"),
	)
	require.Equal(t, 1, chunks.moves)
	require.Len(t, meta.updated, 1)
	var op GazePendingOp
	require.NoError(t, json.Unmarshal(opsFor(pending.ops, "kb-src")[0].Payload, &op))
	require.Equal(t, []string{"concept/source"}, op.PageSlugs)
}
