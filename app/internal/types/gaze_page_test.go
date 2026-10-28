package types

import (
	"encoding/json"
	"testing"
)

func TestGazePageTypes(t *testing.T) {
	// Verify all page type constants are non-empty and unique
	pageTypes := []string{
		GazePageTypeSummary,
		GazePageTypeEntity,
		GazePageTypeConcept,
		GazePageTypeIndex,
		GazePageTypeSynthesis,
		GazePageTypeComparison,
	}
	seen := make(map[string]bool)
	for _, pt := range pageTypes {
		if pt == "" {
			t.Errorf("GazePageType constant is empty")
		}
		if seen[pt] {
			t.Errorf("Duplicate GazePageType: %s", pt)
		}
		seen[pt] = true
	}
	if IsValidGazePageType("log") {
		t.Error("removed Gaze log page type must not remain valid")
	}
}

func TestGazeConfigValueScan(t *testing.T) {
	config := GazeConfig{
		SynthesisModelID:    "model-123",
		ContentInstructions: "keep it brief",
	}

	val, err := config.Value()
	if err != nil {
		t.Fatalf("GazeConfig.Value() error: %v", err)
	}

	var restored GazeConfig
	b, ok := val.([]byte)
	if !ok {
		t.Fatal("GazeConfig.Value() did not return []byte")
	}
	if err := restored.Scan(b); err != nil {
		t.Fatalf("GazeConfig.Scan() error: %v", err)
	}

	if restored.SynthesisModelID != "model-123" {
		t.Error("SynthesisModelID mismatch")
	}
	if restored.ContentInstructions != "keep it brief" {
		t.Error("ContentInstructions mismatch")
	}
}

func TestGazeConfigScanNil(t *testing.T) {
	var config GazeConfig
	if err := config.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) should not error: %v", err)
	}
}

func TestStringArrayValueScan(t *testing.T) {
	arr := StringArray{"a", "b", "c"}

	val, err := arr.Value()
	if err != nil {
		t.Fatalf("StringArray.Value() error: %v", err)
	}

	var restored StringArray
	b, ok := val.([]byte)
	if !ok {
		t.Fatal("StringArray.Value() did not return []byte")
	}
	if err := restored.Scan(b); err != nil {
		t.Fatalf("StringArray.Scan() error: %v", err)
	}

	if len(restored) != 3 || restored[0] != "a" || restored[1] != "b" || restored[2] != "c" {
		t.Errorf("StringArray round-trip failed: got %v", restored)
	}
}

func TestStringArrayEmpty(t *testing.T) {
	// nil StringArray marshals to JSON null via json.Marshal
	var arr StringArray
	val, err := arr.Value()
	if err != nil {
		t.Fatalf("nil StringArray.Value() error: %v", err)
	}
	// json.Marshal(nil) returns "null", which is valid
	if val == nil {
		// Some implementations return nil for nil slices
		return
	}
	b, ok := val.([]byte)
	if !ok {
		t.Fatalf("unexpected type: %T", val)
	}
	s := string(b)
	if s != "null" && s != "[]" {
		t.Errorf("nil StringArray.Value() should return 'null' or '[]', got %q", s)
	}
}

func TestKnowledgeBaseEnsureDefaultsGaze(t *testing.T) {
	kb := &KnowledgeBase{
		Type:       KnowledgeBaseTypeDocument,
		GazeConfig: &GazeConfig{SynthesisModelID: "m-1"},
	}
	kb.EnsureDefaults()

	if kb.GazeConfig == nil {
		t.Fatal("EnsureDefaults should preserve GazeConfig for gaze-enabled KB")
	}
	if kb.GazeConfig.SynthesisModelID != "m-1" {
		t.Error("SynthesisModelID should be preserved")
	}
	if kb.FAQConfig != nil {
		t.Error("Document KB should not have FAQConfig")
	}
}

func TestKnowledgeBaseEnsureDefaultsDocumentWithoutGaze(t *testing.T) {
	kb := &KnowledgeBase{
		Type: KnowledgeBaseTypeDocument,
	}
	kb.EnsureDefaults()

	// GazeConfig should remain nil when not set
	if kb.GazeConfig != nil {
		t.Error("Document KB without gaze should not have GazeConfig after EnsureDefaults")
	}
}

func TestGazePageJSON(t *testing.T) {
	page := GazePage{
		ID:         "test-id",
		Slug:       "entity/test",
		Title:      "Test Entity",
		PageType:   GazePageTypeEntity,
		Content:    "# Test\n\nSome content with [[concept/related]]",
		Summary:    "A test entity page",
		SourceRefs: StringArray{"source-1", "source-2"},
		OutLinks:   StringArray{"concept/related"},
		InLinks:    StringArray{"summary/doc1"},
		Version:    3,
	}

	// Serialize
	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Deserialize
	var restored GazePage
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if restored.Slug != "entity/test" {
		t.Errorf("Slug mismatch: got %s", restored.Slug)
	}
	if restored.Version != 3 {
		t.Errorf("Version mismatch: got %d", restored.Version)
	}
	if len(restored.OutLinks) != 1 || restored.OutLinks[0] != "concept/related" {
		t.Errorf("OutLinks mismatch: got %v", restored.OutLinks)
	}
	if len(restored.SourceRefs) != 2 {
		t.Errorf("SourceRefs mismatch: got %v", restored.SourceRefs)
	}
}

func TestGazeSourceKnowledgeID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"doc-1|排班手册", "doc-1"},
		{"doc-1", "doc-1"},
		{"  doc-1 | title ", "doc-1"},
		{"", ""},
	}
	for _, c := range cases {
		if got := GazeSourceKnowledgeID(c.in); got != c.want {
			t.Errorf("GazeSourceKnowledgeID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	page := &GazePage{SourceRefs: StringArray{"doc-1|手册", "doc-2"}}
	if !page.BuiltFrom(map[string]struct{}{"doc-1": {}}) {
		t.Errorf("page should match doc-1")
	}
	if page.BuiltFrom(map[string]struct{}{"doc-9": {}}) {
		t.Errorf("page should not match an unrelated document")
	}
}

func TestGazeGraphDataJSON(t *testing.T) {
	graph := GazeGraphData{
		Nodes: []GazeGraphNode{
			{Slug: "entity/a", Title: "A", PageType: "entity", LinkCount: 2},
			{Slug: "concept/b", Title: "B", PageType: "concept", LinkCount: 1},
		},
		Edges: []GazeGraphEdge{
			{Source: "entity/a", Target: "concept/b"},
		},
	}

	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var restored GazeGraphData
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(restored.Nodes) != 2 {
		t.Errorf("Nodes count mismatch: got %d", len(restored.Nodes))
	}
	if len(restored.Edges) != 1 {
		t.Errorf("Edges count mismatch: got %d", len(restored.Edges))
	}
	if restored.Edges[0].Source != "entity/a" || restored.Edges[0].Target != "concept/b" {
		t.Errorf("Edge mismatch: got %v", restored.Edges[0])
	}
}

func TestGazeFolderPathSegmentsKeepTypeLikeFolderNames(t *testing.T) {
	got := GazeFolderPathSegments("概念/ 概念 /Concepts//实体")
	want := []string{"概念", "概念", "Concepts", "实体"}
	if len(got) != len(want) {
		t.Fatalf("segments = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("segments[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if GazeFolderPathSegments("  ") != nil {
		t.Fatalf("blank path must yield nil")
	}
	// The model-label cleaner still drops the same names, which is why folder
	// paths must not go through it.
	if cleaned := CleanGazeCategoryPath([]string{"概念"}); len(cleaned) != 0 {
		t.Fatalf("CleanGazeCategoryPath(概念) = %v, want empty", cleaned)
	}
}
