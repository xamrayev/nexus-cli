package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nexus-cli/pkg/deepseek"
)

// OpenAIRequest represents a chat completions request.
type OpenAIRequest struct {
	Model          string                 `json:"model"`
	Messages       []deepseek.ChatMessage `json:"messages"`
	Stream         bool                   `json:"stream"`
	Tools          []interface{}          `json:"tools,omitempty"`
	User           string                 `json:"user,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	ResponseFormat json.RawMessage        `json:"response_format,omitempty"`
}

// OpenAIResponse represents a non-streaming chat completion response.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIMessage struct {
	Role             string              `json:"role"`
	Content          string              `json:"content"`
	ReasoningContent string              `json:"reasoning_content,omitempty"`
	ToolCalls        []deepseek.ToolCall `json:"tool_calls,omitempty"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamChunk represents an SSE chunk for streaming chat completion.
type OpenAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
}

type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

type OpenAIStreamDelta struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

func formatMessages(messages []deepseek.ChatMessage, tools []interface{}) (prompt, systemPrompt string) {
	var sysBuilder strings.Builder
	for _, m := range messages {
		if m.Role == "system" && m.Content != "" {
			sysBuilder.WriteString(m.Content)
			sysBuilder.WriteString("\n")
		}
	}

	if len(tools) > 0 {
		toolBytes, err := json.MarshalIndent(tools, "", "  ")
		if err == nil {
			sysBuilder.WriteString("\n[Available Tools]:\n")
			sysBuilder.WriteString(string(toolBytes))
			sysBuilder.WriteString("\nIf you need to call a tool, you may output using either standard DeepSeek format (<｜｜DSML｜｜ calls><｜｜DSML｜｜ invoke name=\"tool_name\"><｜｜DSML｜｜ parameter name=\"param\" string=\"true\">value</｜｜DSML｜｜ parameter></｜｜DSML｜｜ invoke></｜｜DSML｜｜ calls>) or a JSON object: {\"tool_calls\": [{\"name\": \"tool_name\", \"arguments\": {...}}]}\n")
		}
	}

	var convBuilder strings.Builder
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		if m.Role == "user" && m.Content != "" {
			convBuilder.WriteString(fmt.Sprintf("User: %s\n\n", m.Content))
		} else if m.Role == "assistant" {
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					convBuilder.WriteString(fmt.Sprintf("Assistant: TOOL_CALL: %s\narguments: %s\n\n", tc.Function.Name, tc.Function.Arguments))
				}
			} else if m.Content != "" {
				convBuilder.WriteString(fmt.Sprintf("Assistant: %s\n\n", m.Content))
			}
		} else if m.Role == "tool" && m.Content != "" {
			convBuilder.WriteString(fmt.Sprintf("[Tool Result]:\n%s\n\n", m.Content))
		}
	}

	p := strings.TrimSpace(convBuilder.String())
	s := strings.TrimSpace(sysBuilder.String())
	if s != "" {
		p = fmt.Sprintf("[Instructions]:\n%s\n\n%s", s, p)
	}

	return p, s
}

func (s *Server) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request, req OpenAIRequest, agentID string) {
	// Check "/new" command
	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = strings.TrimSpace(req.Messages[i].Content)
			break
		}
	}

	if lastUserMsg == "/new" {
		s.sessionMgr.Reset(agentID)
		msg := "Started a new chat. Session and history have been reset."
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, _ := w.(http.Flusher)

			chunk := OpenAIStreamChunk{
				ID:      "chatcmpl-reset",
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []OpenAIStreamChoice{
					{
						Index: 0,
						Delta: OpenAIStreamDelta{Content: msg},
					},
				},
			}
			b, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return
		}

		resp := OpenAIResponse{
			ID:      "chatcmpl-reset",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []OpenAIChoice{
				{
					Index: 0,
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: msg,
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	prompt, _ := formatMessages(req.Messages, req.Tools)
	if len(req.ResponseFormat) > 0 {
		prompt += BuildJSONInstruction(req.ResponseFormat)
	}
	if prompt == "" {
		http.Error(w, `{"error":{"message":"Prompt bo'sh bo'lmasligi kerak","type":"invalid_request_error"}}`, http.StatusBadRequest)
		return
	}

	session := s.sessionMgr.GetOrCreate(agentID)
	modelCfg := deepseek.ResolveModelConfig(req.Model)

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		chunkCh, errCh := s.client.ChatStream(r.Context(), deepseek.ChatStreamRequest{
			Prompt:          prompt,
			Model:           modelCfg.ID,
			SessionID:       session.ID,
			ThinkingEnabled: &modelCfg.ThinkingEnabled,
			SearchEnabled:   &modelCfg.SearchEnabled,
		})

		respID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
		created := time.Now().Unix()

		// Send initial role delta
		initialChunk := OpenAIStreamChunk{
			ID:      respID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   modelCfg.ID,
			Choices: []OpenAIStreamChoice{
				{
					Index: 0,
					Delta: OpenAIStreamDelta{Role: "assistant"},
				},
			},
		}
		initialBytes, _ := json.Marshal(initialChunk)
		fmt.Fprintf(w, "data: %s\n\n", initialBytes)
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case err, ok := <-errCh:
				if ok && err != nil {
					errData, _ := json.Marshal(map[string]interface{}{
						"error": map[string]string{"message": err.Error(), "type": "upstream_error"},
					})
					fmt.Fprintf(w, "data: %s\n\n", errData)
					flusher.Flush()
					return
				}
			case chunk, ok := <-chunkCh:
				if !ok {
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
					return
				}

				if chunk.SessionID != "" {
					session.ID = chunk.SessionID
				}

				streamChunk := OpenAIStreamChunk{
					ID:      respID,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   modelCfg.ID,
					Choices: []OpenAIStreamChoice{
						{
							Index: 0,
							Delta: OpenAIStreamDelta{
								Content:          chunk.Content,
								ReasoningContent: chunk.ReasoningContent,
							},
						},
					},
				}
				if chunk.FinishReason != "" {
					reason := chunk.FinishReason
					streamChunk.Choices[0].FinishReason = &reason
				}

				b, _ := json.Marshal(streamChunk)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}
		}

	} else {
		// Non-streaming response
		chunkCh, errCh := s.client.ChatStream(r.Context(), deepseek.ChatStreamRequest{
			Prompt:          prompt,
			Model:           modelCfg.ID,
			SessionID:       session.ID,
			ThinkingEnabled: &modelCfg.ThinkingEnabled,
			SearchEnabled:   &modelCfg.SearchEnabled,
		})

		var contentBuilder, reasoningBuilder strings.Builder
		for {
			select {
			case <-r.Context().Done():
				return
			case err, ok := <-errCh:
				if ok && err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadGateway)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]string{"message": err.Error(), "type": "upstream_error"},
					})
					return
				}
			case chunk, ok := <-chunkCh:
				if !ok {
					fullContent := contentBuilder.String()
					fullReasoning := reasoningBuilder.String()

					if len(req.ResponseFormat) > 0 {
						if cleaned, err := CleanJSON(fullContent); err == nil {
							fullContent = cleaned
						}
					}

					resp := OpenAIResponse{
						ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
						Object:  "chat.completion",
						Created: time.Now().Unix(),
						Model:   modelCfg.ID,
						Choices: []OpenAIChoice{
							{
								Index: 0,
								Message: OpenAIMessage{
									Role:             "assistant",
									Content:          fullContent,
									ReasoningContent: fullReasoning,
								},
								FinishReason: "stop",
							},
						},
						Usage: OpenAIUsage{
							PromptTokens:     len(prompt) / 4,
							CompletionTokens: len(fullContent) / 4,
							TotalTokens:      (len(prompt) + len(fullContent)) / 4,
						},
					}

					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				if chunk.SessionID != "" {
					session.ID = chunk.SessionID
				}
				if chunk.Content != "" {
					contentBuilder.WriteString(chunk.Content)
				}
				if chunk.ReasoningContent != "" {
					reasoningBuilder.WriteString(chunk.ReasoningContent)
				}
			}
		}
	}
}
