package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client interacts with DeepSeek Web API endpoints.
type Client struct {
	auth        *AuthManager
	httpClient  *http.Client
	baseURL     string
	sessionID   string
	parentMsgID *int64
	pool        *AccountPool
	currentAcc  *Account
	mu          sync.Mutex
}

// ClientOptions configures the Client.
type ClientOptions struct {
	Token    string
	AuthPath string
	Timeout  time.Duration
	BaseURL  string
	Pool     *AccountPool
	Account  *Account
}

// NewClient creates a new DeepSeek Web client.
func NewClient(opts ClientOptions) *Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	auth := NewAuthManager(opts.Token, opts.AuthPath)

	return &Client{
		auth:       auth,
		pool:       opts.Pool,
		currentAcc: opts.Account,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Pool returns the AccountPool instance if set.
func (c *Client) Pool() *AccountPool {
	return c.pool
}

// SetPool sets the AccountPool.
func (c *Client) SetPool(pool *AccountPool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pool = pool
}

// SetAccount pins a specific Account to this client.
func (c *Client) SetAccount(acc *Account) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentAcc = acc
}

// CurrentAccount returns the pinned Account if any.
func (c *Client) CurrentAccount() *Account {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentAcc
}

// Auth returns the AuthManager instance.
func (c *Client) Auth() *AuthManager {
	return c.auth
}

// SetSessionID sets the current session ID manually.
func (c *Client) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
	c.parentMsgID = nil
}

// GetSessionID gets the current session ID.
func (c *Client) GetSessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// ResetSession clears the active session to start a fresh chat chain.
func (c *Client) ResetSession() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = ""
	c.parentMsgID = nil
}

func (c *Client) prepareHeaders(powResponse string) http.Header {
	h := make(http.Header)

	c.mu.Lock()
	authData := c.auth.GetData()
	if c.currentAcc != nil {
		authData = c.currentAcc.Data
	}
	c.mu.Unlock()

	h.Set("User-Agent", DefaultUserAgent)
	h.Set("Accept", "*/*")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Origin", "https://chat.deepseek.com")
	h.Set("Referer", "https://chat.deepseek.com/")
	h.Set("x-client-platform", "web")
	h.Set("x-client-version", "2.0.0")
	h.Set("x-client-locale", "en")
	h.Set("x-client-timezone-offset", "14400")
	h.Set("x-app-version", "2.0.0")
	h.Set("Content-Type", "application/json")

	if authData.Token != "" {
		h.Set("Authorization", fmt.Sprintf("Bearer %s", authData.Token))
	}
	if authData.Cookie != "" {
		h.Set("Cookie", authData.Cookie)
	}
	if authData.HifDliq != "" {
		h.Set("x-hif-dliq", authData.HifDliq)
	}
	if authData.HifLeim != "" {
		h.Set("x-hif-leim", authData.HifLeim)
	}
	if powResponse != "" {
		h.Set("x-ds-pow-response", powResponse)
	}

	return h
}

// CreateSession requests a new chat_session_id from DeepSeek.
func (c *Client) CreateSession(ctx context.Context) (string, error) {
	return c.createSessionInternal(ctx, false)
}

func (c *Client) createSessionInternal(ctx context.Context, isRetry bool) (string, error) {
	c.mu.Lock()
	if c.pool != nil && c.currentAcc == nil {
		c.currentAcc = c.pool.NextAvailable()
	}
	c.mu.Unlock()

	reqURL := fmt.Sprintf("%s/api/v0/chat_session/create", c.baseURL)
	headers := c.prepareHeaders("")

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader("{}"))
	if err != nil {
		return "", err
	}
	req.Header = headers

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create_session so'rovi amalga oshmadi: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		if (resp.StatusCode == 429 || resp.StatusCode == 401 || resp.StatusCode == 403) && c.pool != nil {
			c.mu.Lock()
			curAcc := c.currentAcc
			c.mu.Unlock()
			if curAcc != nil {
				c.pool.MarkFailure(curAcc, resp.StatusCode, resp.Header.Get("Retry-After"))
			}
			if !isRetry {
				c.mu.Lock()
				c.currentAcc = c.pool.NextAvailable()
				c.mu.Unlock()
				return c.createSessionInternal(ctx, true)
			}
		}
		return "", fmt.Errorf("chat_session/create xatoligi (HTTP %d): %s", resp.StatusCode, string(bodyBytes[:min(len(bodyBytes), 200)]))
	}

	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BizData struct {
				ID          string `json:"id"`
				ChatSession struct {
					ID string `json:"id"`
				} `json:"chat_session"`
			} `json:"biz_data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", fmt.Errorf("kutilmagan session javobi: %w", err)
	}

	if res.Code != 0 {
		if (res.Code == 40003 || res.Code == 40002) && c.pool != nil {
			c.mu.Lock()
			curAcc := c.currentAcc
			c.mu.Unlock()
			if curAcc != nil {
				c.pool.MarkFailure(curAcc, 401, "")
			}
			if !isRetry {
				c.mu.Lock()
				c.currentAcc = c.pool.NextAvailable()
				c.mu.Unlock()
				return c.createSessionInternal(ctx, true)
			}
		}
		return "", fmt.Errorf("chat sessiyasi yaratilmadi (%d): %s", res.Code, res.Msg)
	}

	sID := res.Data.BizData.ChatSession.ID
	if sID == "" {
		sID = res.Data.BizData.ID
	}

	c.mu.Lock()
	c.sessionID = sID
	c.parentMsgID = nil
	c.mu.Unlock()

	return sID, nil
}

// RequestAndSolvePoW requests a PoW challenge from /api/v0/chat/create_pow_challenge and solves it.
func (c *Client) RequestAndSolvePoW(ctx context.Context, targetPath string) (string, error) {
	reqURL := fmt.Sprintf("%s/api/v0/chat/create_pow_challenge", c.baseURL)
	headers := c.prepareHeaders("")

	payload := map[string]string{"target_path": targetPath}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header = headers

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create_pow_challenge ulanish xatoligi: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create_pow_challenge HTTP xatosi (%d): %s", resp.StatusCode, string(bodyBytes[:min(len(bodyBytes), 200)]))
	}

	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BizData map[string]interface{} `json:"biz_data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", fmt.Errorf("PoW javobini o'qib bo'lmadi: %w", err)
	}

	if res.Code != 0 {
		return "", fmt.Errorf("PoW challenge olinmadi (%d): %s", res.Code, res.Msg)
	}

	targetMap := res.Data.BizData
	if sub, ok := targetMap["challenge"].(map[string]interface{}); ok {
		targetMap = sub
	}

	chal := PowChallenge{
		Algorithm:  getString(targetMap, "algorithm", "DeepSeekHashV1"),
		Challenge:  getString(targetMap, "challenge", ""),
		Salt:       getString(targetMap, "salt", ""),
		Difficulty: getInt64(targetMap, "difficulty", 144000),
		ExpireAt:   getInt64(targetMap, "expire_at", 0),
		Signature:  getString(targetMap, "signature", ""),
		TargetPath: getString(targetMap, "target_path", targetPath),
	}

	if chal.Algorithm == "" {
		chal.Algorithm = "DeepSeekHashV1"
	}
	if chal.TargetPath == "" {
		chal.TargetPath = targetPath
	}

	return SolvePoW(chal)
}

func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
}

func getInt64(m map[string]interface{}, key string, def int64) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	if v, ok := m[key].(int64); ok {
		return v
	}
	return def
}

// ChatStreamRequest holds options for ChatStream.
type ChatStreamRequest struct {
	Prompt          string
	Model           string
	SessionID       string
	ThinkingEnabled *bool
	SearchEnabled   *bool
}

// ChatStream initiates a streaming request to DeepSeek and yields chunks via channel.
func (c *Client) ChatStream(ctx context.Context, req ChatStreamRequest) (<-chan ChatChunk, <-chan error) {
	chunkCh := make(chan ChatChunk, 32)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		err := c.executeChatStream(ctx, req, chunkCh, false)
		if err != nil {
			errCh <- err
		}
	}()

	return chunkCh, errCh
}

func (c *Client) executeChatStream(ctx context.Context, req ChatStreamRequest, chunkCh chan<- ChatChunk, isRetry bool) error {
	sessionID := req.SessionID
	if sessionID == "" {
		c.mu.Lock()
		sessionID = c.sessionID
		c.mu.Unlock()
	}

	if sessionID == "" {
		var err error
		sessionID, err = c.CreateSession(ctx)
		if err != nil {
			return fmt.Errorf("sessiya yaratishda xatolik: %w", err)
		}
	}

	// 1. Solve PoW Challenge
	powHeader, err := c.RequestAndSolvePoW(ctx, "/api/v0/chat/completion")
	if err != nil {
		return fmt.Errorf("PoW hal qilishda xatolik: %w", err)
	}

	// 2. Resolve Model Configuration
	modelCfg := ResolveModelConfig(req.Model)
	thinkingEnabled := modelCfg.ThinkingEnabled
	if req.ThinkingEnabled != nil {
		thinkingEnabled = *req.ThinkingEnabled
	}
	searchEnabled := modelCfg.SearchEnabled
	if req.SearchEnabled != nil {
		searchEnabled = *req.SearchEnabled
	}

	c.mu.Lock()
	parentID := c.parentMsgID
	c.mu.Unlock()

	payload := map[string]interface{}{
		"chat_session_id":   sessionID,
		"parent_message_id": parentID,
		"model_type":        modelCfg.ModelType,
		"prompt":            req.Prompt,
		"ref_file_ids":      []string{},
		"thinking_enabled":  thinkingEnabled,
		"search_enabled":    searchEnabled,
		"action":            nil,
		"preempt":           false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/api/v0/chat/completion", c.baseURL)
	headers := c.prepareHeaders(powHeader)
	headers.Set("Accept", "text/event-stream")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	httpReq.Header = headers

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("chat completion ulanish xatosi: %w", err)
	}
	defer resp.Body.Close()

	// If rate-limited (429) or unauthorized (401, 403), cooldown account and failover
	if (resp.StatusCode == 429 || resp.StatusCode == 401 || resp.StatusCode == 403) && c.pool != nil {
		c.mu.Lock()
		curAcc := c.currentAcc
		c.mu.Unlock()
		if curAcc != nil {
			c.pool.MarkFailure(curAcc, resp.StatusCode, resp.Header.Get("Retry-After"))
		}
		if !isRetry {
			c.ResetSession()
			c.mu.Lock()
			c.currentAcc = c.pool.NextAvailable()
			c.mu.Unlock()
			req.SessionID = ""
			return c.executeChatStream(ctx, req, chunkCh, true)
		}
	}

	// If session expired (400, 404, 500) and not retry, create fresh session and retry
	if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 500 {
		if !isRetry {
			c.ResetSession()
			req.SessionID = ""
			return c.executeChatStream(ctx, req, chunkCh, true)
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DeepSeek xatoligi (HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 300)]))
	}

	// Read SSE lines
	scanner := bufio.NewScanner(resp.Body)
	// Support large chunks up to 1MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	parser := NewStreamParser()
	var lastMsgID string

	for scanner.Scan() {
		line := scanner.Text()
		chunk, err := parser.Feed(line)
		if err != nil {
			return err
		}
		if chunk != nil {
			if chunk.MessageID != "" {
				lastMsgID = chunk.MessageID
			}
			chunk.SessionID = sessionID
			chunkCh <- *chunk
			if chunk.FinishReason == "stop" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("stream o'qishda xatolik: %w", err)
	}

	if lastMsgID != "" {
		var idInt int64
		if _, err := fmt.Sscanf(lastMsgID, "%d", &idInt); err == nil {
			c.mu.Lock()
			c.parentMsgID = &idInt
			c.mu.Unlock()
		}
	}

	return nil
}

// Chat executes a synchronous chat completion and aggregates all output.
func (c *Client) Chat(ctx context.Context, prompt string, model string) (*ChatResponse, error) {
	chunkCh, errCh := c.ChatStream(ctx, ChatStreamRequest{
		Prompt: prompt,
		Model:  model,
	})

	var contentBuilder, reasoningBuilder strings.Builder
	var lastSessionID, lastMessageID string

	for {
		select {
		case err, ok := <-errCh:
			if ok && err != nil {
				return nil, err
			}
		case chunk, ok := <-chunkCh:
			if !ok {
				modelStr := model
				if modelStr == "" {
					modelStr = ModelChat
				}
				return &ChatResponse{
					Content:          contentBuilder.String(),
					ReasoningContent: reasoningBuilder.String(),
					SessionID:        lastSessionID,
					MessageID:        lastMessageID,
					Model:            modelStr,
				}, nil
			}
			if chunk.Content != "" {
				contentBuilder.WriteString(chunk.Content)
			}
			if chunk.ReasoningContent != "" {
				reasoningBuilder.WriteString(chunk.ReasoningContent)
			}
			if chunk.SessionID != "" {
				lastSessionID = chunk.SessionID
			}
			if chunk.MessageID != "" {
				lastMessageID = chunk.MessageID
			}
		}
	}
}
