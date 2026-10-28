package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type gazeDeletePageTool struct {
	BaseTool
	gazePageService interfaces.GazePageService
	kbIDs           []string
	routes          *GazeRouteResolver
}

// NewGazeDeletePageTool creates a new gaze_delete_page tool
func NewGazeDeletePageTool(
	gazePageService interfaces.GazePageService,
	kbIDs []string,
	routes ...*GazeRouteResolver,
) types.Tool {
	return &gazeDeletePageTool{
		BaseTool: NewBaseTool(
			ToolGazeDeletePage,
			"Delete a Gaze page. Automatically cleans up incoming links on other pages to prevent dead links.",
			json.RawMessage(`{
				"type": "object",
				"properties": {
					"slug": {
						"type": "string",
						"description": "The slug of the Gaze page to delete"
					}
				},
				"required": ["slug"]
			}`),
		),
		gazePageService: gazePageService,
		kbIDs:           kbIDs,
		routes:          firstGazeRoute(routes),
	}
}

func (t *gazeDeletePageTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	// Attribute every page write performed by this tool to the agent so
	// revision history distinguishes agent edits from pipeline/user ones.
	ctx = types.WithGazeEditSource(ctx, types.GazeEditSourceAgent)
	var params struct {
		Slug string `json:"slug"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return &types.ToolResult{Success: false, Error: "Failed to parse arguments: " + err.Error()}, nil
	}

	if len(t.kbIDs) == 0 {
		return &types.ToolResult{Success: false, Error: "No knowledge bases available for editing"}, nil
	}
	if params.Slug == "" {
		return &types.ToolResult{Success: false, Error: "slug is required"}, nil
	}
	normalizedSlug, slugErr := normalizeAndValidateGazeSlug(params.Slug)
	if slugErr != nil {
		return &types.ToolResult{Success: false, Error: slugErr.Error()}, nil
	}
	params.Slug = normalizedSlug

	// Fetch page to get its incoming links before deleting
	existingPage, kbID, err := resolveUniqueGazePage(ctx, t.gazePageService, params.Slug, t.kbIDs, t.routes)
	if err != nil {
		return &types.ToolResult{Success: false, Error: "Failed to fetch page to delete: " + err.Error()}, nil
	}
	inLinks := make([]string, len(existingPage.InLinks))
	copy(inLinks, existingPage.InLinks)

	parts := strings.Split(params.Slug, "/")
	readableName := strings.ReplaceAll(parts[len(parts)-1], "-", " ")
	pipeLinkRE := regexp.MustCompile(`\[\[` + regexp.QuoteMeta(params.Slug) + `\|([^\]]+)\]\]`)
	changes, updatedSlugs, rewriteErr := applyIncomingGazeContentRewrite(
		ctx, t.gazePageService, kbID, inLinks,
		func(content string) (string, bool) {
			updated := strings.ReplaceAll(content, "[["+params.Slug+"]]", readableName)
			updated = pipeLinkRE.ReplaceAllString(updated, "$1")
			return updated, updated != content
		},
	)
	if rewriteErr != nil {
		rollbackErr := rollbackGazeContentChanges(ctx, t.gazePageService, changes)
		return &types.ToolResult{
			Success: false,
			Error: "Delete aborted while cleaning incoming links: " +
				joinGazeMutationErrors(rewriteErr, rollbackErr),
		}, nil
	}

	err = t.gazePageService.DeletePage(ctx, kbID, params.Slug)
	if err != nil {
		rollbackErr := rollbackGazeContentChanges(ctx, t.gazePageService, changes)
		return &types.ToolResult{
			Success: false,
			Error: "Delete aborted because the page could not be removed: " +
				joinGazeMutationErrors(err, rollbackErr),
		}, nil
	}
	t.routes.forget(params.Slug, kbID)
	updatedCount := len(updatedSlugs)

	outputMsg := fmt.Sprintf("Successfully deleted page [[%s]] and cleaned up %d incoming links.", params.Slug, updatedCount)
	if updatedCount > 0 {
		outputMsg += fmt.Sprintf("\n- Affected pages: %s", strings.Join(updatedSlugs, ", "))
	}

	return &types.ToolResult{
		Success: true,
		Output:  outputMsg,
		Data: map[string]interface{}{
			"display_type":   "gaze_delete_page",
			"slug":           params.Slug,
			"title":          existingPage.Title,
			"updated_count":  updatedCount,
			"affected_pages": updatedSlugs,
		},
	}, nil
}
