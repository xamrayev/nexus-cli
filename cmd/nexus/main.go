package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/xamrayev/nexus-cli/pkg/cli"
	"github.com/xamrayev/nexus-cli/pkg/deepseek"
	"github.com/xamrayev/nexus-cli/pkg/launcher"
	"github.com/xamrayev/nexus-cli/pkg/server"
)

// Build metadata. Set at build time via ldflags, e.g.:
//
//	go build -ldflags "-X main.version=v0.1.0 -X main.commit=abc1234 -X main.date=2026-10-27T00:00:00Z" ./cmd/nexus
//
// GoReleaser fills these automatically from the git tag.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// versionString returns the version line printed by `nexus version`.
// When ldflags were not set (e.g. `go install ...@v0.1.0`), it falls back
// to the module version and VCS info recorded by the Go toolchain.
func versionString() string {
	v, c, d := version, commit, date
	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				if c == "" && len(kv.Value) >= 7 {
					c = kv.Value[:7]
				}
			case "vcs.time":
				if d == "" {
					d = kv.Value
				}
			}
		}
	}
	out := fmt.Sprintf("Nexus-CLI versiyasi: %s (DeepSeek Web Bridge for Go)", v)
	if c != "" {
		out += "\n  commit: " + c
	}
	if d != "" {
		out += "\n  sana:   " + d
	}
	return out
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 2 {
		// If input is piped, run single prompt
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			client := deepseek.NewClient(deepseek.ClientOptions{})
			if err := cli.RunSinglePrompt(ctx, client, "", deepseek.ModelReasoner, false, false, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Otherwise start interactive REPL
		client := deepseek.NewClient(deepseek.ClientOptions{})
		repl := cli.NewREPL(client, deepseek.ModelReasoner)
		if err := repl.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]

	switch command {
	case "chat":
		handleChat(ctx, os.Args[2:])

	case "serve", "server", "proxy":
		handleServe(ctx, os.Args[2:])

	case "launch":
		handleLaunch(ctx, os.Args[2:])

	case "auth":
		handleAuth(ctx, os.Args[2:])

	case "doctor":
		client := deepseek.NewClient(deepseek.ClientOptions{})
		_ = cli.RunDoctor(ctx, client)

	case "models":
		handleModels()

	case "version", "-v", "--version":
		fmt.Println(versionString())

	case "help", "-h", "--help":
		printHelp()

	default:
		// Check if user passed flags or direct prompt text
		handleRun(ctx, os.Args[1:])
	}
}

func printHelp() {
	cli.PrintBanner(os.Stdout)
	fmt.Print(`Foydalanish:
  nexus [so'rov]                       Oddiy so'rov yuborish yoki chat boshlash
  nexus chat [bayroqlar]               Interaktiv suhbat (REPL) rejimini ishga tushirish
  nexus serve [bayroqlar]              OpenAI / Anthropic mos keluvchi mahalliy API serverni ishga tushirish
  nexus launch <platforma>             Claude Code, Codex, ChatGPT yoki Harness-ni avtomatik ulash
  nexus doctor                         Ulanish va PoW sozlamalarini to'liq tekshirish (diagnostika)
  nexus models                         Mavjud DeepSeek modellari va imkoniyatlarini ko'rish
  nexus auth <buyruq>                  Autentifikatsiya va ko'p akkauntlar hovuzini boshqarish

Launch Integratsiyalari:
  nexus launch claude                  Claude Code CLI (DeepSeek R1 / V3 bilan)
  nexus launch claude --api            Claude Code + DeepSeek rasmiy API ($DEEPSEEK_API_KEY)
  nexus launch dsh                     Official DeepSeek Harness (@deepseek-ai/dsh CLI)
  nexus launch codex                   OpenAI Codex CLI (OpenAI Responses wire API)
  nexus launch opencode                OpenCode CLI (avtomatik config bilan)
  nexus launch goose                   Block Goose AI Agent (OpenAI API provayderi)
  nexus launch aider                   Aider Pair Programming CLI (DeepSeek R1 bilan)
  nexus launch chatgpt                 ChatGPT Desktop ilovasi va mahalliy bridge
  nexus launch harness                 Nexus Ichki Agentic Eval & Benchmark Harness
  nexus launch pool                    Ko'p akkauntlar hovuzi monitori

Auth Buyruqlari:
  nexus auth list                      Mavjud barcha akkauntlar va ularning holati
  nexus auth add <nomi> [--chrome]     Yangi akkaunt qo'shish (Token orqali yoki avtomatik Chrome)
  nexus auth switch <nomi>             Birlamchi faol akkauntni almashtirish
  nexus auth test                      Hovuzdagi barcha akkauntlarni tekshirish
  nexus auth remove <nomi>             Akkauntni hovuzdan o'chirish
  nexus auth login                     Birlamchi tokenni qo'lda kiritish
  nexus auth chrome                    Chrome orqali avtomatik login qilish
  nexus auth whoami                    Joriy hisob ma'lumotlarini ko'rish

Bayroqlar (Flags):
  -m, --model <nom>       Modelni belgilash (default: deepseek-reasoner / R1)
  -t, --thinking          Fikrlash jarayonini (R1 reasoning) yoqish
  -s, --search            Veb-qidiruvni yoqish
  --format <json>         Strukturalangan javob formati (e.g. json)
  --token <token>         DeepSeek tokenni to'g'ridan-to'g'ri ko'rsatish
  --accounts-dir <yo'l>   Akkauntlar papkasi (default: ./accounts)

Misollar:
  nexus "Salom, dunyo!"
  nexus launch claude
  nexus launch claude --api -m pro
  nexus launch codex
  nexus launch harness
  nexus auth list
  nexus serve --port 9655
`)
}

func handleChat(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	model := fs.String("model", deepseek.ModelReasoner, "Model nomi (deepseek-reasoner, deepseek-chat)")
	fs.StringVar(model, "m", deepseek.ModelReasoner, "Model nomi (qisqa)")
	token := fs.String("token", "", "Token")
	_ = fs.Parse(args)

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})
	repl := cli.NewREPL(client, *model)
	if err := repl.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
		os.Exit(1)
	}
}

func handleRun(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	model := fs.String("model", deepseek.ModelReasoner, "Model nomi")
	fs.StringVar(model, "m", deepseek.ModelReasoner, "Model nomi (qisqa)")
	thinking := fs.Bool("thinking", true, "Fikrlash jarayonini yoqish")
	fs.BoolVar(thinking, "t", true, "Fikrlash jarayonini yoqish (qisqa)")
	search := fs.Bool("search", false, "Veb-qidiruvni yoqish")
	fs.BoolVar(search, "s", false, "Veb-qidiruvni yoqish (qisqa)")
	format := fs.String("format", "", "Javob formati (masalan: json)")
	token := fs.String("token", "", "DeepSeek token")

	_ = fs.Parse(args)
	prompt := strings.Join(fs.Args(), " ")

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})
	if err := cli.RunSinglePrompt(ctx, client, prompt, *model, *thinking, *search, *format); err != nil {
		fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
		os.Exit(1)
	}
}

func handleServe(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := fs.String("host", "0.0.0.0", "Tinglash manzili")
	port := fs.Int("port", 9655, "Tinglash porti")
	apiKey := fs.String("key", os.Getenv("PROXY_API_KEY"), "Proxy server API kaliti (ixtiyoriy)")
	maxConcurrent := fs.Int("max-concurrent", 10, "Maksimal bir vaqtdagi so'rovlar soni")
	token := fs.String("token", "", "DeepSeek hisob tokeni")
	accountsDir := fs.String("accounts-dir", "", "Akkauntlar papkasi (default: ./accounts)")
	_ = fs.Parse(args)

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})

	srv := server.NewServer(server.ServerOptions{
		Host:          *host,
		Port:          *port,
		ProxyAPIKey:   *apiKey,
		MaxConcurrent: *maxConcurrent,
		AccountsDir:   *accountsDir,
		Client:        client,
	})

	cli.PrintBanner(os.Stdout)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server xatosi: %v\n", err)
		os.Exit(1)
	}
}

func handleAuth(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Printf("\n%s%sNexus Autentifikatsiya va Ko'p Akkauntlar Hovuzi%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
		fmt.Println("Foydalanish: nexus auth <buyruq> [argumentlar]")
		fmt.Println("  nexus auth list                      Mavjud barcha akkauntlar va ularning holati")
		fmt.Println("  nexus auth add <nomi> [--chrome]     Yangi akkaunt qo'shish (Token yoki avtomatik Chrome)")
		fmt.Println("  nexus auth switch <nomi>             Birlamchi faol akkauntni almashtirish")
		fmt.Println("  nexus auth test                      Hovuzdagi barcha akkauntlarni tekshirish")
		fmt.Println("  nexus auth remove <nomi>             Akkauntni hovuzdan o'chirish")
		fmt.Println("  nexus auth login                     Birlamchi akkaunt tokenni qo'lda kiritish")
		fmt.Println("  nexus auth chrome                    Chrome orqali avtomatik login qilish")
		fmt.Println("  nexus auth whoami                    Joriy faol akkaunt profilini ko'rish")
		fmt.Println("  nexus auth logout                    Birlamchi hisob ma'lumotlarini o'chirish")
		return
	}

	sub := args[0]
	auth := deepseek.NewAuthManager("", "")

	switch sub {
	case "list", "ls":
		pool := deepseek.NewAccountPool("")
		accounts := pool.List()
		activeName := pool.GetActiveAccountName()

		fmt.Printf("\n%s%s=== DEEPSEEK AKKAUNTLAR RO'YXATI ===%s\n", cli.BrightCyan, cli.Bold, cli.Reset)
		if len(accounts) == 0 {
			fmt.Println("Hozircha hech qanday akkaunt saqlanmagan.")
			fmt.Printf("Yangi akkaunt qo'shish uchun: %snexus auth add <nomi>%s\n\n", cli.BrightCyan, cli.Reset)
			return
		}

		fmt.Printf("%-18s %-12s %-12s %-32s %s\n", "AKKAUNT", "HOLATI", "COOLDOWN", "EMAIL / PROFIL", "FAYL")
		fmt.Println(strings.Repeat("-", 85))

		for _, acc := range accounts {
			nameTag := acc.Name
			if acc.Name == activeName || (activeName == "default" && acc.Name == "default") {
				nameTag = fmt.Sprintf("%s%s* %s%s", cli.BrightGreen, cli.Bold, acc.Name, cli.Reset)
			} else {
				nameTag = fmt.Sprintf("  %s", acc.Name)
			}

			status := fmt.Sprintf("%s✅ Tayyor%s", cli.Green, cli.Reset)
			cdSec := acc.CooldownRemaining()
			cdStr := "-"
			if cdSec > 0 {
				status = fmt.Sprintf("%s⏳ Kutish%s", cli.Yellow, cli.Reset)
				cdStr = fmt.Sprintf("%ds", cdSec)
			} else if !acc.IsReady() {
				status = fmt.Sprintf("%s❌ Xato%s", cli.Red, cli.Reset)
			}

			userEmail := "Noma'lum"
			if acc.Profile != nil && acc.Profile.Email != "" {
				userEmail = acc.Profile.Email
			} else if acc.Data.Token != "" {
				t := acc.Data.Token
				if len(t) > 20 {
					userEmail = fmt.Sprintf("%s...%s", t[:8], t[len(t)-6:])
				} else {
					userEmail = t
				}
			}

			fmt.Printf("%-28s %-20s %-12s %-32s %s\n", nameTag, status, cdStr, userEmail, filepath.Base(acc.FilePath))
		}
		fmt.Printf("\n* belgisi — joriy faol birlamchi akkauntni bildiradi.\n")
		fmt.Printf("Akkauntni almashtirish: %snexus auth switch <nomi>%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("Yangi akkaunt qo'shish: %snexus auth add <nomi>%s\n\n", cli.BrightCyan, cli.Reset)

	case "add":
		if len(args) < 2 {
			fmt.Println("Foydalanish: nexus auth add <akkaunt_nomi> [--chrome]")
			fmt.Println("Misol:      nexus auth add acc2")
			fmt.Println("            nexus auth add acc3 --chrome")
			return
		}
		name := args[1]
		useChrome := false
		for _, a := range args[2:] {
			if a == "--chrome" || a == "-c" {
				useChrome = true
			}
		}

		pool := deepseek.NewAccountPool("")
		if useChrome {
			fmt.Printf("Chrome orqali '%s' nomli yangi akkaunt qo'shilmoqda...\n", name)
			helper := deepseek.NewChromeAuthHelper(9334, "", "")
			waitCtx, cancelWait := context.WithCancel(ctx)
			defer cancelWait()
			data, err := helper.LaunchAndExtract(waitCtx, func(msg string) {
				fmt.Println("[Chrome]", msg)
			})
			if err != nil || data == nil || data.Token == "" {
				fmt.Printf("❌ Chrome orqali olishda xatolik: %v\n", err)
				return
			}
			if err := pool.AddAccount(name, *data); err != nil {
				fmt.Printf("❌ Akkauntni saqlashda xatolik: %v\n", err)
				return
			}
			fmt.Printf("✅ '%s' akkaunti muvaffaqiyatli saqlandi!\n", name)
		} else {
			fmt.Printf("'%s' akkaunti uchun DeepSeek userToken (yoki Bearer token)ni kiriting:\n> ", name)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				tokenInput := strings.TrimSpace(scanner.Text())
				if tokenInput == "" {
					fmt.Println("❌ Token kiritilmadi.")
					return
				}
				cleanTok := deepseek.CleanToken(tokenInput)
				fmt.Print("DeepSeek Cookie-larini kiriting (ixtiyoriy, agar bo'sh bo'lsa ENTER):\n> ")
				var cookieInput string
				if scanner.Scan() {
					cookieInput = strings.TrimSpace(scanner.Text())
				}
				data := deepseek.AuthData{
					Token:  cleanTok,
					Cookie: cookieInput,
				}
				if err := pool.AddAccount(name, data); err != nil {
					fmt.Printf("❌ Saqlashda xatolik: %v\n", err)
					return
				}
				fmt.Printf("✅ '%s' akkaunti saqlandi!\n", name)
			}
		}

	case "switch", "use":
		if len(args) < 2 {
			fmt.Println("Foydalanish: nexus auth switch <akkaunt_nomi>")
			return
		}
		name := args[1]
		pool := deepseek.NewAccountPool("")
		acc, err := pool.SwitchActive(name)
		if err != nil {
			fmt.Printf("❌ Xatolik: %v\n", err)
			return
		}
		fmt.Printf("✅ Faol akkaunt almashtirildi: %s (%s)\n", acc.Name, filepath.Base(acc.FilePath))

	case "test", "validate":
		pool := deepseek.NewAccountPool("")
		fmt.Println("Hovuzdagi akkauntlar tekshirilmoqda...")
		pool.ValidateAll(ctx)
		for _, acc := range pool.List() {
			if acc.Profile != nil {
				name := acc.Profile.Nickname
				if name == "" {
					name = acc.Profile.Email
				}
				fmt.Printf("✅ %-14s: Muvaffaqiyatli bog'langan (%s, %s)\n", acc.Name, name, acc.Profile.Email)
			} else {
				fmt.Printf("❌ %-14s: Token yaroqsiz yoki tarmoq xatosi\n", acc.Name)
			}
		}

	case "login", "set":
		fmt.Println("===========================================================")
		fmt.Println("       DEEPSEEK AKKAUNTINI BOG'LASH BO'YICHA YO'RIQNOMA    ")
		fmt.Println("===========================================================")
		fmt.Println("1. Brauzerda https://chat.deepseek.com saytiga kiring.")
		fmt.Println("2. Klaviaturada F12 (yoki Cmd+Option+I) bosing.")
		fmt.Println("3. Application (yoki Storage) bo'limida Local Storage -> chat.deepseek.com")
		fmt.Println("   'userToken' qiymatini nusxalang.")
		fmt.Println("===========================================================")
		fmt.Print("\nDeepSeek userToken (yoki Bearer token)ni kiriting: ")

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			tokenInput := strings.TrimSpace(scanner.Text())
			if tokenInput == "" {
				fmt.Println("❌ Token kiritilmadi.")
				return
			}
			if err := auth.SetToken(tokenInput, true); err != nil {
				fmt.Printf("❌ Faylga saqlashda xatolik: %v\n", err)
				return
			}

			fmt.Println("Tekshirilmoqda...")
			profile, err := auth.ValidateToken(ctx)
			if err != nil {
				fmt.Printf("⚠️ Ogohlantirish: Token saqlandi, lekin server tekshiruvida xatolik: %v\n", err)
				return
			}

			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			fmt.Printf("✅ Muvaffaqiyatli bog'landi! Xush kelibsiz, %s (ID: %s)!\n", name, profile.ID)
		}

	case "chrome":
		fmt.Println("===========================================================")
		fmt.Println("       DEEPSEEK CHROME AVTOMATIK AUTENTIFIKATSIYA         ")
		fmt.Println("===========================================================")
		fmt.Println("1. Yangi toza Chrome oynasi ochilmoqda...")
		fmt.Println("2. Ochilgan Chrome oynasida chat.deepseek.com ga kiring (Login).")
		fmt.Println("3. Tizimga muvaffaqiyatli kirganingizdan so'ng, DeepSeek")
		fmt.Println("   interfeysida biror qisqa xabar (masalan 'ok') yuboring.")
		fmt.Println("4. Token avtomatik olinadi (yoki bu yerda ENTER tugmasini bosing).")
		fmt.Println("===========================================================")

		helper := deepseek.NewChromeAuthHelper(9334, "", "")

		enterCh := make(chan struct{})
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				close(enterCh)
			}
		}()

		var data *deepseek.AuthData
		var err error

		waitCtx, cancelWait := context.WithCancel(ctx)
		defer cancelWait()

		go func() {
			select {
			case <-enterCh:
				fmt.Println("\n[Chrome] Token tekshirilmoqda...")
				d, e := helper.TryExtractNow(waitCtx)
				if e == nil && d != nil && d.Token != "" {
					data = d
					cancelWait()
				}
			case <-waitCtx.Done():
			}
		}()

		d, e := helper.LaunchAndExtract(waitCtx, func(msg string) {
			fmt.Println("[Chrome]", msg)
		})
		if data == nil {
			data = d
			err = e
		}

		if err != nil && (data == nil || data.Token == "") {
			fmt.Printf("❌ Chrome orqali autentifikatsiya qilib bo'lmadi: %v\n", err)
			return
		}

		if err := auth.SetAuthData(*data, true); err != nil {
			fmt.Printf("❌ Saqlashda xatolik: %v\n", err)
			return
		}

		fmt.Println("✅ Token va cookies muvaffaqiyatli saqlandi!")
		profile, err := auth.ValidateToken(ctx)
		if err == nil && profile != nil {
			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			fmt.Printf("🎉 Muvaffaqiyatli bog'landi! Xush kelibsiz, %s (ID: %s)\n", name, profile.ID)
		}

	case "whoami", "status":
		token := auth.GetToken()
		if token == "" {
			fmt.Println("❌ Token topilmadi. 'nexus auth login' orqali tokenni kiriting.")
			return
		}
		profile, err := auth.ValidateToken(ctx)
		if err != nil {
			fmt.Printf("❌ Token yaroqsiz yoki muddati o'tgan: %v\n", err)
			return
		}
		fmt.Printf("✅ Bog'langan hisob:\n  ID:       %s\n  Nickname: %s\n  Email:    %s\n  Phone:    %s\n",
			profile.ID, profile.Nickname, profile.Email, profile.Phone)

	case "logout", "remove", "rm":
		if len(args) >= 2 {
			name := args[1]
			pool := deepseek.NewAccountPool("")
			if err := pool.RemoveAccount(name); err != nil {
				fmt.Printf("❌ O'chirishda xatolik: %v\n", err)
				return
			}
			fmt.Printf("✅ '%s' akkaunti o'chirildi.\n", name)
			return
		}
		_ = auth.Remove()
		fmt.Println("✅ Hisob ma'lumotlari o'chirildi.")

	default:
		fmt.Printf("Noma'lum auth buyrug'i: %s. 'nexus auth' deb yozib yordam oling.\n", sub)
	}
}

func handleModels() {
	fmt.Printf("%s%s=== Qo'llab-quvvatlanadigan DeepSeek Modellari ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
	for id, cfg := range deepseek.ModelConfigs {
		reasoning := "yo'q"
		if cfg.Capabilities.Reasoning {
			reasoning = "ha (R1)"
		}
		search := "yo'q"
		if cfg.Capabilities.WebSearch {
			search = "ha"
		}
		fmt.Printf("  • %s%-24s%s -> %s\n", cli.Bold, id, cli.Reset, cfg.RealModel)
		fmt.Printf("    Fikrlash: %s | Veb-qidiruv: %s\n\n", reasoning, search)
	}
}

func handleLaunch(ctx context.Context, args []string) {
	if len(args) == 0 {
		launcher.PrintLaunchMenu(os.Stdout)
		return
	}

	target := strings.ToLower(args[0])
	rawExtra := args[1:]

	model := ""
	useAPI := false
	apiKey := ""
	var extra []string
	for i := 0; i < len(rawExtra); i++ {
		arg := rawExtra[i]
		if arg == "--api" || arg == "--official" {
			useAPI = true
			continue
		} else if arg == "--api-key" {
			useAPI = true
			if i+1 < len(rawExtra) {
				apiKey = rawExtra[i+1]
				i++
			}
			continue
		} else if strings.HasPrefix(arg, "--api-key=") {
			apiKey = strings.TrimPrefix(arg, "--api-key=")
			useAPI = true
			continue
		} else if arg == "-m" || arg == "--model" {
			if i+1 < len(rawExtra) {
				model = rawExtra[i+1]
				i++
				continue
			}
		} else if strings.HasPrefix(arg, "--model=") {
			model = strings.TrimPrefix(arg, "--model=")
			continue
		} else if i == 0 && (arg == "chat" || arg == "v3" || arg == "r1" || arg == "reasoner" || strings.HasPrefix(arg, "deepseek-")) {
			model = arg
			continue
		}
		extra = append(extra, arg)
	}

	opts := launcher.Options{
		Port:      9655,
		Model:     model,
		ExtraArgs: extra,
		UseAPI:    useAPI,
		APIKey:    apiKey,
	}

	isClaude := target == "claude" || target == "claude-code"
	if useAPI && !isClaude {
		fmt.Fprintf(os.Stderr, "Xatolik: --api rejimi hozircha faqat 'nexus launch claude' uchun.\n")
		os.Exit(1)
	}

	switch target {
	case "claude", "claude-code":
		launch := launcher.LaunchClaude
		if useAPI {
			launch = launcher.LaunchClaudeAPI
		}
		if err := launch(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "dsh", "deepseek-harness":
		if err := launcher.LaunchDSH(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "codex":
		if err := launcher.LaunchCodexDirect(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "opencode":
		if err := launcher.LaunchOpenCode(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "goose":
		if err := launcher.LaunchGoose(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "aider":
		if err := launcher.LaunchAider(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "pool":
		if err := launcher.LaunchPool(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "chatgpt", "chatgpt-desktop":
		if err := launcher.LaunchChatGPT(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	case "harness", "bench", "test-harness":
		if err := launcher.LaunchHarness(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Xatolik: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Noma'lum launcher platformasi: '%s'.\n", target)
		launcher.PrintLaunchMenu(os.Stdout)
	}
}
