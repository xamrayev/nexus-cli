package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
)

// LaunchCodexDirect runs OpenAI Codex CLI wired to Nexus proxy using profile config or CLI overrides.
func LaunchCodexDirect(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	codexBin, err := exec.LookPath("codex")
	if err != nil {
		fmt.Printf("\n%s❌ 'codex' (OpenAI Codex CLI) tizimda topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("O'rnatish uchun quyidagilarni ko'rib chiqing:")
		fmt.Printf("  %snpm install -g @openai/codex%s yoki rasmiy Codex CLI\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("codex binary not found in PATH")
	}

	// Ensure ~/.codex/config.toml has nexus profile
	home, err := os.UserHomeDir()
	if err == nil {
		codexDir := filepath.Join(home, ".codex")
		_ = os.MkdirAll(codexDir, 0o755)
		configPath := filepath.Join(codexDir, "config.toml")

		profileSnippet := fmt.Sprintf(`
# Nexus CLI DeepSeek Profile
[model_providers.nexus]
name = "Nexus"
base_url = "http://localhost:%d/v1"
wire_api = "responses"
`, opts.Port)

		existing, _ := os.ReadFile(configPath)
		if !strings.Contains(string(existing), "[model_providers.nexus]") {
			_ = os.WriteFile(configPath, append(existing, []byte(profileSnippet)...), 0o644)
		}
	}

	srv, started, err := EnsureServer(ctx, opts.Port, opts.AccountsDir)
	if err != nil {
		return err
	}
	if started && srv != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	fmt.Printf("%s⚡ OpenAI Codex CLI ishga tushirilmoqda (DeepSeek R1 / V3 bilan)...%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  Nexus API URL: http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Profile:       nexus\n\n")

	// Pass config overrides matching Ollama codex integration
	codexArgs := []string{
		"-c", "model_provider=\"nexus\"",
		"-c", "model_providers.nexus.name=\"Nexus\"",
		"-c", fmt.Sprintf("model_providers.nexus.base_url=\"http://localhost:%d/v1\"", opts.Port),
		"-c", "model_providers.nexus.wire_api=\"responses\"",
		"-m", "deepseek-reasoner",
	}
	codexArgs = append(codexArgs, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, codexBin, codexArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=nexus",
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
	)

	return cmd.Run()
}
