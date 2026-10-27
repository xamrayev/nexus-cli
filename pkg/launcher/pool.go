package launcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/cli"
	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// LaunchPool displays and monitors the Nexus multi-account pool.
func LaunchPool(ctx context.Context, opts Options) error {
	pool := deepseek.NewAccountPool(opts.AccountsDir)
	accounts := pool.List()

	fmt.Printf("\n%s%s=== NEXUS MULTI-ACCOUNT POOL MONITOR ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
	if len(accounts) == 0 {
		fmt.Println("Hovuzda faol akkauntlar topilmadi.")
		fmt.Printf("Yangi akkaunt qo'shish uchun: %snexus auth add <nomi>%s\n\n", cli.BrightCyan, cli.Reset)
		return nil
	}

	activeName := pool.GetActiveAccountName()
	fmt.Printf("Jami akkauntlar: %d ta | Birlamchi faol: %s%s%s\n\n", len(accounts), cli.BrightGreen, activeName, cli.Reset)
	fmt.Printf("%-20s %-15s %-12s %-25s\n", "AKKAUNT", "HOLATI", "COOLDOWN", "EMAIL")
	fmt.Println(strings.Repeat("-", 75))

	for _, acc := range accounts {
		nameTag := acc.Name
		if acc.Name == activeName {
			nameTag = fmt.Sprintf("%s* %s%s", cli.BrightGreen, acc.Name, cli.Reset)
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

		email := "Noma'lum"
		if acc.Profile != nil && acc.Profile.Email != "" {
			email = acc.Profile.Email
		}

		fmt.Printf("%-20s %-15s %-12s %-25s\n", nameTag, status, cdStr, email)
	}

	fmt.Println("\nBuyruqlar:")
	fmt.Println("  • nexus auth add <nomi>   - yangi hisob qo'shish")
	fmt.Println("  • nexus auth test         - barcha hisoblarni tekshirish")
	fmt.Println("  • nexus auth switch <nom> - faol akkauntni almashtirish")
	fmt.Println()

	// If interactive monitoring is requested
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(1 * time.Second):
		return nil
	}
}
