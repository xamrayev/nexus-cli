package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer builds a server whose account pool points at an empty temp dir,
// so tests never touch real accounts or the network.
func newTestServer(t *testing.T, apiKey string) http.Handler {
	t.Helper()
	srv := NewServer(ServerOptions{
		Port:        9999,
		ProxyAPIKey: apiKey,
		AccountsDir: t.TempDir(),
	})
	return srv.Handler()
}

func do(h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestRoutes_PublicEndpoints(t *testing.T) {
	h := newTestServer(t, "")

	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/health", http.StatusOK},
		{"GET", "/readyz", http.StatusOK},
		{"GET", "/v1/models", http.StatusOK},
		{"GET", "/v1/model-capabilities", http.StatusOK},
		{"GET", "/v1/sessions", http.StatusOK},
		{"POST", "/v1/sessions/reset", http.StatusOK},
		{"GET", "/api/version", http.StatusOK},
		{"GET", "/api/tags", http.StatusOK},
		{"GET", "/api/ps", http.StatusOK},
		{"GET", "/no-such-route", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			w := do(h, c.method, c.path, nil)
			if w.Code != c.want {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, c.want, w.Body.String())
			}
		})
	}
}

func TestRoutes_HealthPayload(t *testing.T) {
	h := newTestServer(t, "")
	w := do(h, "GET", "/health", nil)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("health is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v, want ok", body["status"])
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestRoutes_ModelsListNotEmpty(t *testing.T) {
	h := newTestServer(t, "")
	w := do(h, "GET", "/v1/models", nil)

	var body struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("models is not JSON: %v", err)
	}
	if body.Object != "list" || len(body.Data) == 0 {
		t.Fatalf("unexpected models payload: %+v", body)
	}
}

func TestRoutes_ProxyAPIKey(t *testing.T) {
	const key = "test-key"
	h := newTestServer(t, key)

	if w := do(h, "GET", "/v1/models", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("no key: status = %d, want 401", w.Code)
	}
	if w := do(h, "GET", "/v1/models", map[string]string{"Authorization": "Bearer wrong"}); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong key: status = %d, want 401", w.Code)
	}
	if w := do(h, "GET", "/v1/models", map[string]string{"Authorization": "Bearer " + key}); w.Code != http.StatusOK {
		t.Errorf("bearer key: status = %d, want 200", w.Code)
	}
	if w := do(h, "GET", "/v1/models", map[string]string{"x-api-key": key}); w.Code != http.StatusOK {
		t.Errorf("x-api-key: status = %d, want 200", w.Code)
	}
	// Health stays public even when a key is configured.
	if w := do(h, "GET", "/health", nil); w.Code != http.StatusOK {
		t.Errorf("health with key set: status = %d, want 200", w.Code)
	}
	// Dispatch endpoints reject a missing key before touching DeepSeek.
	for _, p := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
		if w := do(h, "POST", p, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("POST %s without key: status = %d, want 401", p, w.Code)
		}
	}
}

func TestRoutes_DispatchRejectsGET(t *testing.T) {
	h := newTestServer(t, "")
	for _, p := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
		if w := do(h, "GET", p, nil); w.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET %s: status = %d, want 405", p, w.Code)
		}
	}
}

func TestRoutes_CORS(t *testing.T) {
	h := newTestServer(t, "")

	w := do(h, "OPTIONS", "/v1/chat/completions", map[string]string{"Origin": "http://example.test"})
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://example.test" {
		t.Errorf("Allow-Origin = %q, want echoed origin", got)
	}

	w = do(h, "GET", "/health", nil)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin without Origin header = %q, want *", got)
	}
}
