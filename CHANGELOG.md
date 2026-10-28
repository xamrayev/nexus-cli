# O'zgarishlar jurnali

Barcha muhim o'zgarishlar shu faylda qayd etiladi.

Format [Keep a Changelog](https://keepachangelog.com/) asosida, versiyalash
[Semantic Versioning](https://semver.org/lang/uz/) bo'yicha (qoidalar `TODO.md` → "Versiyalash tartibi").

## [Unreleased]

### Qo'shildi
- `nexus launch claude --api` — Claude Code'ni DeepSeek rasmiy API'siga (`api.deepseek.com/anthropic`) to'g'ridan-to'g'ri ulash: mahalliy proxy va web-akkauntsiz. Kalit `DEEPSEEK_API_KEY` yoki `--api-key` orqali; modellar `-m flash` (default, `deepseek-flash[1m]`) va `-m pro` (`deepseek-v4-pro`).

### Tuzatildi
- `nexus launch claude` endi `ANTHROPIC_AUTH_TOKEN` va `ANTHROPIC_API_KEY` ni birga o'rnatmaydi. Ikkalasi ham qo'yilganda Claude Code "auth may not work as expected" deb ogohlantirar, `ANTHROPIC_API_KEY` ni tashqi auth deb hisoblab claude.ai connector'larini o'chirib qo'yardi. Endi faqat `ANTHROPIC_AUTH_TOKEN` beriladi — proxy uni bearer token sifatida qabul qiladi.
- `nexus launch claude` va `nexus launch --exec` endi meros qilingan `ANTHROPIC_*` o'zgaruvchilarni tozalaydi, shuning uchun foydalanuvchining haqiqiy Anthropic kaliti yoki model sozlamalari proxy sessiyasiga aralashmaydi.

## [0.1.0] - 2026-10-27

Birinchi rasmiy reliz.

### Qo'shildi
- DeepSeek web interfeysi uchun CLI: bitta so'rov, quvur (`cat file | nexus "..."`) va interaktiv REPL rejimlari.
- Lokal API server (`nexus serve`): OpenAI Chat Completions, OpenAI Responses, Anthropic Messages va Ollama mos endpointlar.
- `nexus launch`: Claude Code, Codex, OpenCode, Goose, Aider, DeepSeek Harness, ChatGPT Desktop integratsiyalari.
- Ko'p akkauntli hovuz (round-robin, 429 failover) va `nexus auth` buyruqlari, Chrome orqali avtomatik login.
- `nexus doctor` diagnostikasi va `nexus models` ro'yxati.
- GitHub Actions CI: format, vet, 3 OS'da testlar, CLI smoke test, reliz yig'ilishini tekshirish.
- GoReleaser orqali avtomatik relizlar (linux/darwin amd64+arm64, windows amd64).
- `nexus version` endi commit va build sanasini ham ko'rsatadi.

### O'zgardi
- Modul yo'li `github.com/xamrayev/nexus-cli` ga o'zgardi — endi `go install github.com/xamrayev/nexus-cli/cmd/nexus@latest` ishlaydi.
- Kirish nuqtasi faqat `cmd/nexus/main.go`; build buyrug'i: `go build -o bin/nexus ./cmd/nexus`.
- Minimal Go versiyasi: 1.24.
- Versiya kodda qo'lda yozilmaydi, git tagdan olinadi.

### Xavfsizlik
- `.env` git tarixidan olib tashlandi va `.gitignore` ga qo'shildi.

[Unreleased]: https://github.com/xamrayev/nexus-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/xamrayev/nexus-cli/releases/tag/v0.1.0
