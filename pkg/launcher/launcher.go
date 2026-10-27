package launcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
	"github.com/xamrayev/nexus-cli/pkg/deepseek"
	"github.com/xamrayev/nexus-cli/pkg/server"
)

// Options holds common launcher options.
type Options struct {
	Port        int
	Model       string
	AccountsDir string
	ExtraArgs   []string
}

// EnsureServer verifies if Nexus proxy server is running, or starts an embedded one.
// Returns the server instance, whether it was started by the launcher, and any error.
func EnsureServer(ctx context.Context, port int, accountsDir string) (*server.Server, bool, error) {
	if port <= 0 {
		port = 9655
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := http.Client{Timeout: 350 * time.Millisecond}
	resp, err := client.Get(healthURL)
	if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		return nil, false, nil // Server already running
	}

	fmt.Printf("%s[Nexus]%s Mahalliy DeepSeek API serveri fon rejimida ishga tushirilmoqda (Port: %d)...\n", cli.BrightCyan, cli.Reset, port)

	srv := server.NewServer(server.ServerOptions{
		Host:        "127.0.0.1",
		Port:        port,
		AccountsDir: accountsDir,
	})

	serverErrCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()

	// Wait for server to become healthy
	start := time.Now()
	for time.Since(start) < 3*time.Second {
		select {
		case err := <-serverErrCh:
			return nil, false, fmt.Errorf("server ishga tushmadi: %w", err)
		default:
		}

		resp, err := client.Get(healthURL)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			fmt.Printf("%s[Nexus]%s ✅ Mahalliy proxy faol: http://localhost:%d\n\n", cli.BrightGreen, cli.Reset, port)
			return srv, true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return srv, true, nil
}

// LaunchClaude runs Claude Code CLI wired to Nexus proxy.
func LaunchClaude(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	modelName := opts.Model
	if modelName == "" {
		modelName = deepseek.ModelReasoner
	}
	modelCfg := deepseek.ResolveModelConfig(modelName)
	modelName = modelCfg.ID

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Printf("\n%s❌ 'claude' (Claude Code) CLI tizimda topilmadi.%s\n", cli.Red, cli.Reset)
		fmt.Println("O'rnatish uchun quyidagi buyruqni bajaring:")
		fmt.Printf("  %snpm install -g @anthropic-ai/claude-code%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("claude binary not found in PATH")
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

	var args []string
	hasModelArg := false
	hasNameArg := false

	for _, a := range opts.ExtraArgs {
		if a == "--model" || a == "-m" {
			hasModelArg = true
		}
		if a == "--name" || a == "-n" {
			hasNameArg = true
		}
		args = append(args, a)
	}

	if !hasModelArg {
		args = append([]string{"--model", modelName}, args...)
	}
	if !hasNameArg {
		args = append([]string{"-n", fmt.Sprintf("DeepSeek (%s)", modelName)}, args...)
	}

	fmt.Printf("%s⚡ Claude Code ishga tushirilmoqda (%s)...%s\n", cli.BrightGreen, modelCfg.RealModel, cli.Reset)
	fmt.Printf("  API Base URL: http://localhost:%d\n", opts.Port)
	fmt.Printf("  Faol Model:   %s\n", modelName)
	fmt.Printf("  Modelni almashtirish: Claude ichida /model <nomi> yoki nexus launch claude -m <nomi>\n\n")

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://localhost:%d", opts.Port),
		"ANTHROPIC_AUTH_TOKEN=nexus",
		"ANTHROPIC_API_KEY=nexus",
		fmt.Sprintf("ANTHROPIC_MODEL=%s", modelName),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// LaunchCodex runs OpenAI Codex / OpenCode / Aider CLI wired to Nexus proxy.
func LaunchCodex(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	// Check codex, opencode, or aider
	var binPath, toolName string
	candidates := []string{"codex", "opencode", "aider"}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			binPath = p
			toolName = c
			break
		}
	}

	if binPath == "" {
		fmt.Printf("\n%s❌ Codex / OpenCode / Aider CLI topilmadi.%s\n", cli.Yellow, cli.Reset)
		fmt.Println("O'rnatish uchun tavsiya etilgan variantlar:")
		fmt.Printf("  1. Aider:    %spip install aider-chat%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("  2. OpenCode: %snpm install -g opencode-ai%s\n\n", cli.BrightCyan, cli.Reset)
		fmt.Println("Yoki muhit o'zgaruvchilarini qo'lda eksport qiling:")
		fmt.Printf("  export OPENAI_BASE_URL=\"http://localhost:%d/v1\"\n", opts.Port)
		fmt.Println("  export OPENAI_API_KEY=\"nexus\"")
		return fmt.Errorf("no codex/opencode binary found")
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

	fmt.Printf("%s⚡ %s ishga tushirilmoqda (DeepSeek OpenAI API quvvatida)...%s\n", cli.BrightGreen, toolName, cli.Reset)
	fmt.Printf("  OpenAI Base URL: http://localhost:%d/v1\n\n", opts.Port)

	cmd := exec.CommandContext(ctx, binPath, opts.ExtraArgs...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// LaunchChatGPT opens ChatGPT Desktop app and prints bridge connection guide.
func LaunchChatGPT(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	srv, started, err := EnsureServer(ctx, opts.Port, opts.AccountsDir)
	if err != nil {
		return err
	}
	if started && srv != nil {
		// Keep server alive while user interacts
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	fmt.Printf("%s%s=== CHATGPT DESKTOP & LOCAL BRIDGE ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)

	if runtime.GOOS == "darwin" {
		appPath := "/Applications/ChatGPT.app"
		if _, err := os.Stat(appPath); err == nil {
			fmt.Printf("✅ ChatGPT Desktop ilovasi ochilmoqda...\n")
			_ = exec.Command("open", "-a", appPath).Run()
		} else {
			fmt.Println("ℹ️ ChatGPT Desktop ilovasi /Applications ichida topilmadi.")
		}
	}

	fmt.Println("\nMahalliy DeepSeek bridge parametrlari:")
	fmt.Printf("  • OpenAI API URL:  %shttp://localhost:%d/v1%s\n", cli.BrightGreen, opts.Port, cli.Reset)
	fmt.Printf("  • API Key:         %snexus%s (yoki istalgan kalit)\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  • Tavsiya model:   %sdeepseek-reasoner%s (R1) yoki %sdeepseek-chat%s (V3)\n", cli.Bold, cli.Reset, cli.Bold, cli.Reset)
	fmt.Printf("\nChatGPT yoki uchinchi tomon dasturlari sozlamalarida Base URL sifatida kiriting.\n")
	fmt.Printf("Chiqish uchun Ctrl+C bosing...\n")

	<-ctx.Done()
	return nil
}

// PrintLaunchMenu outputs the launcher help menu.
func PrintLaunchMenu(w io.Writer) {
	fmt.Fprintf(w, `
%s%sNexus Universal Launcher (Integratsiyalar)%s

Foydalanish: nexus launch <platforma> [qo'shimcha bayroqlar]

Mavjud platformalar:
  %sclaude%s          Claude Code CLI (Anthropic Messages API /v1/messages)
  %sdsh%s             Official DeepSeek Harness (@deepseek-ai/dsh CLI)
  %scodex%s           OpenAI Codex CLI (OpenAI Responses wire API)
  %sopencode%s        OpenCode CLI (avtomatik config bilan)
  %sgoose%s           Block Goose AI Agent (OpenAI API provayderi)
  %saider%s           Aider Pair Programming CLI (DeepSeek R1 bilan)
  %schatgpt%s         ChatGPT Desktop ilovasi va mahalliy OpenAI bridge
  %sharness%s         Nexus Ichki Agentic Eval & Benchmark Harness
  %spool%s            Ko'p akkauntlar hovuzi monitori

Misollar:
  nexus launch claude
  nexus launch claude -p "Ushbu loyihani tahlil qil"
  nexus launch dsh
  nexus launch codex
  nexus launch opencode
  nexus launch goose
  nexus launch aider
  nexus launch chatgpt
  nexus launch pool
`, cli.BrightCyan, cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset)
}
