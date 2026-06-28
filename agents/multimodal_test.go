package agents

import (
	"testing"

	"github.com/zhenruyan/postgrebase/models/settings"
)

func TestHistoryToMessagesImages(t *testing.T) {
	history := []SessionMessage{
		{Role: "user", Content: "describe this", Images: []SessionImage{{MimeType: "image/png", Data: "AAAA"}}},
		{Role: "assistant", Content: "it is a chart"},
		{Role: "tool", Content: "data.query: {...}"}, // must be dropped
		{Role: "user", Content: "plain text"},
	}

	msgs := historyToMessages(history)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 replayed messages (tool dropped), got %d", len(msgs))
	}

	// first message is multimodal: text + image blocks
	first := msgs[0]
	if len(first.Contents) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(first.Contents))
	}
	if first.Contents[0].Type != "text" || first.Contents[1].Type != "image" {
		t.Fatalf("expected [text,image] blocks, got [%s,%s]", first.Contents[0].Type, first.Contents[1].Type)
	}
	if first.Contents[1].Image == nil || first.Contents[1].Image.Data != "AAAA" {
		t.Fatal("image block payload not preserved")
	}

	// last message is plain text
	last := msgs[2]
	if last.Content != "plain text" || len(last.Contents) != 0 {
		t.Fatalf("expected plain text user message, got content=%q contents=%d", last.Content, len(last.Contents))
	}
}

func TestModelSupportsVision(t *testing.T) {
	provider := settings.AgentProviderConfig{
		Models: []settings.AgentProviderModel{
			{Name: "gpt-4o", ProviderModelId: "gpt-4o", SupportsVision: true},
			{Name: "deepseek-chat", ProviderModelId: "deepseek-chat", SupportsVision: false},
		},
	}

	if !modelSupportsVision(provider, "gpt-4o") {
		t.Error("gpt-4o should support vision")
	}
	if modelSupportsVision(provider, "deepseek-chat") {
		t.Error("deepseek-chat should not support vision")
	}
	if modelSupportsVision(provider, "unknown") {
		t.Error("unknown model should default to no vision")
	}
}
