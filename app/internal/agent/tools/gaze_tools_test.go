package tools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"gaze/internal/application/repository"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type fakeGazePageService struct {
	interfaces.GazePageService

	mu            sync.Mutex
	pages         map[string]*types.GazePage
	searchResults map[string][]*types.GazePage
	getErrors     map[string]error
	searchErrors  map[string]error
	getCalls      []string
	searchCalls   []string
}

func gazePageKey(kbID, slug string) string {
	return kbID + "\x00" + slug
}

func (f *fakeGazePageService) GetPageBySlug(_ context.Context, kbID, slug string) (*types.GazePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls = append(f.getCalls, gazePageKey(kbID, slug))
	if err := f.getErrors[gazePageKey(kbID, slug)]; err != nil {
		return nil, err
	}
	return f.pages[gazePageKey(kbID, slug)], nil
}

func (f *fakeGazePageService) SearchPages(_ context.Context, kbID, _ string, _ int) ([]*types.GazePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searchCalls = append(f.searchCalls, kbID)
	if err := f.searchErrors[kbID]; err != nil {
		return nil, err
	}
	return f.searchResults[kbID], nil
}

func (f *fakeGazePageService) resetCalls() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls = nil
	f.searchCalls = nil
}

func newTestGazePage(kbID, slug string) *types.GazePage {
	return &types.GazePage{
		KnowledgeBaseID: kbID,
		Slug:            slug,
		Title:           slug,
		PageType:        types.GazePageTypeConcept,
		Content:         "content for " + slug,
	}
}

func TestTruncateForSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{
			name:    "short content",
			content: "Hello world",
			maxLen:  50,
			want:    "Hello world",
		},
		{
			name:    "strips heading prefix",
			content: "# My Title\n\nSome content",
			maxLen:  50,
			want:    "My Title",
		},
		{
			name:    "strips h2 prefix",
			content: "## Section\n\nContent here",
			maxLen:  50,
			want:    "Section",
		},
		{
			name:    "truncates long content",
			content: "This is a very long paragraph that should be truncated at some point because it exceeds the maximum length",
			maxLen:  20,
			want:    "This is a very long ..."},
		{
			name:    "takes first paragraph",
			content: "First paragraph.\n\nSecond paragraph.",
			maxLen:  100,
			want:    "First paragraph.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForSummary(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateForSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGazeToolConstants(t *testing.T) {
	// Verify all gaze tool constants are defined and unique
	names := []string{
		ToolGazeReadPage,
		ToolGazeSearch,
	}

	seen := make(map[string]bool)
	for _, name := range names {
		if name == "" {
			t.Error("Gaze tool constant is empty")
		}
		if seen[name] {
			t.Errorf("Duplicate gaze tool constant: %s", name)
		}
		seen[name] = true

		// Verify name length is within OpenAI limit
		if len(name) > maxFunctionNameLength {
			t.Errorf("Gaze tool name too long: %s (%d chars, max %d)", name, len(name), maxFunctionNameLength)
		}
	}
}

func TestGazeToolsInAvailableDefinitions(t *testing.T) {
	defs := AvailableToolDefinitions()
	gazeTools := map[string]bool{
		ToolGazeReadPage: false,
		ToolGazeSearch:   false,
	}

	for _, def := range defs {
		if _, ok := gazeTools[def.Name]; ok {
			gazeTools[def.Name] = true
		}
	}

	for name, found := range gazeTools {
		if !found {
			t.Errorf("Gaze tool %s missing from AvailableToolDefinitions()", name)
		}
	}
}

func TestGazeReadPageSchemaUsesAutomaticKBRouting(t *testing.T) {
	tool := NewGazeReadPageTool(
		&fakeGazePageService{}, nil,
		NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver(),
	)
	if strings.Contains(string(tool.Parameters()), "knowledge_base_id") {
		t.Fatalf("gaze_read_page schema must not ask the model to choose a knowledge base: %s", tool.Parameters())
	}
}

func TestGazeReadPageRoutesEachSlugIndependently(t *testing.T) {
	pageA := newTestGazePage("kb-1", "concept/a")
	pageB := newTestGazePage("kb-2", "concept/b")
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", pageA.Slug): pageA,
		gazePageKey("kb-2", pageB.Slug): pageB,
	}}
	routes := NewGazeRouteResolver()
	routes.remember(pageA.Slug, "kb-1")
	routes.remember(pageB.Slug, "kb-2")
	tool := NewGazeReadPageTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"}), routes,
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/a","concept/b"]}`))
	if err != nil || result == nil || !result.Success {
		t.Fatalf("gaze_read_page failed: result=%+v err=%v", result, err)
	}
	wantCalls := []string{
		gazePageKey("kb-1", "concept/a"),
		gazePageKey("kb-2", "concept/a"),
		gazePageKey("kb-2", "concept/b"),
		gazePageKey("kb-1", "concept/b"),
	}
	if !reflect.DeepEqual(service.getCalls, wantCalls) {
		t.Fatalf("GetPageBySlug calls = %v, want %v", service.getCalls, wantCalls)
	}
}

func TestGazeReadPageFallsBackWhenCachedRouteIsStale(t *testing.T) {
	page := newTestGazePage("kb-2", "concept/target")
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-2", page.Slug): page,
	}}
	routes := NewGazeRouteResolver()
	routes.remember(page.Slug, "kb-1")
	tool := NewGazeReadPageTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"}), routes,
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/target"]}`))
	if err != nil || result == nil || !result.Success {
		t.Fatalf("gaze_read_page should recover from a stale route: result=%+v err=%v", result, err)
	}
	wantCalls := []string{
		gazePageKey("kb-1", page.Slug),
		gazePageKey("kb-2", page.Slug),
	}
	if !reflect.DeepEqual(service.getCalls, wantCalls) {
		t.Fatalf("GetPageBySlug calls = %v, want stale route then fallback %v", service.getCalls, wantCalls)
	}
}

func TestGazeReadPageDoesNotLetLegacyKBArgumentPinAllSlugs(t *testing.T) {
	page := newTestGazePage("kb-2", "concept/target")
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-2", page.Slug): page,
	}}
	tool := NewGazeReadPageTool(
		service, nil,
		NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"}), NewGazeRouteResolver(),
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(
		`{"slugs":["concept/target"],"knowledge_base_id":"b2"}`,
	))
	if err != nil || result == nil || !result.Success {
		t.Fatalf("legacy KB argument must not pin gaze_read_page: result=%+v err=%v", result, err)
	}
	wantCalls := []string{
		gazePageKey("kb-1", page.Slug),
		gazePageKey("kb-2", page.Slug),
	}
	if !reflect.DeepEqual(service.getCalls, wantCalls) {
		t.Fatalf("GetPageBySlug calls = %v, want automatic scope scan %v", service.getCalls, wantCalls)
	}
}

func TestGazeSearchSharesRoutesWithGazeReadPage(t *testing.T) {
	page := newTestGazePage("kb-2", "concept/target")
	service := &fakeGazePageService{
		pages: map[string]*types.GazePage{
			gazePageKey("kb-2", page.Slug): page,
		},
		searchResults: map[string][]*types.GazePage{
			"kb-2": {page},
		},
	}
	scopes := NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"})
	routes := NewGazeRouteResolver()
	searchTool := NewGazeSearchTool(service, nil, scopes, routes)
	readTool := NewGazeReadPageTool(service, nil, scopes, routes)

	searchResult, err := searchTool.Execute(context.Background(), json.RawMessage(`{"queries":["target"]}`))
	if err != nil || searchResult == nil || !searchResult.Success {
		t.Fatalf("gaze_search failed: result=%+v err=%v", searchResult, err)
	}
	service.resetCalls()

	readResult, err := readTool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/target"]}`))
	if err != nil || readResult == nil || !readResult.Success {
		t.Fatalf("gaze_read_page failed: result=%+v err=%v", readResult, err)
	}
	wantCalls := []string{
		gazePageKey("kb-2", page.Slug),
		gazePageKey("kb-1", page.Slug),
	}
	if !reflect.DeepEqual(service.getCalls, wantCalls) {
		t.Fatalf("GetPageBySlug calls = %v, want cached owner first and full ambiguity scan %v", service.getCalls, wantCalls)
	}
}

func TestGazeReadPageCachedRouteCannotHideDuplicateSlug(t *testing.T) {
	slug := "concept/shared"
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", slug): newTestGazePage("kb-1", slug),
		gazePageKey("kb-2", slug): newTestGazePage("kb-2", slug),
	}}
	routes := NewGazeRouteResolver()
	routes.remember(slug, "kb-1")
	tool := NewGazeReadPageTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"}), routes,
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/shared"]}`))
	if err != nil || result == nil || !result.Success {
		t.Fatalf("gaze_read_page failed: result=%+v err=%v", result, err)
	}
	if !strings.Contains(result.Output, "<knowledge_base_id>kb-1</knowledge_base_id>") ||
		!strings.Contains(result.Output, "<knowledge_base_id>kb-2</knowledge_base_id>") {
		t.Fatalf("cached provenance hid an ambiguous page: %s", result.Output)
	}
}

func TestGazeReadPageReturnsSameSlugFromAllScopesOnCacheMiss(t *testing.T) {
	slug := "concept/shared"
	pageA := newTestGazePage("kb-1", slug)
	pageB := newTestGazePage("kb-2", slug)
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", slug): pageA,
		gazePageKey("kb-2", slug): pageB,
	}}
	tool := NewGazeReadPageTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1", "kb-2"}), NewGazeRouteResolver(),
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/shared"]}`))
	if err != nil || result == nil || !result.Success {
		t.Fatalf("gaze_read_page failed: result=%+v err=%v", result, err)
	}
	if !strings.Contains(result.Output, "<knowledge_base_id>kb-1</knowledge_base_id>") ||
		!strings.Contains(result.Output, "<knowledge_base_id>kb-2</knowledge_base_id>") {
		t.Fatalf("expected pages from both scopes, got: %s", result.Output)
	}
}

func TestGazeSearchRejectsKnowledgeBaseOutsideScope(t *testing.T) {
	service := &fakeGazePageService{}
	tool := NewGazeSearchTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver(),
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"queries":["target"],"knowledge_base_id":"kb-outside"}`))
	if err != nil {
		t.Fatalf("gaze_search returned unexpected error: %v", err)
	}
	if result == nil || result.Success || !strings.Contains(result.Error, "not within") {
		t.Fatalf("expected scope rejection, got: %+v", result)
	}
	if len(service.searchCalls) != 0 {
		t.Fatalf("out-of-scope search must not reach the service, calls=%v", service.searchCalls)
	}
}

func TestGazeReadPageSurfacesBackendFailureInsteadOfNotFound(t *testing.T) {
	service := &fakeGazePageService{getErrors: map[string]error{
		gazePageKey("kb-1", "concept/a"): errors.New("database unavailable"),
	}}
	tool := NewGazeReadPageTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver(),
	)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"slugs":["concept/a"]}`))
	if err != nil || result == nil || result.Success || !strings.Contains(result.Error, "database unavailable") {
		t.Fatalf("backend failure must be preserved: result=%+v err=%v", result, err)
	}
	if strings.Contains(result.Error, "not found") {
		t.Fatalf("backend failure must not be rewritten as not found: %s", result.Error)
	}
}

func TestGazeSearchFailsWhenEveryAllowedBackendLookupFails(t *testing.T) {
	service := &fakeGazePageService{searchErrors: map[string]error{
		"kb-1": errors.New("search backend unavailable"),
	}}
	tool := NewGazeSearchTool(
		service, nil, NewGazeScopesFromKBIDs([]string{"kb-1"}), NewGazeRouteResolver(),
	)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"queries":["target"]}`))
	if err != nil || result == nil || result.Success || !strings.Contains(result.Error, "search backend unavailable") {
		t.Fatalf("search backend failure must be preserved: result=%+v err=%v", result, err)
	}
}

func TestResolveUniqueGazePageRejectsAmbiguousMutationTarget(t *testing.T) {
	slug := "concept/shared"
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", slug): newTestGazePage("kb-1", slug),
		gazePageKey("kb-2", slug): newTestGazePage("kb-2", slug),
	}}
	_, _, err := resolveUniqueGazePage(
		context.Background(), service, slug, []string{"kb-1", "kb-2"}, NewGazeRouteResolver(),
	)
	if err == nil || !strings.Contains(err.Error(), "multiple knowledge bases") {
		t.Fatalf("expected ambiguous target rejection, got %v", err)
	}
}

func TestResolveUniqueGazePageUsesActualOwnerWithoutModelKBArgument(t *testing.T) {
	slug := "concept/target"
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-2", slug): newTestGazePage("kb-2", slug),
	}}
	page, kbID, err := resolveUniqueGazePage(
		context.Background(), service, slug, []string{"kb-1", "kb-2"}, NewGazeRouteResolver(),
	)
	if err != nil || page == nil || kbID != "kb-2" {
		t.Fatalf("target resolution failed: page=%+v kb=%s err=%v", page, kbID, err)
	}
}

func TestResolveUniqueGazePageSkipsOnlyNotFoundErrors(t *testing.T) {
	slug := "concept/target"
	service := &fakeGazePageService{
		pages: map[string]*types.GazePage{
			gazePageKey("kb-2", slug): newTestGazePage("kb-2", slug),
		},
		getErrors: map[string]error{
			gazePageKey("kb-1", slug): repository.ErrGazePageNotFound,
		},
	}
	page, kbID, err := resolveUniqueGazePage(
		context.Background(), service, slug, []string{"kb-1", "kb-2"}, NewGazeRouteResolver(),
	)
	if err != nil || page == nil || kbID != "kb-2" {
		t.Fatalf("not-found should continue to other allowed scopes: page=%+v kb=%s err=%v", page, kbID, err)
	}

	service.getErrors[gazePageKey("kb-1", slug)] = errors.New("database unavailable")
	_, _, err = resolveUniqueGazePage(
		context.Background(), service, slug, []string{"kb-1", "kb-2"}, NewGazeRouteResolver(),
	)
	if err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("non-not-found lookup errors must fail closed, got %v", err)
	}
}

func TestResolveUniqueGazePageRejectsMismatchedServiceOwner(t *testing.T) {
	slug := "concept/target"
	service := &fakeGazePageService{pages: map[string]*types.GazePage{
		gazePageKey("kb-1", slug): newTestGazePage("kb-outside", slug),
	}}
	_, _, err := resolveUniqueGazePage(
		context.Background(), service, slug, []string{"kb-1"}, NewGazeRouteResolver(),
	)
	if err == nil || !strings.Contains(err.Error(), "returned knowledge base kb-outside") {
		t.Fatalf("mismatched service owner must be rejected, got %v", err)
	}
}

func TestResolveGazeCreateKBRequiresUnambiguousServerContext(t *testing.T) {
	routes := NewGazeRouteResolver()
	if _, err := resolveGazeCreateKB("concept/new", []string{"kb-1", "kb-2"}, routes); err == nil {
		t.Fatal("new page creation across multiple unhinted KBs must be rejected")
	}
	routes.remember("concept/new", "kb-2")
	kbID, err := resolveGazeCreateKB("concept/new", []string{"kb-1", "kb-2"}, routes)
	if err != nil || kbID != "kb-2" {
		t.Fatalf("cached provenance should choose kb-2: kb=%s err=%v", kbID, err)
	}
}
