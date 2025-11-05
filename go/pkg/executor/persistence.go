package executor

import (
	"context"
	"time"
)

// ConversationRecorder captures prompt/response pairs for debugging/cost tracking.
type ConversationRecorder interface {
	RecordConversation(ctx context.Context, rec ConversationRecord) error
}

// ConversationRecord describes a single executor → LLM interaction.
type ConversationRecord struct {
	ModelID          string
	Prompt           string
	PromptTokens     int
	Response         string
	CompletionTokens int
	TotalTokens      int
	ModelName        string
	Timestamp        time.Time
	Topic            string
}

type noopConversationRecorder struct{}

func (noopConversationRecorder) RecordConversation(ctx context.Context, rec ConversationRecord) error {
	return nil
}

// ExecutorOption customises BasicExecutor construction.
type ExecutorOption func(*BasicExecutor)

// WithConversationRecorder injects a recorder used to persist prompt/response pairs.
func WithConversationRecorder(recorder ConversationRecorder) ExecutorOption {
	return func(exec *BasicExecutor) {
		if recorder == nil {
			exec.conversations = noopConversationRecorder{}
			return
		}
		exec.conversations = recorder
	}
}
