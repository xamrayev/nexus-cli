package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// LaunchHarness runs DeepSeek evaluation benchmarks or a custom test harness script.
func LaunchHarness(ctx context.Context, opts Options) error {
	// 1. If user passed a custom external command/script to run
	if len(opts.ExtraArgs) > 0 && (opts.ExtraArgs[0] == "--exec" || opts.ExtraArgs[0] == "-e") {
		if len(opts.ExtraArgs) < 2 {
			return fmt.Errorf("--exec dan so'ng buyruq ko'rsatilishi kerak (masalan: nexus launch harness --exec 'pytest tests/')")
		}
		return runExternalHarness(ctx, opts)
	}

	// 2. Otherwise run built-in DeepSeek Eval Harness Suite
	fmt.Printf("\n%s%s===========================================================%s\n", cli.BrightCyan, cli.Bold, cli.Reset)
	fmt.Printf("%s%s            DEEPSEEK AGENTIC EVAL & BENCHMARK HARNESS      %s\n", cli.BrightCyan, cli.Bold, cli.Reset)
	fmt.Printf("%s%s===========================================================%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)

	client := deepseek.NewClient(deepseek.ClientOptions{})

	// Test 1: PoW Challenge & Solver Speed
	fmt.Printf("%s[1/3] PoW Kriptografik Dvigatelini Testlash (Keccak-f[1600])...%s\n", cli.Bold, cli.Reset)
	powStart := time.Now()
	powAns, err := client.RequestAndSolvePoW(ctx, "/api/v0/chat/completion")
	powDuration := time.Since(powStart)
	if err != nil {
		fmt.Printf("  ❌ PoW xatoligi: %v\n", err)
		return err
	}
	fmt.Printf("  ✅ PoW muvaffaqiyatli yechildi! Vaqt: %s (Header uzunligi: %d bayt)\n\n", powDuration.Round(time.Millisecond), len(powAns))

	// Test 2: R1 Deep Thinking & Reasoning Verification
	fmt.Printf("%s[2/3] DeepSeek-R1 Fikrlash (Reasoning / Reflection) Zanjirini Testlash...%s\n", cli.Bold, cli.Reset)
	prompt := "Quyidagi mantiqiy savolga qadamma-qadam tahlil bilan javob ber: Bir qafasda quyonlar va tovuqlar bor. Jami boshlar soni 12 ta, oyoqlar soni esa 34 ta. Qafasda nechtadan quyon va tovuq bor?"

	thinkingEnabled := true
	searchEnabled := false
	streamStart := time.Now()
	var ttft time.Duration
	var hasTTFT bool
	var thoughtTokens int
	var answerTokens int
	var thoughtSample string

	chunkCh, errCh := client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           deepseek.ModelReasoner,
		ThinkingEnabled: &thinkingEnabled,
		SearchEnabled:   &searchEnabled,
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				fmt.Printf("  ❌ Xatolik: %v\n", err)
				return err
			}
		case chunk, ok := <-chunkCh:
			if !ok {
				goto DoneStream
			}
			if !hasTTFT {
				ttft = time.Since(streamStart)
				hasTTFT = true
			}
			if chunk.ReasoningContent != "" {
				thoughtTokens += len(strings.Fields(chunk.ReasoningContent))
				if len(thoughtSample) < 180 {
					thoughtSample += chunk.ReasoningContent
				}
			}
			if chunk.Content != "" {
				answerTokens += len(strings.Fields(chunk.Content))
			}
		}
	}

DoneStream:
	totalTime := time.Since(streamStart)
	fmt.Printf("  ✅ TTFT (Time to First Token): %s\n", ttft.Round(time.Millisecond))
	fmt.Printf("  ✅ Fikrlash so'zlari: ~%d ta so'z\n", thoughtTokens)
	fmt.Printf("  ✅ Javob so'zlari:    ~%d ta so'z\n", answerTokens)
	fmt.Printf("  ✅ Umumiy vaqt:       %s\n", totalTime.Round(time.Millisecond))
	if thoughtSample != "" {
		fmt.Printf("  🧠 R1 Fikrlash parchasidan namuna:\n     %s\"%s...\"%s\n\n", cli.Dim, strings.TrimSpace(thoughtSample), cli.Reset)
	}

	// Test 3: Account Pool Health & Failover Readiness
	fmt.Printf("%s[3/3] Akkauntlar Hovuzi va Failover Tayyorgarligi...%s\n", cli.Bold, cli.Reset)
	pool := deepseek.NewAccountPool(opts.AccountsDir)
	accounts := pool.List()
	readyCount := 0
	for _, a := range accounts {
		if a.IsReady() {
			readyCount++
		}
	}
	fmt.Printf("  ✅ Jami yuklangan akkauntlar: %d ta (Tayyor: %d ta)\n", len(accounts), readyCount)

	fmt.Printf("\n%s%s🎉 HARNESS TESTLARI MUVAFFAQIYATLI YAKUNLANDI! BARCHA SISTEMALAR ISHLAMOQDA.%s\n\n", cli.BrightGreen, cli.Bold, cli.Reset)
	return nil
}

func runExternalHarness(ctx context.Context, opts Options) error {
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

	cmdArgs := opts.ExtraArgs[1:] // skip --exec
	cmdStr := strings.Join(cmdArgs, " ")

	fmt.Printf("%s⚡ Maxsus Harness skripti ishga tushirilmoqda: %s%s\n", cli.BrightCyan, cmdStr, cli.Reset)
	fmt.Printf("  Injected OPENAI_BASE_URL:    http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Injected ANTHROPIC_BASE_URL: http://localhost:%d\n\n", opts.Port)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	// Drop inherited ANTHROPIC_* first: the script talks to the proxy, so a real
	// Anthropic setup would only conflict with the key injected below.
	cmd.Env = append(withoutClaudeEnv(os.Environ()),
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://localhost:%d", opts.Port),
		"OPENAI_API_KEY=nexus",
		"ANTHROPIC_API_KEY=nexus",
		"DEEPSEEK_API_KEY=nexus",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
