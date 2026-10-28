package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gazeEnqueueFailureKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	knowledge        *types.Knowledge
	expectedSubtasks int
}

func (r *gazeEnqueueFailureKnowledgeRepo) GetKnowledgeByIDOnly(
	context.Context,
	string,
) (*types.Knowledge, error) {
	return r.knowledge, nil
}

func (r *gazeEnqueueFailureKnowledgeRepo) SetFinalizing(
	_ context.Context,
	_ string,
	expectedSubtasks int,
) (bool, error) {
	r.expectedSubtasks = expectedSubtasks
	r.knowledge.ParseStatus = types.ParseStatusFinalizing
	return true, nil
}

func (r *gazeEnqueueFailureKnowledgeRepo) FinalizeSubtask(
	context.Context,
	string,
) (int, bool, error) {
	return 0, false, nil
}

func (r *gazeEnqueueFailureKnowledgeRepo) UpdateKnowledgeColumn(
	context.Context,
	string,
	string,
	interface{},
) error {
	return nil
}

type gazeEnqueueFailureKBService struct {
	interfaces.KnowledgeBaseService
	kb *types.KnowledgeBase
}

func (s *gazeEnqueueFailureKBService) GetKnowledgeBaseByIDOnly(
	context.Context,
	string,
) (*types.KnowledgeBase, error) {
	return s.kb, nil
}

type gazeEnqueueFailureChunkService struct {
	interfaces.ChunkService
	chunks []*types.Chunk
}

func (s *gazeEnqueueFailureChunkService) ListChunksByKnowledgeID(
	context.Context,
	string,
) ([]*types.Chunk, error) {
	return s.chunks, nil
}

type gazeEnqueueFailureTaskQueue struct {
	interfaces.TaskEnqueuer
	taskTypes       []string
	extractChunkIDs []string
	gazeErr         error
}

func (q *gazeEnqueueFailureTaskQueue) Enqueue(
	task *asynq.Task,
	_ ...asynq.Option,
) (*asynq.TaskInfo, error) {
	q.taskTypes = append(q.taskTypes, task.Type())
	if task.Type() == types.TypeGazeIngest {
		return nil, q.gazeErr
	}
	if task.Type() == types.TypeChunkExtract {
		var payload types.ExtractChunkPayload
		if err := json.Unmarshal(task.Payload(), &payload); err == nil {
			q.extractChunkIDs = append(q.extractChunkIDs, payload.ChunkID)
		}
	}
	return &asynq.TaskInfo{ID: "queued", Type: task.Type()}, nil
}

type gazeEnqueueFailurePendingRepo struct {
	interfaces.TaskPendingOpsRepository
	seedErr       error
	seedCalls     int
	seededOp      *types.TaskPendingOp
	knowledgeRepo *gazeEnqueueFailureKnowledgeRepo
}

func (r *gazeEnqueueFailurePendingRepo) SeedKnowledgeFinalizingWithPendingOp(
	_ context.Context,
	_ string,
	expectedSubtasks int,
	op *types.TaskPendingOp,
) (bool, error) {
	r.seedCalls++
	if r.seedErr != nil {
		return false, r.seedErr
	}
	r.seededOp = op
	r.knowledgeRepo.expectedSubtasks = expectedSubtasks
	r.knowledgeRepo.knowledge.ParseStatus = types.ParseStatusFinalizing
	return true, nil
}

type gazeEnqueueFailureChunkRepo struct {
	interfaces.ChunkRepository
	chunks []*types.Chunk
}

func (r *gazeEnqueueFailureChunkRepo) ListChunksByKnowledgeIDAndTypes(
	_ context.Context,
	_ uint64,
	_ string,
	chunkTypes []types.ChunkType,
) ([]*types.Chunk, error) {
	allowed := make(map[types.ChunkType]struct{}, len(chunkTypes))
	for _, ct := range chunkTypes {
		allowed[ct] = struct{}{}
	}
	var out []*types.Chunk
	for _, c := range r.chunks {
		if _, ok := allowed[c.ChunkType]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func newGazeEnqueueTestService(
	knowledgeID string,
	pendingRepo *gazeEnqueueFailurePendingRepo,
	queue *gazeEnqueueFailureTaskQueue,
) (*KnowledgePostProcessService, *gazeEnqueueFailureKnowledgeRepo) {
	repo := &gazeEnqueueFailureKnowledgeRepo{
		knowledge: &types.Knowledge{
			ID:              knowledgeID,
			TenantID:        7,
			KnowledgeBaseID: "kb-gaze",
			ParseStatus:     types.ParseStatusProcessing,
		},
	}
	pendingRepo.knowledgeRepo = repo
	return &KnowledgePostProcessService{
		knowledgeRepo: repo,
		kbService: &gazeEnqueueFailureKBService{kb: &types.KnowledgeBase{
			ID:       "kb-gaze",
			TenantID: 7,
			IndexingStrategy: types.IndexingStrategy{
				GazeEnabled: true,
			},
		}},
		chunkService: &gazeEnqueueFailureChunkService{chunks: []*types.Chunk{
			{
				ID:              "chunk-1",
				TenantID:        7,
				KnowledgeID:     knowledgeID,
				KnowledgeBaseID: "kb-gaze",
				ChunkType:       types.ChunkTypeText,
			},
		}},
		chunkRepo: &gazeEnqueueFailureChunkRepo{chunks: []*types.Chunk{
			{
				ID:              "chunk-1",
				TenantID:        7,
				KnowledgeID:     knowledgeID,
				KnowledgeBaseID: "kb-gaze",
				ChunkType:       types.ChunkTypeText,
				Content:         "plain text",
			},
		}},
		taskEnqueuer: queue,
		pendingRepo:  pendingRepo,
	}, repo
}

func newGazeEnqueuePostProcessTask(t *testing.T, knowledgeID string) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(types.KnowledgePostProcessPayload{
		TenantID:        7,
		KnowledgeID:     knowledgeID,
		KnowledgeBaseID: "kb-gaze",
	})
	require.NoError(t, err)
	return asynq.NewTask(types.TypeKnowledgePostProcess, payload)
}

func TestKnowledgePostProcessAtomicallySeedsGazeSlot(t *testing.T) {
	const knowledgeID = "knowledge-gaze-enqueue-failure"
	tests := []struct {
		name       string
		pendingErr error
		wantTasks  []string
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "pending op persistence fails",
			pendingErr: errors.New("postgres unavailable"),
			wantStatus: types.ParseStatusProcessing,
			wantErr:    true,
		},
		{
			name:       "pending op and trigger succeed",
			wantTasks:  []string{types.TypeSummaryGeneration, types.TypeGazeIngest},
			wantStatus: types.ParseStatusFinalizing,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pendingRepo := &gazeEnqueueFailurePendingRepo{seedErr: test.pendingErr}
			queue := &gazeEnqueueFailureTaskQueue{}
			service, repo := newGazeEnqueueTestService(knowledgeID, pendingRepo, queue)

			err := service.Handle(context.Background(), newGazeEnqueuePostProcessTask(t, knowledgeID))

			if test.wantErr {
				require.ErrorIs(t, err, test.pendingErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2, repo.expectedSubtasks, "summary and gaze each seed one slot")
				require.NotNil(t, pendingRepo.seededOp)
				assert.Equal(t, knowledgeID, pendingRepo.seededOp.DedupKey)
			}
			assert.Equal(t, test.wantStatus, repo.knowledge.ParseStatus)
			assert.Equal(t, test.wantTasks, queue.taskTypes)
			assert.Equal(t, 1, pendingRepo.seedCalls)
		})
	}
}

func TestKnowledgePostProcessRetriesGazeTriggerWithoutDoubleAccounting(t *testing.T) {
	const knowledgeID = "knowledge-gaze-trigger-retry"
	gazeErr := errors.New("redis unavailable")
	pendingRepo := &gazeEnqueueFailurePendingRepo{}
	queue := &gazeEnqueueFailureTaskQueue{gazeErr: gazeErr}
	service, repo := newGazeEnqueueTestService(knowledgeID, pendingRepo, queue)
	task := newGazeEnqueuePostProcessTask(t, knowledgeID)

	err := service.Handle(context.Background(), task)

	require.ErrorIs(t, err, gazeErr)
	assert.Equal(t, 2, repo.expectedSubtasks, "summary and gaze each seed one slot")
	assert.Equal(t, 1, pendingRepo.seedCalls)
	assert.Equal(t, []string{types.TypeSummaryGeneration, types.TypeGazeIngest}, queue.taskTypes)

	queue.gazeErr = nil
	err = service.Handle(context.Background(), task)

	require.NoError(t, err)
	assert.Equal(t, 1, pendingRepo.seedCalls, "retry must not append another pending op")
	assert.Equal(t,
		[]string{types.TypeSummaryGeneration, types.TypeGazeIngest, types.TypeGazeIngest},
		queue.taskTypes,
	)
}

func TestPostProcessRejectsMovedKnowledgeBeforeGazeOrSpanWrites(t *testing.T) {
	pending := &gazeEnqueueFailurePendingRepo{}
	queue := &gazeEnqueueFailureTaskQueue{}
	svc, repo := newGazeEnqueueTestService("moved-doc", pending, queue)
	repo.knowledge.KnowledgeBaseID = "other-kb"
	require.ErrorIs(t, svc.Handle(context.Background(), newGazeEnqueuePostProcessTask(t, "moved-doc")), asynq.SkipRetry)
	require.Zero(t, repo.expectedSubtasks)
	require.Equal(t, types.ParseStatusProcessing, repo.knowledge.ParseStatus)
}
