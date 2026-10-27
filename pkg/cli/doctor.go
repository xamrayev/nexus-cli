package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// RunDoctor performs end-to-end diagnostics on DeepSeek API connectivity, auth, and PoW.
func RunDoctor(ctx context.Context, client *deepseek.Client) error {
	fmt.Printf("%s%s=== Nexus-CLI / DeepSeek Diagnostika (Doctor) ===%s\n\n", BrightCyan, Bold, Reset)

	allOK := true

	// 1. Auth File & Token Check
	auth := client.Auth()
	token := auth.GetToken()
	data := auth.GetData()

	fmt.Print("[1/4] Autentifikatsiya tokeni tekshirilmoqda: ")
	if token == "" {
		fmt.Printf("%s❌ Token topilmadi!%s\n", Red, Reset)
		fmt.Println("      Qanday to'g'irlash kerak:")
		fmt.Println("      1. 'nexus auth login' buyrug'ini ishga tushiring")
		fmt.Println("      2. Yoki .env fayliga DEEPSEEK_TOKEN=sizning_tokeningiz deb yozing")
		allOK = false
	} else {
		masked := token
		if len(token) > 12 {
			masked = token[:6] + "..." + token[len(token)-4:]
		}
		fmt.Printf("%s✅ Mavjud (%s)%s\n", BrightGreen, masked, Reset)
		if data.Cookie != "" {
			fmt.Printf("      Cookies: %s✅ Bor (%d belgi)%s\n", Gray, len(data.Cookie), Reset)
		}
	}

	// 2. Validate Token via DeepSeek API
	fmt.Print("[2/4] DeepSeek hisob ma'lumotlari tekshirilmoqda: ")
	if token != "" {
		profile, err := auth.ValidateToken(ctx)
		if err != nil {
			fmt.Printf("%s❌ Xatolik (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			if name == "" {
				name = profile.ID
			}
			fmt.Printf("%s✅ Muvaffaqiyatli! Foydalanuvchi: %s (ID: %s)%s\n", BrightGreen, name, profile.ID, Reset)
		}
	} else {
		fmt.Printf("%s⚠️ O'tkazib yuborildi (token yo'q)%s\n", Yellow, Reset)
	}

	// 3. PoW Challenge & Solver Speed
	fmt.Print("[3/4] PoW (Proof-of-Work) challenge va Keccak tezligi: ")
	if token != "" {
		start := time.Now()
		powResp, err := client.RequestAndSolvePoW(ctx, "/api/v0/chat/completion")
		duration := time.Since(start)
		if err != nil {
			fmt.Printf("%s❌ PoW hal qilinmadi (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			fmt.Printf("%s✅ Muvaffaqiyatli! Hal qilish vaqti: %v (B64 len: %d)%s\n", BrightGreen, duration, len(powResp), Reset)
		}
	} else {
		fmt.Printf("%s⚠️ O'tkazib yuborildi%s\n", Yellow, Reset)
	}

	// 4. Session Creation
	fmt.Print("[4/4] Chat sessiyasi yaratish (/api/v0/chat_session/create): ")
	if token != "" {
		sessionID, err := client.CreateSession(ctx)
		if err != nil {
			fmt.Printf("%s❌ Sessiya yaratilmadi (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			fmt.Printf("%s✅ Sessiya yaratildi (ID: %s)%s\n", BrightGreen, sessionID, Reset)
		}
	} else {
		fmt.Printf("%s⚠️ O'tkazib yuborildi%s\n", Yellow, Reset)
	}

	fmt.Println()
	if allOK {
		fmt.Printf("%s🎉 Barcha testlar muvaffaqiyatli o'tdi! Dastur ishlashga to'liq tayyor.%s\n", BrightGreen, Reset)
	} else {
		fmt.Printf("%s⚠️ Ba'zi testlarda xatolik aniqlandi. Yuqoridagi ko'rsatmalarga qarang.%s\n", Yellow, Reset)
	}

	return nil
}
