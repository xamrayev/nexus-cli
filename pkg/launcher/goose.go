package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
)

// LaunchGoose runs Goose CLI agent wired to Nexus proxy.
func LaunchGoose(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	gooseBin, err := exec.LookPath("goose")
	if err != nil {
		fmt.Printf("\n%s❌ 'goose' (Goose AI Agent) CLI tizimda topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("O'rnatish uchun:")
		fmt.Printf("  %scurl -fsSL https://github.com/block/goose/releases/latest/download/download.sh | bash%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("goose binary not found in PATH")
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

	fmt.Printf("%s⚡ Goose AI Agent ishga tushirilmoqda (DeepSeek R1 / V3 bilan)...%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  OpenAI Base URL: http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Provider:        openai\n\n")

	args := []string{"session"}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, gooseBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"GOOSE_PROVIDER=openai",
		"GOOSE_MODEL=deepseek-reasoner",
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)

	return cmd.Run()
}
