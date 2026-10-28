package launcher

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestResolveAPIModel(t *testing.T) {
	cases := map[string]string{
		"":                "deepseek-flash[1m]",
		"flash":           "deepseek-flash[1m]",
		"Flash":           "deepseek-flash[1m]",
		"chat":            "deepseek-flash[1m]",
		"pro":             "deepseek-v4-pro",
		"r1":              "deepseek-v4-pro",
		"deepseek-v4-pro": "deepseek-v4-pro",
		"some-new-model":  "some-new-model",
	}
	for in, want := range cases {
		if got := ResolveAPIModel(in); got != want {
			t.Errorf("ResolveAPIModel(%q) = %q, want %q", in, got, want)
		}
	}
}

func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		if _, dup := m[k]; dup {
			m[k] = "<DUPLICATE>"
			continue
		}
		m[k] = v
	}
	return m
}

func TestClaudeAPIEnv(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"HOME=/home/u",
		"ANTHROPIC_API_KEY=sk-ant-real-key",
		"ANTHROPIC_BASE_URL=http://localhost:9655",
		"ANTHROPIC_MODEL=claude-sonnet",
	}
	m := envMap(ClaudeAPIEnv(base, "sk-ds-test", "pro"))

	want := map[string]string{
		"PATH":                           "/usr/bin",
		"HOME":                           "/home/u",
		"ANTHROPIC_BASE_URL":             "https://api.deepseek.com/anthropic",
		"ANTHROPIC_AUTH_TOKEN":           "sk-ds-test",
		"ANTHROPIC_MODEL":                "deepseek-v4-pro",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "deepseek-v4-pro",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-pro",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "deepseek-flash",
		"CLAUDE_CODE_SUBAGENT_MODEL":     "deepseek-flash",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}
	// A real Anthropic key from the user's shell must never reach DeepSeek.
	if _, ok := m["ANTHROPIC_API_KEY"]; ok {
		t.Error("ANTHROPIC_API_KEY must be removed from the environment")
	}
	// pro is not a [1m] model → no compaction override.
	if _, ok := m["CLAUDE_CODE_AUTO_COMPACT_WINDOW"]; ok {
		t.Error("compact window must not be set for deepseek-v4-pro")
	}
}

func TestClaudeAPIEnv_FlashCompactWindow(t *testing.T) {
	m := envMap(ClaudeAPIEnv([]string{"PATH=/usr/bin"}, "k", ""))
	if m["ANTHROPIC_MODEL"] != "deepseek-flash[1m]" {
		t.Fatalf("default model = %q", m["ANTHROPIC_MODEL"])
	}
	if m["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "786432" {
		t.Errorf("compact window = %q, want 786432", m["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}

	// The user's own value wins.
	m = envMap(ClaudeAPIEnv([]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW=200000"}, "k", "flash"))
	if m["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "200000" {
		t.Errorf("user compact window overridden: %q", m["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
}

func TestLaunchClaudeAPI_NoKey(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	err := LaunchClaudeAPI(context.Background(), Options{})
	if !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
}

func TestAPIKeyFrom(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", " sk-env ")
	if got := apiKeyFrom(""); got != "sk-env" {
		t.Errorf("env fallback = %q", got)
	}
	if got := apiKeyFrom("sk-flag"); got != "sk-flag" {
		t.Errorf("explicit key = %q", got)
	}
}
