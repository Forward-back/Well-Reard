package service

import (
	"encoding/json"
	"testing"

	"gaze/internal/types"
)

func TestNormalizeReadingDepth(t *testing.T) {
	cases := map[string]string{
		"":         types.ReadingDepthStandard,
		"standard": types.ReadingDepthStandard,
		"quick":    types.ReadingDepthQuick,
		"deep":     types.ReadingDepthDeep,
		"DEEP":     types.ReadingDepthDeep,
		"unknown":  types.ReadingDepthStandard,
	}
	for input, want := range cases {
		if got := normalizeReadingDepth(input); got != want {
			t.Fatalf("normalizeReadingDepth(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestReadingDepthProfiles(t *testing.T) {
	if readingStepLimit(types.ReadingDepthQuick) >= readingStepLimit(types.ReadingDepthStandard) {
		t.Fatal("quick depth should allow fewer steps than standard")
	}
	if readingStepLimit(types.ReadingDepthDeep) <= readingStepLimit(types.ReadingDepthStandard) {
		t.Fatal("deep depth should allow more steps than standard")
	}
	if readingDepthGuidance(types.ReadingDepthQuick) == readingDepthGuidance(types.ReadingDepthDeep) {
		t.Fatal("depth guidance should differ")
	}
}

func TestReadingRunDepthFallsBackToPlan(t *testing.T) {
	planJSON, _ := json.Marshal(types.ReadingPlan{Depth: types.ReadingDepthDeep})
	run := &types.ReadingRun{Plan: types.JSON(planJSON)}
	if got := readingRunDepth(run); got != types.ReadingDepthDeep {
		t.Fatalf("readingRunDepth() = %q, want deep", got)
	}
	if got := readingRunDepth(nil); got != types.ReadingDepthStandard {
		t.Fatalf("readingRunDepth(nil) = %q, want standard", got)
	}
}

func TestParseReadingLLMAllowsSelectedDocSearch(t *testing.T) {
	parsed, err := parseReadingLLM(`{"action":"search_selected_docs","search_query":"D+1 垫资"}`)
	if err != nil {
		t.Fatalf("parseReadingLLM() error = %v", err)
	}
	if parsed.Action != "search_selected_docs" || parsed.SearchQuery != "D+1 垫资" {
		t.Fatalf("unexpected parsed response: %+v", parsed)
	}
	if _, err := parseReadingLLM(`{"action":"search_selected_docs"}`); err == nil {
		t.Fatal("expected empty search query to fail")
	}
}

func TestReadingNotesPatchAcceptsObjectTerms(t *testing.T) {
	var patch types.ReadingNotesPatch
	if err := json.Unmarshal([]byte(`{"terms":[{"term":"D+1","definition":"次日到账"}],"confirmed":["支持对私"]}`), &patch); err != nil {
		t.Fatalf("unmarshal notes patch: %v", err)
	}
	if len(patch.Terms) != 1 || patch.Terms[0] != "D+1：次日到账" {
		t.Fatalf("terms = %#v", patch.Terms)
	}
	if len(patch.Confirmed) != 1 || patch.Confirmed[0] != "支持对私" {
		t.Fatalf("confirmed = %#v", patch.Confirmed)
	}
}
func TestReadingChunkMatches(t *testing.T) {
	if !readingChunkMatches("D+1 结算产品支持节假日到账", "节假日") {
		t.Fatal("expected exact substring match")
	}
	if !readingChunkMatches("D+1 结算产品支持节假日到账", "结算 到账") {
		t.Fatal("expected token match")
	}
	if readingChunkMatches("D+1 结算产品", "垫资") {
		t.Fatal("unexpected match")
	}
	if got := readingTruncateRunes("一二三四五", 3); got != "一二三…" {
		t.Fatalf("readingTruncateRunes() = %q", got)
	}
}
