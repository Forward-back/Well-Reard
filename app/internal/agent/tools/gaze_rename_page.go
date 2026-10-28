package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type gazeRenamePageTool struct {
	BaseTool
	gazePageService interfaces.GazePageService
	kbIDs           []string
	routes          *GazeRouteResolver
}

// NewGazeRenamePageTool creates a new gaze_rename_page tool
func NewGazeRenamePageTool(
	gazePageService interfaces.GazePageService,
	kbIDs []string,
	routes ...*GazeRouteResolver,
) types.Tool {
	return &gazeRenamePageTool{
		BaseTool: NewBaseTool(
			ToolGazeRenamePage,
			"Rename a Gaze page's slug. Automatically cascades the new slug to all pages that linked to the old one.",
			json.RawMessage(`{
				"type": "object",
				"properties": {
					"slug": {
						"type": "string",
						"description": "The current slug of the Gaze page"
					},
					"new_slug": {
						"type": "string",
						"description": "The new slug for the page"
					}
				},
				"required": ["slug", "new_slug"]
			}`),
		),
		gazePageService: gazePageService,
		kbIDs:           kbIDs,
		routes:          firstGazeRoute(routes),
	}
}

func (t *gazeRenamePageTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	// Attribute every page write performed by this tool to the agent so
	// revision history distinguishes agent edits from pipeline/user ones.
	ctx = types.WithGazeEditSource(ctx, types.GazeEditSourceAgent)
	var params struct {
		Slug    string `json:"slug"`
		NewSlug string `json:"new_slug"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return &types.ToolResult{Success: false, Error: "Failed to parse arguments: " + err.Error()}, nil
	}

	if len(t.kbIDs) == 0 {
		return &types.ToolResult{Success: false, Error: "No knowledge bases available for editing"}, nil
	}
	if params.NewSlug == "" {
		return &types.ToolResult{Success: false, Error: "new_slug is required"}, nil
	}
	normalizedSlug, slugErr := normalizeAndValidateGazeSlug(params.Slug)
	if slugErr != nil {
		return &types.ToolResult{Success: false, Error: slugErr.Error()}, nil
	}
	params.Slug = normalizedSlug
	normalizedNewSlug, slugErr := normalizeAndValidateGazeSlug(params.NewSlug)
	if slugErr != nil {
		return &types.ToolResult{Success: false, Error: slugErr.Error()}, nil
	}
	params.NewSlug = normalizedNewSlug
	if params.NewSlug == params.Slug {
		return &types.ToolResult{Success: false, Error: "new_slug must be different from old slug"}, nil
	}

	// Get existing page
	existingPage, kbID, err := resolveUniqueGazePage(ctx, t.gazePageService, params.Slug, t.kbIDs, t.routes)
	if err != nil {
		return &types.ToolResult{Success: false, Error: "Failed to resolve page to rename: " + err.Error()}, nil
	}

	inLinks := make([]string, len(existingPage.InLinks))
	copy(inLinks, existingPage.InLinks)

	// Create new page with new slug but same content
	newPage := &types.GazePage{
		TenantID:        existingPage.TenantID,
		KnowledgeBaseID: kbID,
		Slug:            params.NewSlug,
		Title:           existingPage.Title,
		Summary:         existingPage.Summary,
		Content:         existingPage.Content,
		PageType:        existingPage.PageType,
		Status:          existingPage.Status,
		Aliases:         append(types.StringArray(nil), existingPage.Aliases...),
		ParentSlug:      existingPage.ParentSlug,
		FolderID:        existingPage.FolderID,
		SortOrder:       existingPage.SortOrder,
		SourceRefs:      append(types.StringArray(nil), existingPage.SourceRefs...),
		ChunkRefs:       append(types.StringArray(nil), existingPage.ChunkRefs...),
		InLinks:         append(types.StringArray(nil), existingPage.InLinks...),
		PageMetadata:    append(types.JSON(nil), existingPage.PageMetadata...),
	}
	_, err = t.gazePageService.CreatePage(ctx, newPage)
	if err != nil {
		return &types.ToolResult{Success: false, Error: "Failed to create renamed page: " + err.Error()}, nil
	}

	changes, updatedSlugs, rewriteErr := applyIncomingGazeContentRewrite(
		ctx, t.gazePageService, kbID, inLinks,
		func(content string) (string, bool) {
			updated := strings.ReplaceAll(
				content, "[["+params.Slug+"]]", "[["+params.NewSlug+"]]",
			)
			updated = strings.ReplaceAll(
				updated, "[["+params.Slug+"|", "[["+params.NewSlug+"|",
			)
			return updated, updated != content
		},
	)
	if rewriteErr != nil {
		rollbackErr := rollbackGazeContentChanges(ctx, t.gazePageService, changes)
		cleanupErr := t.gazePageService.DeletePage(ctx, kbID, params.NewSlug)
		return &types.ToolResult{
			Success: false,
			Error: "Rename aborted while updating incoming links: " +
				joinGazeMutationErrors(rewriteErr, rollbackErr, cleanupErr),
		}, nil
	}
	updatedCount := len(updatedSlugs)

	// Delete old page
	err = t.gazePageService.DeletePage(ctx, kbID, params.Slug)
	if err != nil {
		rollbackErr := rollbackGazeContentChanges(ctx, t.gazePageService, changes)
		cleanupErr := t.gazePageService.DeletePage(ctx, kbID, params.NewSlug)
		return &types.ToolResult{
			Success: false,
			Error: "Rename aborted because the old page could not be deleted: " +
				joinGazeMutationErrors(err, rollbackErr, cleanupErr),
		}, nil
	}
	t.routes.forget(params.Slug, kbID)
	t.routes.remember(params.NewSlug, kbID)

	// Inject cross-links so other pages know about this new slug
	t.gazePageService.InjectCrossLinks(ctx, kbID, []string{params.NewSlug})

	// Rebuild the index page to reflect the new/updated summary
	_ = t.gazePageService.RebuildIndexPage(ctx, kbID)

	outputMsg := fmt.Sprintf("Successfully renamed page [[%s]] → [[%s]] and updated %d incoming links.", params.Slug, params.NewSlug, updatedCount)
	if updatedCount > 0 {
		outputMsg += fmt.Sprintf("\n- Affected pages: %s", strings.Join(updatedSlugs, ", "))
	}

	return &types.ToolResult{
		Success: true,
		Output:  outputMsg,
		Data: map[string]interface{}{
			"display_type":   "gaze_rename_page",
			"old_slug":       params.Slug,
			"new_slug":       params.NewSlug,
			"title":          existingPage.Title,
			"updated_count":  updatedCount,
			"affected_pages": updatedSlugs,
		},
	}, nil
}
