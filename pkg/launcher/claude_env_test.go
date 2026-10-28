package launcher

import "testing"

func TestClaudeProxyEnv(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"HOME=/home/u",
		"ANTHROPIC_API_KEY=sk-ant-real-key",
		"ANTHROPIC_AUTH_TOKEN=sk-ant-real-token",
		"ANTHROPIC_BASE_URL=https://api.anthropic.com",
		"ANTHROPIC_MODEL=claude-sonnet",
		"CLAUDE_CODE_SUBAGENT_MODEL=claude-haiku",
	}
	m := envMap(ClaudeProxyEnv(base, 9655, "deepseek-chat"))

	want := map[string]string{
		"PATH":                 "/usr/bin",
		"HOME":                 "/home/u",
		"ANTHROPIC_BASE_URL":   "http://localhost:9655",
		"ANTHROPIC_AUTH_TOKEN": "nexus",
		"ANTHROPIC_MODEL":      "deepseek-chat",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}

	// Setting both makes Claude Code warn about conflicting auth and disables
	// claude.ai connectors, so only the bearer token may survive.
	if _, ok := m["ANTHROPIC_API_KEY"]; ok {
		t.Error("ANTHROPIC_API_KEY must not be set alongside ANTHROPIC_AUTH_TOKEN")
	}
	// Inherited model overrides would otherwise outlive the proxy's own model.
	if _, ok := m["CLAUDE_CODE_SUBAGENT_MODEL"]; ok {
		t.Error("inherited CLAUDE_CODE_SUBAGENT_MODEL must be dropped")
	}
}
