package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xamrayev/nexus-cli/pkg/cli"
)

// DeepSeek official Anthropic-compatible API.
// Docs: https://api-docs.deepseek.com/quick_start/agent_integrations/claude_code
const (
	DeepSeekAnthropicBaseURL = "https://api.deepseek.com/anthropic"
	DeepSeekAPIKeysURL       = "https://platform.deepseek.com/api_keys"

	deepSeekFlash1M = "deepseek-flash[1m]"
	deepSeekFlash   = "deepseek-flash"
	deepSeekPro     = "deepseek-v4-pro"
)

// ErrNoAPIKey is returned when official-API mode is requested without a key.
var ErrNoAPIKey = errors.New("DeepSeek API kaliti topilmadi")

// ResolveAPIModel maps short names used with `nexus launch claude --api -m <name>`
// to DeepSeek official API model IDs. Unknown names are passed through unchanged.
func ResolveAPIModel(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "flash", "deepseek-flash", "chat", "v3":
		return deepSeekFlash1M
	case "pro", "v4-pro", "v4", "deepseek-v4-pro", "reasoner", "r1":
		return deepSeekPro
	default:
		return name
	}
}

// ClaudeAPIEnv builds the environment for Claude Code talking directly to the
// DeepSeek official API. It drops inherited ANTHROPIC_* / Claude model variables
// (so a real Anthropic key or an earlier Nexus setup cannot leak in or conflict)
// and sets the variables recommended by DeepSeek's Claude Code guide.
func ClaudeAPIEnv(base []string, apiKey, model string) []string {
	drop := map[string]bool{
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

	env := make([]string, 0, len(base)+10)
	hasCompactWindow := false
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if k == "CLAUDE_CODE_AUTO_COMPACT_WINDOW" {
			hasCompactWindow = true
		}
		if !drop[k] {
			env = append(env, kv)
		}
	}

	main := ResolveAPIModel(model)
	// DeepSeek's guide pairs the 1M-context flash model with this compaction
	// threshold. Keep the user's own value if they set one.
	if strings.HasSuffix(main, "[1m]") && !hasCompactWindow {
		env = append(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW=786432")
	}
	return append(env,
		"ANTHROPIC_BASE_URL="+DeepSeekAnthropicBaseURL,
		"ANTHROPIC_AUTH_TOKEN="+apiKey,
		"ANTHROPIC_MODEL="+main,
		"ANTHROPIC_DEFAULT_OPUS_MODEL="+main,
		"ANTHROPIC_DEFAULT_SONNET_MODEL="+main,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL="+deepSeekFlash,
		"CLAUDE_CODE_SUBAGENT_MODEL="+deepSeekFlash,
	)
}

// apiKeyFrom returns the explicit key, or DEEPSEEK_API_KEY from the environment.
func apiKeyFrom(explicit string) string {
	if k := strings.TrimSpace(explicit); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
}

// LaunchClaudeAPI runs Claude Code against the DeepSeek official API
// (api.deepseek.com/anthropic). No local proxy and no web account are used.
func LaunchClaudeAPI(ctx context.Context, opts Options) error {
	key := apiKeyFrom(opts.APIKey)
	if key == "" {
		fmt.Printf("\n%s❌ DeepSeek API kaliti topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("Rasmiy API rejimi uchun kalit kerak (pullik, lekin barqaror):")
		fmt.Printf("  1. Kalit oling: %s%s%s\n", cli.BrightCyan, DeepSeekAPIKeysURL, cli.Reset)
		fmt.Printf("  2. %sexport DEEPSEEK_API_KEY=sk-...%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("  3. %snexus launch claude --api%s\n\n", cli.BrightCyan, cli.Reset)
		return ErrNoAPIKey
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Printf("\n%s❌ 'claude' (Claude Code) CLI tizimda topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("O'rnatish uchun quyidagi buyruqni bajaring:")
		fmt.Printf("  %snpm install -g @anthropic-ai/claude-code%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("claude binary not found in PATH")
	}

	model := ResolveAPIModel(opts.Model)

	args := opts.ExtraArgs
	hasName := false
	for _, a := range args {
		if a == "--name" || a == "-n" {
			hasName = true
		}
	}
	if !hasName {
		args = append([]string{"-n", fmt.Sprintf("DeepSeek API (%s)", model)}, args...)
	}

	fmt.Printf("%s⚡ Claude Code ishga tushirilmoqda — DeepSeek rasmiy API%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  API Base URL: %s\n", DeepSeekAnthropicBaseURL)
	fmt.Printf("  Faol Model:   %s\n", model)
	fmt.Printf("  Modelni almashtirish: nexus launch claude --api -m pro|flash\n\n")

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = ClaudeAPIEnv(os.Environ(), key, opts.Model)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
