package launcher

import (
	"fmt"
	"strings"
)

// claudeOwnedEnv lists the Claude Code variables Nexus sets itself. They are
// stripped from the inherited environment before launching so a real Anthropic
// key or an earlier Nexus setup cannot leak in or conflict. In particular,
// leaving ANTHROPIC_API_KEY and ANTHROPIC_AUTH_TOKEN both set makes Claude Code
// warn that auth may not work and disables claude.ai connectors.
var claudeOwnedEnv = map[string]bool{
	"ANTHROPIC_BASE_URL":             true,
	"ANTHROPIC_API_KEY":              true,
	"ANTHROPIC_AUTH_TOKEN":           true,
	"ANTHROPIC_MODEL":                true,
	"ANTHROPIC_SMALL_FAST_MODEL":     true,
	"ANTHROPIC_DEFAULT_OPUS_MODEL":   true,
	"ANTHROPIC_DEFAULT_SONNET_MODEL": true,
	"ANTHROPIC_DEFAULT_HAIKU_MODEL":  true,
	"CLAUDE_CODE_SUBAGENT_MODEL":     true,
}

// withoutClaudeEnv returns base with every variable Nexus owns removed.
func withoutClaudeEnv(base []string) []string {
	env := make([]string, 0, len(base)+10)
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if !claudeOwnedEnv[k] {
			env = append(env, kv)
		}
	}
	return env
}

// ClaudeProxyEnv builds the environment for Claude Code talking to the local
// Nexus proxy. Only ANTHROPIC_AUTH_TOKEN is set: the proxy accepts it as a
// bearer token, and setting ANTHROPIC_API_KEY alongside it would make Claude
// Code warn about conflicting auth.
func ClaudeProxyEnv(base []string, port int, model string) []string {
	return append(withoutClaudeEnv(base),
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://localhost:%d", port),
		"ANTHROPIC_AUTH_TOKEN=nexus",
		"ANTHROPIC_MODEL="+model,
	)
}
