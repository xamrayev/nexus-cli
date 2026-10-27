package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// AnthropicRequest represents a request to /v1/messages.
type AnthropicRequest struct {
	Model     string        `json:"model"`
	Messages  []interface{} `json:"messages"`
	System    interface{}   `json:"system,omitempty"`
	Stream    bool          `json:"stream"`
	Tools     []interface{} `json:"tools,omitempty"`
	MaxTokens *int          `json:"max_tokens,omitempty"`
}

func (s *Server) handleAnthropicMessages(w http.ResponseWriter, r *http.Request, rawBody []byte, agentID string) {
	var req AnthropicRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		http.Error(w, `{"error":{"type":"invalid_request_error","message":"Invalid JSON payload"}}`, http.StatusBadRequest)
		return
	}

	// Normalize messages and system prompt
	var messages []deepseek.ChatMessage
	var systemPrompt string

	if req.System != nil {
		switch v := req.System.(type) {
		case string:
			systemPrompt = v
		case []interface{}:
			var sb strings.Builder
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if t, ok := m["text"].(string); ok {
						sb.WriteString(t)
						sb.WriteString("\n")
					}
				}
			}
			systemPrompt = sb.String()
		}
	}

	session := s.sessionMgr.GetOrCreate(agentID)

	modelToUse := req.Model
	if session.ModelOverride != "" {
		modelToUse = session.ModelOverride
	}
	modelCfg := deepseek.ResolveModelConfig(modelToUse)

	identityPrompt := fmt.Sprintf("You are %s (%s), powering Claude Code CLI via Nexus-CLI bridge. When asked about your model or identity, you should state that you are DeepSeek (%s) running through Nexus-CLI.", modelCfg.ID, modelCfg.RealModel, modelCfg.RealModel)
	if systemPrompt != "" {
		systemPrompt = identityPrompt + "\n\n" + systemPrompt
	} else {
		systemPrompt = identityPrompt
	}

	if systemPrompt != "" {
		messages = append(messages, deepseek.ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, item := range req.Messages {
		mMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := mMap["role"].(string)
		content := ""

		switch c := mMap["content"].(type) {
		case string:
			content = c
		case []interface{}:
			var sb strings.Builder
			for _, b := range c {
				if bMap, ok := b.(map[string]interface{}); ok {
					bType, _ := bMap["type"].(string)
					switch bType {
					case "text":
						if t, ok := bMap["text"].(string); ok {
							sb.WriteString(t)
							sb.WriteString("\n")
						}
					case "tool_result":
						toolID, _ := bMap["tool_use_id"].(string)
						isErr, _ := bMap["is_error"].(bool)
						status := "success"
						if isErr {
							status = "error"
						}
						resContent := ""
						if rc, ok := bMap["content"].(string); ok {
							resContent = rc
						} else if rcArr, ok := bMap["content"].([]interface{}); ok {
							for _, sub := range rcArr {
								if subm, ok := sub.(map[string]interface{}); ok {
									if st, ok := subm["text"].(string); ok {
										resContent += st + "\n"
									}
								}
							}
						}
						sb.WriteString(fmt.Sprintf("\n[Tool Result (%s, status: %s)]:\n%s\n", toolID, status, strings.TrimSpace(resContent)))
					case "tool_use":
						tName, _ := bMap["name"].(string)
						tID, _ := bMap["id"].(string)
						inputBytes, _ := json.Marshal(bMap["input"])
						sb.WriteString(fmt.Sprintf("\nAssistant called tool '%s' (ID: %s) with arguments: %s\n", tName, tID, string(inputBytes)))
					}
				}
			}
			content = sb.String()
		}

		messages = append(messages, deepseek.ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	prompt, _ := formatMessages(messages, req.Tools)

	// Check for slash commands in the latest user message
	lastUserMsg := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = strings.TrimSpace(messages[i].Content)
			break
		}
	}

	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Handle /new command
	if lastUserMsg == "/new" {
		savedOverride := session.ModelOverride
		s.sessionMgr.Reset(agentID)
		session = s.sessionMgr.GetOrCreate(agentID)
		session.ModelOverride = savedOverride

		responseText := "🔄 **Yangi suhbat boshlandi.** Oldingi suhbat tarixi va konteksti tozalandi."
		s.sendImmediateAnthropicResponse(w, r, req.Stream, msgID, modelCfg.ID, responseText)
		return
	}

	// Handle /model or /models status command
	if lastUserMsg == "/model" || lastUserMsg == "/models" {
		responseText := fmt.Sprintf("🤖 **Joriy faol model:** `%s` (%s)\n\n📋 **Mavjud DeepSeek modellari:**\n  • `deepseek-reasoner` (qisqacha: `r1`, `sonnet`, `opus`) — DeepSeek R1 Thinking\n  • `deepseek-chat` (qisqacha: `v3`, `haiku`) — DeepSeek V3 Tezkor Chat\n  • `deepseek-reasoner-search` — R1 Thinking + Veb Qidiruv\n  • `deepseek-chat-search` — V3 + Veb Qidiruv\n\n💡 **Modelni o'zgartirish uchun:**\n`/model <nomi>` (masalan: `/model deepseek-chat` yoki `/model r1`)", modelCfg.ID, modelCfg.RealModel)
		s.sendImmediateAnthropicResponse(w, r, req.Stream, msgID, modelCfg.ID, responseText)
		return
	}

	// Handle /model <name> switch command
	if strings.HasPrefix(lastUserMsg, "/model ") {
		targetName := strings.TrimSpace(strings.TrimPrefix(lastUserMsg, "/model "))
		newCfg := deepseek.ResolveModelConfig(targetName)
		session.ModelOverride = newCfg.ID

		responseText := fmt.Sprintf("✅ **Model muvaffaqiyatli almashtirildi!**\n\nYangi faol model: `%s` (%s).\nBundan keyingi barcha so'rovlar ushbu model yordamida bajariladi.", newCfg.ID, newCfg.RealModel)
		s.sendImmediateAnthropicResponse(w, r, req.Stream, msgID, newCfg.ID, responseText)
		return
	}

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 1. message_start
		startEv := map[string]interface{}{
			"type": "message_start",
			"message": map[string]interface{}{
				"id":      msgID,
				"type":    "message",
				"role":    "assistant",
				"content": []interface{}{},
				"model":   modelCfg.ID,
				"usage":   map[string]int{"input_tokens": len(prompt) / 4, "output_tokens": 0},
			},
		}
		b, _ := json.Marshal(startEv)
		fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", b)
		flusher.Flush()

		chunkCh, errCh := s.client.ChatStream(r.Context(), deepseek.ChatStreamRequest{
			Prompt:          prompt,
			Model:           modelCfg.ID,
			SessionID:       session.ID,
			ThinkingEnabled: &modelCfg.ThinkingEnabled,
			SearchEnabled:   &modelCfg.SearchEnabled,
		})

		var accumulatedContent strings.Builder
		var accumulatedReasoning strings.Builder

		for {
			select {
			case <-r.Context().Done():
				return
			case err, ok := <-errCh:
				if ok && err != nil {
					return
				}
			case chunk, ok := <-chunkCh:
				if !ok {
					goto DoneStream
				}
				if chunk.SessionID != "" {
					session.ID = chunk.SessionID
				}
				if chunk.ReasoningContent != "" {
					accumulatedReasoning.WriteString(chunk.ReasoningContent)
				}
				if chunk.Content != "" {
					accumulatedContent.WriteString(chunk.Content)
				}
			}
		}

	DoneStream:
		fullContent := accumulatedContent.String()
		if fullContent == "" && accumulatedReasoning.Len() > 0 {
			fullContent = accumulatedReasoning.String()
		}

		parsed := ParseToolCalls(fullContent)
		blockIndex := 0
		outputTokens := len(fullContent) / 4

		// 2. Emit text block if any clean text or if no tools
		if parsed.CleanText != "" || !parsed.HasTools {
			textVal := parsed.CleanText
			if !parsed.HasTools {
				textVal = fullContent
			}

			if textVal != "" {
				// content_block_start (text)
				blockStartEv := map[string]interface{}{
					"type":  "content_block_start",
					"index": blockIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				}
				b, _ := json.Marshal(blockStartEv)
				fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", b)

				// content_block_delta (text)
				deltaEv := map[string]interface{}{
					"type":  "content_block_delta",
					"index": blockIndex,
					"delta": map[string]interface{}{
						"type": "text_delta",
						"text": textVal,
					},
				}
				b, _ = json.Marshal(deltaEv)
				fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", b)

				// content_block_stop
				blockStopEv := map[string]interface{}{
					"type":  "content_block_stop",
					"index": blockIndex,
				}
				b, _ = json.Marshal(blockStopEv)
				fmt.Fprintf(w, "event: content_block_stop\ndata: %s\n\n", b)
				flusher.Flush()

				blockIndex++
			}
		}

		// 3. Emit tool_use blocks if tools detected
		if parsed.HasTools {
			for _, tc := range parsed.ToolCalls {
				// content_block_start (tool_use)
				toolStartEv := map[string]interface{}{
					"type":  "content_block_start",
					"index": blockIndex,
					"content_block": map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Name,
						"input": map[string]interface{}{},
					},
				}
				b, _ := json.Marshal(toolStartEv)
				fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", b)

				// content_block_delta (input_json_delta)
				argsBytes, _ := json.Marshal(tc.Arguments)
				toolDeltaEv := map[string]interface{}{
					"type":  "content_block_delta",
					"index": blockIndex,
					"delta": map[string]interface{}{
						"type":         "input_json_delta",
						"partial_json": string(argsBytes),
					},
				}
				b, _ = json.Marshal(toolDeltaEv)
				fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", b)

				// content_block_stop
				toolStopEv := map[string]interface{}{
					"type":  "content_block_stop",
					"index": blockIndex,
				}
				b, _ = json.Marshal(toolStopEv)
				fmt.Fprintf(w, "event: content_block_stop\ndata: %s\n\n", b)
				flusher.Flush()

				blockIndex++
			}
		}

		// 4. message_delta
		stopReason := "end_turn"
		if parsed.HasTools {
			stopReason = "tool_use"
		}
		msgDeltaEv := map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason":   stopReason,
				"stop_sequence": nil,
			},
			"usage": map[string]int{"output_tokens": outputTokens},
		}
		b, _ = json.Marshal(msgDeltaEv)
		fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", b)

		// 5. message_stop
		fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()

	} else {
		// Non-streaming Anthropic response
		res, err := s.client.Chat(r.Context(), prompt, modelCfg.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"type": "upstream_error", "message": err.Error()},
			})
			return
		}

		responseContent := res.Content
		if responseContent == "" && res.ReasoningContent != "" {
			responseContent = res.ReasoningContent
		}

		parsed := ParseToolCalls(responseContent)
		var contentBlocks []interface{}

		if parsed.CleanText != "" || !parsed.HasTools {
			textVal := parsed.CleanText
			if !parsed.HasTools {
				textVal = responseContent
			}
			if textVal != "" {
				contentBlocks = append(contentBlocks, map[string]string{
					"type": "text",
					"text": textVal,
				})
			}
		}

		stopReason := "end_turn"
		if parsed.HasTools {
			stopReason = "tool_use"
			for _, tc := range parsed.ToolCalls {
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": tc.Arguments,
				})
			}
		}

		resp := map[string]interface{}{
			"id":          msgID,
			"type":        "message",
			"role":        "assistant",
			"content":     contentBlocks,
			"model":       modelCfg.ID,
			"stop_reason": stopReason,
			"usage": map[string]int{
				"input_tokens":  len(prompt) / 4,
				"output_tokens": len(responseContent) / 4,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) sendImmediateAnthropicResponse(w http.ResponseWriter, r *http.Request, isStream bool, msgID, modelID, text string) {
	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		startEv := map[string]interface{}{
			"type": "message_start",
			"message": map[string]interface{}{
				"id":      msgID,
				"type":    "message",
				"role":    "assistant",
				"content": []interface{}{},
				"model":   modelID,
				"usage":   map[string]int{"input_tokens": 10, "output_tokens": len(text) / 4},
			},
		}
		b, _ := json.Marshal(startEv)
		fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", b)

		blockStartEv := map[string]interface{}{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]interface{}{
				"type": "text",
				"text": "",
			},
		}
		b, _ = json.Marshal(blockStartEv)
		fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", b)

		deltaEv := map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": text,
			},
		}
		b, _ = json.Marshal(deltaEv)
		fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", b)

		blockStopEv := map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		}
		b, _ = json.Marshal(blockStopEv)
		fmt.Fprintf(w, "event: content_block_stop\ndata: %s\n\n", b)

		msgDeltaEv := map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason":   "end_turn",
				"stop_sequence": nil,
			},
			"usage": map[string]int{"output_tokens": len(text) / 4},
		}
		b, _ = json.Marshal(msgDeltaEv)
		fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", b)

		fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
	} else {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"id":          msgID,
			"type":        "message",
			"role":        "assistant",
			"content":     []map[string]string{{"type": "text", "text": text}},
			"model":       modelID,
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": len(text) / 4,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
