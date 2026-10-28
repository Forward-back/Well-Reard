package chat

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestParseCompletionResponseCapturesReasoningContent(t *testing.T) {
	response := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Content:          `{"summary":"ok"}`,
				ReasoningContent: "thinking trace",
			},
		}},
	}
	parsed, err := (&RemoteAPIChat{}).parseCompletionResponse(response)
	if err != nil {
		t.Fatalf("parseCompletionResponse: %v", err)
	}
	if parsed.Content != `{"summary":"ok"}` {
		t.Fatalf("content = %q", parsed.Content)
	}
	if parsed.ReasoningContent != "thinking trace" {
		t.Fatalf("reasoning = %q, want thinking trace", parsed.ReasoningContent)
	}
}
