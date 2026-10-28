package container

import (
	"context"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// graphStub is a no-op RetrieveGraphRepository. GraphRAG was removed from the
// product (§2.6) but the chat pipeline plugins and knowledge services still
// reference the interface; the stub keeps the wiring alive until the code is
// fully stripped.
type graphStub struct{}

func (graphStub) AddGraph(context.Context, types.NameSpace, []*types.GraphData) error {
	return nil
}

func (graphStub) DelGraph(context.Context, []types.NameSpace) error {
	return nil
}

func (graphStub) SearchNode(context.Context, types.NameSpace, []string) (*types.GraphData, error) {
	return &types.GraphData{}, nil
}

var _ interfaces.RetrieveGraphRepository = graphStub{}
