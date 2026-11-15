// Package core implements the Cherry-Nexus core: a multi-provider AI gateway
// with the CCR (Claude Code Router) smart routing logic and seamless fallback
// between web-based providers (DeepSeek Web, Gemini Web, GLM Web, Qwen Web, ...).
//
// This package is provider-agnostic: each provider is adapted to the Provider
// interface and registered in a Registry. The RuleRouter then picks the best
// provider/model per request and the FallbackRunner retries across providers
// on 429/network errors, exactly as described in the Cherry-Nexus concept.
package core

import (
	"context"
)

// Capability describes what a single model can do.
type Capability struct {
	Reasoning        bool  // deep thinking / chain-of-thought
	WebSearch        bool  // real-time web search
	ImageGeneration  bool  // image generation
	MaxContextTokens int   // context window size in tokens
}

// ModelInfo describes a single model served by a provider.
type ModelInfo struct {
	ID        string     // provider-local model id, e.g. "deepseek-reasoner"
	RealModel string     // upstream / friendly model name
	Provider  string     // provider name that serves this model (set by Registry)
	Caps      Capability
}

// ChatMessage is a single turn in a conversation.
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ChatRequest carries everything a provider needs to stream one completion.
type ChatRequest struct {
	Prompt           string
	Model            string
	SessionID        string
	ParentMessageID  *int64
	ThinkingEnabled  *bool
	SearchEnabled    *bool
	Messages         []ChatMessage // full conversation history (when the provider supports it)
}

// ChatChunk is an incremental piece of streamed response text or reasoning.
type ChatChunk struct {
	Content          string
	ReasoningContent string
	SessionID        string
	MessageID        string
	FinishReason     string
}

// AuthStatus reports whether a provider is authenticated and ready to serve.
type AuthStatus struct {
	Provider string
	Ready    bool
	Message  string // human-readable diagnosis, e.g. "token konfiguratsiya qilindi"
}

// Provider is a generic AI provider adapter.
type Provider interface {
	// Name returns the provider identifier, e.g. "deepseek", "gemini".
	Name() string
	// Models returns every model this provider can serve.
	Models() []ModelInfo
	// Capability returns the capability flags for a model.
	Capability(model string) Capability
	// AuthStatus reports whether the provider is authenticated/ready.
	AuthStatus() AuthStatus
	// ChatStream streams a chat completion, yielding chunks and errors.
	// Mirrors deepseek.Client.ChatStream semantics.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error)
}
