package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"gaze/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newBulkyGazePage builds a page whose body alone is larger than a fair share
// of the default output budget.
func newBulkyGazePage(kbID, slug string, bodyRunes int) *types.GazePage {
	page := newTestGazePage(kbID, slug)
	page.Summary = "summary of " + slug
	page.Content = slug + " body " + strings.Repeat("x", bodyRunes)
	return page
}

func readPageOutput(t *testing.T, ctx context.Context, tool types.Tool, slugs []string) *types.ToolResult {
	t.Helper()
	args, err := json.Marshal(map[string]any{"slugs": slugs})
	require.NoError(t, err)
	result, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success, "gaze_read_page failed: %s", result.Error)
	return result
}

// assertWellFormedPages guards the property the old tail truncation broke: the
// UI pairs <gaze_page> with </gaze_page>, so an unbalanced block is dropped
// entirely rather than shown partially.
func assertWellFormedPages(t *testing.T, output string, want int) {
	t.Helper()
	assert.Equal(t, want, strings.Count(output, "<gaze_page>"), "opening tags")
	assert.Equal(t, want, strings.Count(output, "</gaze_page>"), "closing tags")
	assert.Equal(t, want, strings.Count(output, "</content>"), "content sections")
}

// A batch read used to lose its middle pages: the pages were concatenated and
// then cut to a head/tail window by the registry, so slugs 3..5 of a 5-slug
// call disappeared without any signal to the model.
func TestGazeReadPageKeepsEveryRequestedSlugWithinBudget(t *testing.T) {
	slugs := []string{
		"concept/open-source-accessibility",
		"entity/open-source-accessibility-community-day",
		"concept/assistive-technology",
		"entity/github-multilingual-repositories-dataset",
		"concept/multilingual-ai",
	}
	pages := map[string]*types.GazePage{}
	for _, slug := range slugs {
		pages[gazePageKey("kb-1", slug)] = newBulkyGazePage("kb-1", slug, 8000)
	}
	service := &fakeGazePageService{pages: pages}
	tool := NewGazeReadPageTool(service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver())

	result := readPageOutput(t, context.Background(), tool, slugs)

	assertWellFormedPages(t, result.Output, len(slugs))
	for _, slug := range slugs {
		assert.Contains(t, result.Output, fmt.Sprintf("[[%s|%s]]", slug, slug),
			"every requested slug must still be rendered")
	}
	assert.LessOrEqual(t, utf8.RuneCountInString(result.Output), DefaultMaxToolOutput)
	assert.NotContains(t, result.Output, "<omitted_pages")
	assert.Len(t, result.Data["truncated_slugs"], len(slugs))
}

// Trimming has a floor. Below it, dropping a page by name beats rendering a
// stub, but the model must be told which slugs it still has not seen.
func TestGazeReadPageNamesOmittedPagesWhenBudgetIsTooSmall(t *testing.T) {
	slugs := []string{"concept/a", "concept/b", "concept/c", "concept/d", "concept/e"}
	pages := map[string]*types.GazePage{}
	for _, slug := range slugs {
		pages[gazePageKey("kb-1", slug)] = newBulkyGazePage("kb-1", slug, 4000)
	}
	service := &fakeGazePageService{pages: pages}
	tool := NewGazeReadPageTool(service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver())

	ctx := WithOutputBudget(context.Background(), 2000)
	result := readPageOutput(t, ctx, tool, slugs)

	omitted, ok := result.Data["omitted_slugs"].([]string)
	require.True(t, ok)
	require.NotEmpty(t, omitted, "a 2000-rune budget cannot hold five pages")

	assertWellFormedPages(t, result.Output, len(slugs)-len(omitted))
	assert.Contains(t, result.Output, "<omitted_pages")
	for _, slug := range omitted {
		assert.Contains(t, result.Output, slug)
	}
	assert.Contains(t, result.Output, "Call gaze_read_page again with fewer slugs")
}

// A page short enough to fit must not be trimmed just because a sibling in the
// same batch is huge.
func TestGazeReadPageKeepsSmallPagesIntactBesideLargeOnes(t *testing.T) {
	small := newTestGazePage("kb-1", "concept/small")
	small.Content = "a compact body that easily fits the budget"
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", "concept/small"): small,
		gazePageKey("kb-1", "concept/huge"):  newBulkyGazePage("kb-1", "concept/huge", 40000),
	}}
	tool := NewGazeReadPageTool(service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver())

	result := readPageOutput(t, context.Background(), tool, []string{"concept/small", "concept/huge"})

	assertWellFormedPages(t, result.Output, 2)
	assert.Contains(t, result.Output, small.Content, "the small page must survive untrimmed")
	assert.Equal(t, []string{"concept/huge"}, result.Data["truncated_slugs"])
}

// Inlining a full summary for every neighbour costs one query each and used to
// consume more budget than the bodies the caller actually asked for.
func TestGazeReadPageCapsInlinedLinkSummaries(t *testing.T) {
	page := newTestGazePage("kb-1", "concept/hub")
	pages := map[string]*types.GazePage{gazePageKey("kb-1", "concept/hub"): page}
	for i := 0; i < 40; i++ {
		slug := fmt.Sprintf("entity/neighbour-%02d", i)
		page.OutLinks = append(page.OutLinks, slug)
		neighbour := newTestGazePage("kb-1", slug)
		neighbour.Summary = "NEIGHBOUR_SUMMARY " + strings.Repeat("y", 500)
		pages[gazePageKey("kb-1", slug)] = neighbour
	}
	service := &fakeGazePageService{pages: pages}
	tool := NewGazeReadPageTool(service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver())

	result := readPageOutput(t, context.Background(), tool, []string{"concept/hub"})

	assert.Equal(t, gazeMaxLinkSummaries, strings.Count(result.Output, "NEIGHBOUR_SUMMARY"),
		"only the first %d neighbour summaries may be inlined", gazeMaxLinkSummaries)
	for i := 0; i < 40; i++ {
		assert.Contains(t, result.Output, fmt.Sprintf("[[entity/neighbour-%02d]]", i),
			"every neighbour slug stays visible even without its summary")
	}
	assert.NotContains(t, result.Output, strings.Repeat("y", gazeLinkSummaryMaxRunes+1),
		"each inlined summary must be capped")
}

// The same slug arriving twice used to be resolved and rendered twice, spending
// budget that a distinct page needed.
func TestGazeReadPageDedupesRequestedSlugs(t *testing.T) {
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", "concept/a"): newTestGazePage("kb-1", "concept/a"),
	}}
	tool := NewGazeReadPageTool(service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver())

	args := json.RawMessage(`{"slugs":["concept/a","concept/a"],"slug":"concept/a"}`)
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	require.True(t, result.Success)

	assertWellFormedPages(t, result.Output, 1)
}
