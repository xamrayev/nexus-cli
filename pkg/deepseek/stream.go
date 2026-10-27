package deepseek

import (
	"encoding/json"
	"fmt"
	"strings"
)

const fragmentContentPath = "response/fragments/-1/content"

// StreamParser parses Server-Sent Events (SSE) lines from DeepSeek Web API.
type StreamParser struct {
	event        string
	path         string
	fragmentType string
	messageID    string
}

// NewStreamParser creates a fresh stateful parser for one completion stream.
func NewStreamParser() *StreamParser {
	return &StreamParser{}
}

// Feed processes a single line from the SSE stream.
// Returns a non-nil ChatChunk if the line carries text or finish signal.
func (p *StreamParser) Feed(line string) (*ChatChunk, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	if strings.HasPrefix(line, "event:") {
		p.event = strings.TrimSpace(line[6:])
		return nil, nil
	}

	if !strings.HasPrefix(line, "data:") {
		return nil, nil
	}

	event := p.event
	p.event = ""

	payload := strings.TrimSpace(line[5:])
	if payload == "[DONE]" {
		return &ChatChunk{FinishReason: "stop"}, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, nil
	}

	// Filter out metadata events
	switch event {
	case "hint":
		if errType, ok := raw["type"].(string); ok && errType == "error" {
			errMsg, _ := raw["content"].(string)
			if errMsg == "" {
				errMsg = "server xatosi"
			}
			return nil, fmt.Errorf("DeepSeek xatoligi: %s", errMsg)
		}
		return nil, nil
	case "title", "update_session", "close", "keep-alive":
		return nil, nil
	case "ready":
		if msgID, ok := raw["response_message_id"]; ok && msgID != nil {
			p.messageID = fmt.Sprintf("%v", msgID)
		}
		return nil, nil
	}

	// Format 1: OpenAI delta format
	if choicesRaw, ok := raw["choices"].([]interface{}); ok && len(choicesRaw) > 0 {
		if choice, ok := choicesRaw[0].(map[string]interface{}); ok {
			delta, _ := choice["delta"].(map[string]interface{})
			finishReason, _ := choice["finish_reason"].(string)

			content, _ := delta["content"].(string)
			reasoning, _ := delta["reasoning_content"].(string)
			deltaType, _ := delta["type"].(string)

			if deltaType == "thinking" || reasoning != "" {
				if reasoning == "" {
					reasoning = content
					content = ""
				}
				return &ChatChunk{
					ReasoningContent: reasoning,
					FinishReason:     finishReason,
					MessageID:        p.messageID,
				}, nil
			}

			if content != "" || finishReason != "" {
				return &ChatChunk{
					Content:      content,
					FinishReason: finishReason,
					MessageID:    p.messageID,
				}, nil
			}
		}
	}

	// Format 2: DeepSeek web JSON patch format
	if _, ok := raw["v"]; ok {
		return p.applyPatch(raw)
	}

	return nil, nil
}

func (p *StreamParser) applyPatch(data map[string]interface{}) (*ChatChunk, error) {
	if pVal, ok := data["p"].(string); ok {
		p.path = pVal
	}

	val := data["v"]

	// Snapshot object: {"v": {"response": {"fragments": [...], "message_id": ...}}}
	if vMap, ok := val.(map[string]interface{}); ok {
		if respMap, ok := vMap["response"].(map[string]interface{}); ok {
			if mID, ok := respMap["message_id"]; ok && mID != nil {
				p.messageID = fmt.Sprintf("%v", mID)
			}
			if frags, ok := respMap["fragments"].([]interface{}); ok {
				return p.parseFragments(frags), nil
			}
		}
	}

	// Appending fragments list: {"p": "response/fragments", "v": [...]}
	if p.path == "response/fragments" {
		if frags, ok := val.([]interface{}); ok {
			p.path = fragmentContentPath
			return p.parseFragments(frags), nil
		}
	}

	// Incremental text on latest fragment: {"p": "response/fragments/-1/content", "v": "..."}
	if p.path == fragmentContentPath {
		if text, ok := val.(string); ok {
			return p.createTextChunk(p.fragmentType, text), nil
		}
	}

	// Legacy paths
	if p.path == "response/content" || p.path == "response/thinking_content" {
		if text, ok := val.(string); ok {
			fType := "RESPONSE"
			if strings.Contains(p.path, "thinking") {
				fType = "THINK"
			}
			return p.createTextChunk(fType, text), nil
		}
	}

	// Finished status
	if p.path == "response/status" {
		if statusStr, ok := val.(string); ok && statusStr == "FINISHED" {
			return &ChatChunk{FinishReason: "stop", MessageID: p.messageID}, nil
		}
	}

	return nil, nil
}

func (p *StreamParser) parseFragments(frags []interface{}) *ChatChunk {
	var contentBuilder, reasoningBuilder strings.Builder

	for _, item := range frags {
		fMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if t, ok := fMap["type"].(string); ok {
			p.fragmentType = t
		}

		text, ok := fMap["content"].(string)
		if !ok {
			continue
		}

		switch p.fragmentType {
		case "THINK", "THINKING", "REASONING":
			reasoningBuilder.WriteString(text)
		case "RESPONSE", "TEXT", "SEARCH":
			contentBuilder.WriteString(text)
		}
	}

	content := contentBuilder.String()
	reasoning := reasoningBuilder.String()

	if content == "" && reasoning == "" {
		return nil
	}

	return &ChatChunk{
		Content:          content,
		ReasoningContent: reasoning,
		MessageID:        p.messageID,
	}
}

func (p *StreamParser) createTextChunk(fType, text string) *ChatChunk {
	switch fType {
	case "THINK", "THINKING", "REASONING":
		return &ChatChunk{
			ReasoningContent: text,
			MessageID:        p.messageID,
		}
	case "RESPONSE", "TEXT", "SEARCH":
		fallthrough
	default:
		return &ChatChunk{
			Content:   text,
			MessageID: p.messageID,
		}
	}
}
