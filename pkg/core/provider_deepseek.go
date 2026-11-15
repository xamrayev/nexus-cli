package core

import (
	"context"

	"github.com/xamrayev/go-deepseek-web/deepseek"
)

// deepseekProvider adapts go-deepseek-web to the core.Provider interface.
type deepseekProvider struct {
	client *deepseek.Client
}

var _ Provider = (*deepseekProvider)(nil)

// NewDeepSeekProvider wraps a DeepSeek web client as a core Provider.
func NewDeepSeekProvider(token string) Provider {
	return &deepseekProvider{
		client: deepseek.NewClient(deepseek.ClientOptions{Token: token}),
	}
}

func (p *deepseekProvider) Name() string { return "deepseek" }

func (p *deepseekProvider) Models() []ModelInfo {
	return []ModelInfo{
		{
			ID:        deepseek.ModelReasoner,
			RealModel: "DeepSeek-R1 Reasoner (Web Default + Thinking)",
			Caps: Capability{Reasoning: true, MaxContextTokens: 128000},
		},
		{
			ID:        deepseek.ModelChat,
			RealModel: "DeepSeek-V3 Chat (Web Default)",
			Caps: Capability{MaxContextTokens: 128000},
		},
		{
			ID:        deepseek.ModelReasonerSearch,
			RealModel: "DeepSeek-R1 Reasoner + Web Search",
			Caps: Capability{Reasoning: true, WebSearch: true, MaxContextTokens: 128000},
		},
		{
			ID:        deepseek.ModelChatSearch,
			RealModel: "DeepSeek-V3 Chat + Web Search",
			Caps: Capability{WebSearch: true, MaxContextTokens: 128000},
		},
	}
}

func (p *deepseekProvider) Capability(model string) Capability {
	for _, m := range p.Models() {
		if m.ID == model {
			return m.Caps
		}
	}
	return Capability{}
}

func (p *deepseekProvider) AuthStatus() AuthStatus {
	token := p.client.Auth().GetToken()
	if token == "" {
		return AuthStatus{
			Provider: "deepseek",
			Ready:    false,
			Message:  "DeepSeek token sozlanmagan (nexus auth login)",
		}
	}
	return AuthStatus{
		Provider: "deepseek",
		Ready:    true,
		Message:  "DeepSeek token konfiguratsiya qilindi",
	}
}

// ChatStream forwards the request to the DeepSeek web client and translates
// chunks into core.ChatChunk. Unknown model ids are resolved by
// deepseek.ResolveModelConfig, so a CCR-chosen gemini/kimi model gracefully
// falls back to the closest DeepSeek equivalent when only this provider exists.
func (p *deepseekProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	dreq := deepseek.ChatStreamRequest{
		Prompt:          req.Prompt,
		Model:           req.Model,
		SessionID:       req.SessionID,
		ThinkingEnabled: req.ThinkingEnabled,
		SearchEnabled:   req.SearchEnabled,
	}

	chunkCh, errCh := p.client.ChatStream(ctx, dreq)

	outChunk := make(chan ChatChunk, 32)
	outErr := make(chan error, 1)

	go func() {
		defer close(outChunk)
		defer close(outErr)

		for {
			select {
			case <-ctx.Done():
				outErr <- ctx.Err()
				return
			case chunk, ok := <-chunkCh:
				if !ok {
					return
				}
				outChunk <- ChatChunk{
					Content:          chunk.Content,
					ReasoningContent: chunk.ReasoningContent,
					SessionID:        chunk.SessionID,
					MessageID:        chunk.MessageID,
					FinishReason:     chunk.FinishReason,
				}
				if chunk.FinishReason != "" {
					return
				}
			case err, ok := <-errCh:
				if !ok {
					return
				}
				if err != nil {
					outErr <- err
					return
				}
			}
		}
	}()

	return outChunk, outErr
}
