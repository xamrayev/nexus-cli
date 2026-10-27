package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// OllamaMessage represents a message in Ollama chat API.
type OllamaMessage struct {
	Role     string   `json:"role"`
	Content  string   `json:"content"`
	Thinking string   `json:"thinking,omitempty"`
	Images   []string `json:"images,omitempty"`
}

// OllamaChatRequest represents request to /api/chat.
type OllamaChatRequest struct {
	Model     string          `json:"model"`
	Messages  []OllamaMessage `json:"messages"`
	Stream    *bool           `json:"stream,omitempty"`
	Format    json.RawMessage `json:"format,omitempty"`
	Think     *bool           `json:"think,omitempty"`
	KeepAlive string          `json:"keep_alive,omitempty"`
	Options   map[string]any  `json:"options,omitempty"`
}

// OllamaChatResponse represents streaming and non-streaming response for /api/chat.
type OllamaChatResponse struct {
	Model              string        `json:"model"`
	CreatedAt          time.Time     `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

// OllamaGenerateRequest represents request to /api/generate.
type OllamaGenerateRequest struct {
	Model     string          `json:"model"`
	Prompt    string          `json:"prompt"`
	System    string          `json:"system,omitempty"`
	Template  string          `json:"template,omitempty"`
	Context   []int           `json:"context,omitempty"`
	Stream    *bool           `json:"stream,omitempty"`
	Raw       bool            `json:"raw,omitempty"`
	Format    json.RawMessage `json:"format,omitempty"`
	Think     *bool           `json:"think,omitempty"`
	KeepAlive string          `json:"keep_alive,omitempty"`
	Options   map[string]any  `json:"options,omitempty"`
}

// OllamaGenerateResponse represents response for /api/generate.
type OllamaGenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Thinking           string    `json:"thinking,omitempty"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason,omitempty"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// OllamaModelDetails represents model metadata.
type OllamaModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// OllamaModelItem represents an item in /api/tags or /api/ps.
type OllamaModelItem struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	ModifiedAt time.Time          `json:"modified_at"`
	Size       int64              `json:"size"`
	Digest     string             `json:"digest"`
	Details    OllamaModelDetails `json:"details"`
	ExpiresAt  string             `json:"expires_at,omitempty"`
	SizeVRAM   int64              `json:"size_vram,omitempty"`
}

// OllamaTagsResponse represents /api/tags response.
type OllamaTagsResponse struct {
	Models []OllamaModelItem `json:"models"`
}

// OllamaShowResponse represents /api/show response.
type OllamaShowResponse struct {
	Modelfile  string             `json:"modelfile"`
	Parameters string             `json:"parameters"`
	Template   string             `json:"template"`
	Details    OllamaModelDetails `json:"details"`
}

// OllamaVersionResponse represents /api/version response.
type OllamaVersionResponse struct {
	Version string `json:"version"`
}

func getSupportedOllamaModels() []OllamaModelItem {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	details := OllamaModelDetails{
		ParentModel:       "",
		Format:            "gguf",
		Family:            "deepseek",
		Families:          []string{"deepseek"},
		ParameterSize:     "671B",
		QuantizationLevel: "FP8",
	}

	return []OllamaModelItem{
		{
			Name:       "deepseek-reasoner:latest",
			Model:      "deepseek-reasoner:latest",
			ModifiedAt: now,
			Size:       671000000000,
			Digest:     "sha256:nexus-deepseek-r1-reasoner",
			Details:    details,
		},
		{
			Name:       "deepseek-reasoner",
			Model:      "deepseek-reasoner",
			ModifiedAt: now,
			Size:       671000000000,
			Digest:     "sha256:nexus-deepseek-r1-reasoner",
			Details:    details,
		},
		{
			Name:       "deepseek-chat:latest",
			Model:      "deepseek-chat:latest",
			ModifiedAt: now,
			Size:       671000000000,
			Digest:     "sha256:nexus-deepseek-v3-chat",
			Details:    details,
		},
		{
			Name:       "deepseek-chat",
			Model:      "deepseek-chat",
			ModifiedAt: now,
			Size:       671000000000,
			Digest:     "sha256:nexus-deepseek-v3-chat",
			Details:    details,
		},
		{
			Name:       "nexus:latest",
			Model:      "nexus:latest",
			ModifiedAt: now,
			Size:       671000000000,
			Digest:     "sha256:nexus-deepseek-bridge",
			Details:    details,
		},
	}
}

// handleOllamaTags handles GET /api/tags
func (s *Server) handleOllamaTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := OllamaTagsResponse{
		Models: getSupportedOllamaModels(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOllamaVersion handles GET /api/version
func (s *Server) handleOllamaVersion(w http.ResponseWriter, r *http.Request) {
	resp := OllamaVersionResponse{
		Version: "0.1.32", // Emulate compatible Ollama version
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOllamaPs handles GET /api/ps
func (s *Server) handleOllamaPs(w http.ResponseWriter, r *http.Request) {
	models := getSupportedOllamaModels()
	for i := range models {
		models[i].ExpiresAt = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		models[i].SizeVRAM = models[i].Size
	}

	resp := OllamaTagsResponse{
		Models: models[:2],
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOllamaShow handles POST /api/show
func (s *Server) handleOllamaShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Model string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	modelName := body.Name
	if modelName == "" {
		modelName = body.Model
	}
	if modelName == "" {
		modelName = "deepseek-reasoner"
	}

	resp := OllamaShowResponse{
		Modelfile:  fmt.Sprintf("# Modelfile for %s (Emulated by Nexus)\nFROM %s\nPARAMETER temperature 0.7\n", modelName, modelName),
		Parameters: "temperature 0.7\nstop <|im_end|>\n",
		Template:   "{{ if .System }}<|im_start|>system\n{{ .System }}<|im_end|>\n{{ end }}{{ if .Prompt }}<|im_start|>user\n{{ .Prompt }}<|im_end|>\n{{ end }}<|im_start|>assistant\n",
		Details: OllamaModelDetails{
			ParentModel:       "",
			Format:            "gguf",
			Family:            "deepseek",
			Families:          []string{"deepseek"},
			ParameterSize:     "671B",
			QuantizationLevel: "FP8",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOllamaChat handles POST /api/chat
func (s *Server) handleOllamaChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OllamaChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Stream defaults to true in Ollama
	stream := true
	if req.Stream != nil {
		stream = *req.Stream
	}

	// Format messages into prompt
	var sysBuilder strings.Builder
	var convBuilder strings.Builder

	for _, m := range req.Messages {
		if strings.EqualFold(m.Role, "system") {
			sysBuilder.WriteString(m.Content)
			sysBuilder.WriteString("\n")
		} else if strings.EqualFold(m.Role, "user") {
			convBuilder.WriteString(fmt.Sprintf("User: %s\n\n", m.Content))
		} else if strings.EqualFold(m.Role, "assistant") {
			convBuilder.WriteString(fmt.Sprintf("Assistant: %s\n\n", m.Content))
		}
	}

	// Structured Output instruction injection if requested
	if len(req.Format) > 0 {
		sysBuilder.WriteString(BuildJSONInstruction(req.Format))
	}

	prompt := strings.TrimSpace(convBuilder.String())
	sysPrompt := strings.TrimSpace(sysBuilder.String())
	if sysPrompt != "" {
		prompt = fmt.Sprintf("[Instructions]:\n%s\n\n%s", sysPrompt, prompt)
	}

	if prompt == "" {
		http.Error(w, `{"error":"No prompt or messages provided"}`, http.StatusBadRequest)
		return
	}

	// Resolve model
	modelName := req.Model
	if modelName == "" {
		modelName = "deepseek-reasoner"
	}
	modelCfg := deepseek.ResolveModelConfig(modelName)

	// If think is explicitly false, disable thinking
	if req.Think != nil && !*req.Think {
		modelCfg.ThinkingEnabled = false
	}

	agentID := "ollama-client"
	session := s.sessionMgr.GetOrCreate(agentID)

	startTime := time.Now()

	if stream {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		s.sem <- struct{}{}
		defer func() { <-s.sem }()

		chunkCh, errCh := s.client.ChatStream(ctx, deepseek.ChatStreamRequest{
			Prompt:          prompt,
			Model:           modelCfg.ID,
			SessionID:       session.ID,
			ThinkingEnabled: &modelCfg.ThinkingEnabled,
			SearchEnabled:   &modelCfg.SearchEnabled,
		})

		evalCount := 0

		for chunk := range chunkCh {
			if chunk.SessionID != "" {
				session.ID = chunk.SessionID
			}

			if chunk.ReasoningContent != "" {
				resp := OllamaChatResponse{
					Model:     modelName,
					CreatedAt: time.Now().UTC(),
					Done:      false,
					Message: OllamaMessage{
						Role:     "assistant",
						Thinking: chunk.ReasoningContent,
					},
				}
				line, _ := json.Marshal(resp)
				fmt.Fprintf(w, "%s\n", line)
				flusher.Flush()
			}

			if chunk.Content != "" {
				evalCount++
				resp := OllamaChatResponse{
					Model:     modelName,
					CreatedAt: time.Now().UTC(),
					Done:      false,
					Message: OllamaMessage{
						Role:    "assistant",
						Content: chunk.Content,
					},
				}
				line, _ := json.Marshal(resp)
				fmt.Fprintf(w, "%s\n", line)
				flusher.Flush()
			}
		}

		if err := <-errCh; err != nil {
			errResp := map[string]string{"error": err.Error()}
			line, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
			return
		}

		// Final done chunk
		totalDuration := time.Since(startTime)
		doneResp := OllamaChatResponse{
			Model:              modelName,
			CreatedAt:          time.Now().UTC(),
			Done:               true,
			DoneReason:         "stop",
			TotalDuration:      totalDuration.Nanoseconds(),
			LoadDuration:       1000000,
			PromptEvalCount:    len(strings.Fields(prompt)),
			PromptEvalDuration: 5000000,
			EvalCount:          evalCount,
			EvalDuration:       totalDuration.Nanoseconds(),
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "",
			},
		}
		doneLine, _ := json.Marshal(doneResp)
		fmt.Fprintf(w, "%s\n", doneLine)
		flusher.Flush()
		return
	}

	// Non-streaming response
	ctx := r.Context()
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	chunkCh, errCh := s.client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           modelCfg.ID,
		SessionID:       session.ID,
		ThinkingEnabled: &modelCfg.ThinkingEnabled,
		SearchEnabled:   &modelCfg.SearchEnabled,
	})

	var fullContent strings.Builder
	var fullThinking strings.Builder
	evalCount := 0

	for chunk := range chunkCh {
		if chunk.SessionID != "" {
			session.ID = chunk.SessionID
		}
		if chunk.ReasoningContent != "" {
			fullThinking.WriteString(chunk.ReasoningContent)
		}
		if chunk.Content != "" {
			fullContent.WriteString(chunk.Content)
			evalCount++
		}
	}

	if err := <-errCh; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	finalContent := fullContent.String()
	// Clean JSON if structured format requested
	if len(req.Format) > 0 {
		if cleaned, err := CleanJSON(finalContent); err == nil {
			finalContent = cleaned
		}
	}

	totalDuration := time.Since(startTime)
	resp := OllamaChatResponse{
		Model:              modelName,
		CreatedAt:          time.Now().UTC(),
		Done:               true,
		DoneReason:         "stop",
		TotalDuration:      totalDuration.Nanoseconds(),
		LoadDuration:       1000000,
		PromptEvalCount:    len(strings.Fields(prompt)),
		PromptEvalDuration: 5000000,
		EvalCount:          evalCount,
		EvalDuration:       totalDuration.Nanoseconds(),
		Message: OllamaMessage{
			Role:     "assistant",
			Content:  finalContent,
			Thinking: fullThinking.String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleOllamaGenerate handles POST /api/generate
func (s *Server) handleOllamaGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OllamaGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	stream := true
	if req.Stream != nil {
		stream = *req.Stream
	}

	prompt := strings.TrimSpace(req.Prompt)
	sysPrompt := strings.TrimSpace(req.System)
	if len(req.Format) > 0 {
		sysPrompt += BuildJSONInstruction(req.Format)
	}

	if sysPrompt != "" {
		prompt = fmt.Sprintf("[Instructions]:\n%s\n\nUser: %s", sysPrompt, prompt)
	}

	if prompt == "" {
		http.Error(w, `{"error":"No prompt provided"}`, http.StatusBadRequest)
		return
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "deepseek-reasoner"
	}
	modelCfg := deepseek.ResolveModelConfig(modelName)
	if req.Think != nil && !*req.Think {
		modelCfg.ThinkingEnabled = false
	}

	agentID := "ollama-generate"
	session := s.sessionMgr.GetOrCreate(agentID)
	startTime := time.Now()

	if stream {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		s.sem <- struct{}{}
		defer func() { <-s.sem }()

		chunkCh, errCh := s.client.ChatStream(ctx, deepseek.ChatStreamRequest{
			Prompt:          prompt,
			Model:           modelCfg.ID,
			SessionID:       session.ID,
			ThinkingEnabled: &modelCfg.ThinkingEnabled,
			SearchEnabled:   &modelCfg.SearchEnabled,
		})

		evalCount := 0
		for chunk := range chunkCh {
			if chunk.SessionID != "" {
				session.ID = chunk.SessionID
			}

			if chunk.ReasoningContent != "" {
				resp := OllamaGenerateResponse{
					Model:     modelName,
					CreatedAt: time.Now().UTC(),
					Thinking:  chunk.ReasoningContent,
					Done:      false,
				}
				line, _ := json.Marshal(resp)
				fmt.Fprintf(w, "%s\n", line)
				flusher.Flush()
			}

			if chunk.Content != "" {
				evalCount++
				resp := OllamaGenerateResponse{
					Model:     modelName,
					CreatedAt: time.Now().UTC(),
					Response:  chunk.Content,
					Done:      false,
				}
				line, _ := json.Marshal(resp)
				fmt.Fprintf(w, "%s\n", line)
				flusher.Flush()
			}
		}

		if err := <-errCh; err != nil {
			errResp := map[string]string{"error": err.Error()}
			line, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
			return
		}

		totalDuration := time.Since(startTime)
		doneResp := OllamaGenerateResponse{
			Model:              modelName,
			CreatedAt:          time.Now().UTC(),
			Done:               true,
			DoneReason:         "stop",
			TotalDuration:      totalDuration.Nanoseconds(),
			LoadDuration:       1000000,
			PromptEvalCount:    len(strings.Fields(prompt)),
			PromptEvalDuration: 5000000,
			EvalCount:          evalCount,
			EvalDuration:       totalDuration.Nanoseconds(),
		}
		doneLine, _ := json.Marshal(doneResp)
		fmt.Fprintf(w, "%s\n", doneLine)
		flusher.Flush()
		return
	}

	// Non-streaming
	ctx := r.Context()
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	chunkCh, errCh := s.client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           modelCfg.ID,
		SessionID:       session.ID,
		ThinkingEnabled: &modelCfg.ThinkingEnabled,
		SearchEnabled:   &modelCfg.SearchEnabled,
	})

	var fullResponse strings.Builder
	var fullThinking strings.Builder
	evalCount := 0

	for chunk := range chunkCh {
		if chunk.SessionID != "" {
			session.ID = chunk.SessionID
		}
		if chunk.ReasoningContent != "" {
			fullThinking.WriteString(chunk.ReasoningContent)
		}
		if chunk.Content != "" {
			fullResponse.WriteString(chunk.Content)
			evalCount++
		}
	}

	if err := <-errCh; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	finalRespText := fullResponse.String()
	if len(req.Format) > 0 {
		if cleaned, err := CleanJSON(finalRespText); err == nil {
			finalRespText = cleaned
		}
	}

	totalDuration := time.Since(startTime)
	resp := OllamaGenerateResponse{
		Model:              modelName,
		CreatedAt:          time.Now().UTC(),
		Response:           finalRespText,
		Thinking:           fullThinking.String(),
		Done:               true,
		DoneReason:         "stop",
		TotalDuration:      totalDuration.Nanoseconds(),
		LoadDuration:       1000000,
		PromptEvalCount:    len(strings.Fields(prompt)),
		PromptEvalDuration: 5000000,
		EvalCount:          evalCount,
		EvalDuration:       totalDuration.Nanoseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
