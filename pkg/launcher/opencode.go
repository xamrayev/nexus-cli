package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
)

// LaunchOpenCode runs OpenCode CLI wired to Nexus proxy using OPENCODE_CONFIG_CONTENT.
func LaunchOpenCode(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	var opencodeBin string
	if p, err := exec.LookPath("opencode"); err == nil {
		opencodeBin = p
	} else {
		home, _ := os.UserHomeDir()
		fallback := filepath.Join(home, ".opencode", "bin", "opencode")
		if _, err := os.Stat(fallback); err == nil {
			opencodeBin = fallback
		}
	}

	if opencodeBin == "" {
		fmt.Printf("\n%s❌ 'opencode' CLI tizimda topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("O'rnatish uchun quyidagi buyruqlardan birini bajaring:")
		fmt.Printf("  %scurl -fsSL https://opencode.ai/install | bash%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("  yoki: %snpm install -g opencode-ai%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("opencode binary not found")
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

	// Prepare inline OpenCode config matching Ollama pattern
	configObj := map[string]interface{}{
		"provider": map[string]interface{}{
			"nexus": map[string]interface{}{
				"name":     "Nexus",
				"base_url": fmt.Sprintf("http://localhost:%d/v1", opts.Port),
				"api_key":  "nexus",
				"models":   []string{"deepseek-reasoner", "deepseek-chat"},
			},
		},
		"default_model": "nexus/deepseek-reasoner",
	}
	configBytes, _ := json.Marshal(configObj)

	fmt.Printf("%s⚡ OpenCode CLI ishga tushirilmoqda (DeepSeek R1 / V3 bilan)...%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  Nexus API URL: http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Default Model: deepseek-reasoner\n\n")

	cmd := exec.CommandContext(ctx, opencodeBin, opts.ExtraArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_CONFIG_CONTENT=%s", string(configBytes)),
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)

	return cmd.Run()
}
