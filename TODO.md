# TODO

## 1. Versiyalash tartibi

### Qoidalar
- **Format:** [Semantic Versioning](https://semver.org/lang/uz/) — `vMAJOR.MINOR.PATCH`, git tag sifatida (`v0.2.0`).
  - **MAJOR** — orqaga mos kelmaydigan o'zgarish: CLI buyrug'i/bayrog'i o'chirildi yoki nomi o'zgardi, API endpoint yo'li yoki javob formati o'zgardi, `accounts/` / `auth.json` formati o'zgardi.
  - **MINOR** — yangi imkoniyat, mos holda: yangi `launch` platformasi, yangi endpoint, yangi model, yangi bayroq.
  - **PATCH** — xato tuzatish, DeepSeek web API o'zgarishiga moslashish, hujjat, ichki refaktor.
- **Pre-release:** `v0.3.0-rc.1`, `v0.3.0-rc.2` … — sinov uchun, GitHub'da "pre-release" deb belgilanadi.
- **Boshlang'ich versiya:** `v0.1.0`. `1.0.0` gacha MINOR ham buzuvchi o'zgarish bo'lishi mumkin — loyiha norasmiy DeepSeek web API'ga tayanadi va u ogohlantirishsiz o'zgarishi mumkin. CLI va API barqaror deb e'lon qilinganda `v1.0.0` chiqariladi.
- **Yagona manba — git tag.** Kodda versiya qo'lda yozilmaydi; build vaqtida tagdan olinadi.
- **Commit xabarlari:** [Conventional Commits](https://www.conventionalcommits.org/) — `feat:` → MINOR, `fix:` → PATCH, `feat!:` yoki `BREAKING CHANGE:` → MAJOR, `docs:`/`test:`/`ci:`/`chore:` → versiya o'zgarmaydi.
- **O'zgarishlar jurnali:** `CHANGELOG.md` ([Keep a Changelog](https://keepachangelog.com/) formati), har relizdan oldin `Unreleased` bo'limi versiya raqamiga aylantiriladi.
- **Tarmoqlar:** `main` doim yashil (CI o'tgan). Ishlar `feat/…`, `fix/…` tarmoqlarda, PR orqali birlashtiriladi. Reliz — `main`dan tag qo'yish.

### Reliz jarayoni
1. `CHANGELOG.md` → `Unreleased` ni `## [0.2.0] - YYYY-MM-DD` ga o'zgartirish.
2. `git tag -a v0.2.0 -m "v0.2.0" && git push origin v0.2.0`
3. Release workflow binarlarni yig'ib, GitHub Release yaratadi.

### Amalga oshirish vazifalari
- [x] `version`, `commit`, `date` — `var`, default `"dev"`; ldflags orqali to'ldiriladi (`cmd/nexus/main.go`).
- [x] ldflags bo'lmasa (`go install ...@v0.1.0`) — versiya va commit Go build ma'lumotidan olinadi.
- [x] `nexus version` commit va build sanasini ko'rsatadi.
- [x] `CHANGELOG.md` yaratildi (birinchi yozuv: `v0.1.0`).
- [x] `.goreleaser.yaml` + `.github/workflows/release.yml` (`v*` tag push'da ishlaydi); CI har PR'da snapshot reliz yig'ib tekshiradi.
- [ ] Birinchi tag: `v0.1.0`.
- [ ] (ixtiyoriy) PR sarlavhasi Conventional Commits formatida ekanini CI'da tekshirish.
- [ ] Eslatma: `pkg/server/ollama.go` dagi `"0.1.32"` — bu emulyatsiya qilinayotgan Ollama versiyasi, Nexus versiyasi emas; tegmaslik.

## 2. Xavfsizlik
- [x] Oshkor bo'lgan `API_KEY` bekor qilindi, yangisi olindi.
- [x] `.env` git tarixidan tozalandi (force push), eski PR tarmog'i o'chirildi.
- [x] `.gitignore` ga `.env` va `.env.*` qo'shildi.

## 3. Ko'rib chiqishda topilgan kamchiliklar
- [x] Dublikat `main.go` o'chirildi; yagona kirish nuqtasi — `cmd/nexus/main.go`. README'lar yangilandi.
- [x] Modul nomi `github.com/xamrayev/nexus-cli`; `go install github.com/xamrayev/nexus-cli/cmd/nexus@latest` ishlaydi.
- [x] `go mod tidy` — `gorilla/websocket` to'g'ridan-to'g'ri bog'liqlik.
- [x] Minimal Go versiyasi `1.24` (README'larda ham).

## 4. Testlar
Hozirgi holat (CI'da): `gofmt`, `go vet`, unit testlar (linux/macos/windows, linuxda `-race`), `scripts/smoke.sh` (CLI buyruqlari + lokal API server), GoReleaser snapshot (5 platforma).

Qamrov: `server` ~25%, `deepseek` ~25%, `cli` ~10%, `launcher` ~0%.
- [ ] `/v1/chat/completions`, `/v1/messages`, `/v1/responses` uchun DeepSeek javobini soxtalashtirilgan (`httptest`) klient bilan testlar — hozir faqat auth/405 tekshiriladi.
- [ ] `deepseek.Client` uchun bazaviy URL'ni sozlanadigan qilish (testda soxta serverga yo'naltirish uchun).
- [ ] `AccountPool` failover (429 → keyingi akkaunt) stsenariysi testi.
- [ ] `launcher` — config fayllarini yozish funksiyalari (opencode, goose, codex) uchun `t.TempDir()` bilan testlar.
- [ ] Haqiqiy DeepSeek bilan integratsion test — alohida, qo'lda ishga tushiriladigan workflow (`workflow_dispatch`), token GitHub Secret'da.
