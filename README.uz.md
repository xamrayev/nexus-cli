# Nexus-CLI (DeepSeek Web Bridge for Golang) ⚡

[English](README.md) | [Русский](README.ru.md) | [O'zbekcha](README.uz.md)

---

DeepSeek rasmiy pullik API kalitisiz, bepul veb-akkaunt (`chat.deepseek.com`) orqali har qanday ilovani DeepSeek **V3** (`deepseek-chat`) va **R1** (`deepseek-reasoner` / Thinking) modellariga ulash imkonini beruvchi yuqori tezlikdagi Golang CLI dasturi va mahalliy API proxy serveri.

Ushbu dastur Node.js (`FreeDeepseekAPI-EN`) va Python (`deepseek_bridge`) loyihalarining to'liq tahlili asosida, sof Go (Golang) tilida **nol tashqi muhit bog'liqligisiz (zero runtime dependencies)** qayta yozildi.

---

## 🚀 Asosiy Imkoniyatlar

1. **Sof Golang parallel PoW (Proof-of-Work) Solver:**
   - DeepSeek serverlarining `DeepSeekHashV1` (Keccak-f[1600] 1..23 raundli SHA3-256) algoritmini CPU yadrolari bo'ylab parallel yechadi.
   - 144,000 qiyinlikdagi challenge'ni atigi **~40-50 millisekundda** yechadi (hech qanday tashqi CGO, WASM yoki Python talab qilmaydi).
2. **Interaktiv Terminal Chat (REPL):**
   - R1 modelining fikrlash jarayonini (`reasoning_content`) real vaqt rejimida ajratilgan uslubda (italic/cyan) translatsiya qiladi.
   - Rangli va qulay interfeys, animatsiyali yuklanish ko'rsatkichlari (spinner).
   - Chat ichida tezkor buyruqlar: `/help`, `/new`, `/model`, `/thinking`, `/search`, `/whoami`, `/clear`, `/exit`.
3. **Bir martalik va Quvur (Pipe) buyruqlari:**
   - Terminaldan to'g'ridan-to'g'ri so'rov yuborish: `nexus "Kvant fizikasi nima?"`
   - Linux/Mac quvurlari bilan ishlash: `cat main.go | nexus "Kodni tahlil qil"`
4. **OpenAI, Anthropic va Ollama Mos API Serveri (`nexus serve`):**
   - `POST /v1/chat/completions` (OpenAI format, SSE streaming va JSON)
   - `POST /v1/messages` (Anthropic Messages API — **Claude Code** va Anthropic SDK bilan 100% mos!)
   - `POST /v1/responses` (OpenAI Responses API)
   - `GET /v1/models` va `GET /v1/model-capabilities`
   - `GET /v1/sessions` va `POST /v1/sessions/reset`
   - Open WebUI, Continue.dev, Cursor, Cline va boshqa OpenAI/Anthropic vositalariga ulanadi.
5. **Native Ollama API Emulyatsiyasi 🦙:**
   - `GET /api/tags`, `POST /api/chat`, `POST /api/generate`, `GET /api/version`
   - Open WebUI, Dify, AnythingLLM, LlamaIndex va rasmiy Ollama SDKlari bilan to'liq ishlaydi.
6. **Strukturalangan Natijalar (Structured Outputs & JSON Schema) 🎯:**
   - CLI, Ollama va OpenAI orqali qat'iy JSON formatida javob olish imkoniyati.
7. **Universal Integratsiyalar (Launcher) 🚀:**
   - Claude Code, DeepSeek Harness, Codex, OpenCode, Goose, Aider kabi AI vositalarini bitta buyruq bilan avtomatik konfiguratsiya qilib ishga tushirish.
8. **Ko'p Akkauntlar Hovuzi (Multi-Account Pool) 👥:**
   - Cheklovlarga (HTTP 429) tushmaslik uchun bir nechta DeepSeek akkauntlarini navbatma-navbat (Round-Robin) ishlatish.
9. **Autentifikatsiya Boshqaruvi (`nexus auth`):**
   - Qo'lda kiritish: `nexus auth login`
   - Chrome orqali avtomatik olish: `nexus auth chrome` (CDP orqali brauzer xotirasidagi tokenni avtomatik o'qiydi)
   - Hisob holati: `nexus auth whoami`
10. **Diagnostika (Doctor):**
    - `nexus doctor` — token yaroqliligi, serverga ulanish, PoW tezligi va sessiya yaratishni bir zumda tekshirib beradi.

---

## 📦 O'rnatish va Kompilyatsiya

Go 1.24+ o'rnatilgan bo'lishi kifoya:

```bash
# Repozitoriyani yuklab oling va papkaga o'ting:
git clone https://github.com/xamrayev/nexus-cli.git
cd nexus-cli

# Binar faylni yaratish:
go build -o bin/nexus ./cmd/nexus

# Tizimga global o'rnatish (ixtiyoriy):
go install github.com/xamrayev/nexus-cli/cmd/nexus@latest
```

---

## 🔑 Akkauntni Bog'lash (Autentifikatsiya)

DeepSeek hisobingizni bog'lashning 3 xil usuli mavjud:

### 1. Interaktiv buyruq orqali:
```bash
./bin/nexus auth login
```
Sizdan `userToken` so'raladi:
1. Brauzeringizda `https://chat.deepseek.com` saytiga kiring.
2. `F12` (yoki Mac-da `Cmd+Option+I`) bosing.
3. `Application` -> `Local Storage` -> `https://chat.deepseek.com` -> `userToken` qiymatini nusxalang va dasturga kiriting.

### 2. Chrome orqali avtomatik (CDP):
```bash
./bin/nexus auth chrome
```
Dastur Chrome brauzerini ochadi, siz tizimga kirganingizdan so'ng tokenni avtomatik saqlab oladi.

### 3. `.env` fayl orqali:
Loyiha ildizidagi `.env` fayliga yozing:
```env
DEEPSEEK_TOKEN=sizning_deepseek_tokeningiz
```

---

## 💡 Foydalanish Misollari

### 1. Interaktiv Chat (REPL)
```bash
./bin/nexus chat
```
Yoki shunchaki:
```bash
./bin/nexus
```
Fikrlash (R1) jarayoni real vaqtda chiroyli ko'rsatiladi:
```text
nexus ❯ Nega osmon moviy rangda?

💭 Fikrlash jarayoni (Thinking):
Foydalanuvchi osmonning nima uchun moviy ekanligini so'ramoqda...
Reley sochilish hodisasini (Rayleigh scattering) tushuntirish kerak...

🤖 DeepSeek (deepseek-reasoner):
Osmonning moviy ko'rinishiga asosiy sabab — Reley sochilishi (Rayleigh scattering) hodisasidir...
```

### 2. Bir martalik so'rov (One-off prompt)
```bash
./bin/nexus "O'zbekistonning diqqatga sazovor joylari haqida 3 ta fakt keltir"
```

### 3. Kodni tahlil qilish (Unix Pipe)
```bash
cat main.go | ./bin/nexus "Ushbu kodni xavfsizlik va tezlik bo'yicha tahlil qil"
```

### 4. Mahalliy API Serverni ishga tushirish (Proxy)
```bash
./bin/nexus serve --port 9655
```
Endi har qanday dasturni (Cursor, Open WebUI, Claude Code) `http://localhost:9655/v1` manziliga yo'naltirishingiz mumkin!

**Curl orqali OpenAI formatida so'rov:**
```bash
curl http://localhost:9655/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-reasoner",
    "messages": [{"role": "user", "content": "Salom!"}],
    "stream": true
  }'
```

### 5. Native Ollama API Emulyatsiyasi 🦙
Nexus to'liq Ollama API spetsifikatsiyasini qo'llab-quvvatlaydi! Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex yoki rasmiy Ollama SDKlari to'g'ridan-to'g'ri `http://localhost:9655` ga ulanadi:

```bash
# 1. Modellar ro'yxatini olish (Ollama formati)
curl http://localhost:9655/api/tags

# 2. Ollama Chat formati (NDJSON oqimi)
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Salom!"}],
  "stream": true
}'
```

**Python Ollama SDK orqali foydalanish:**
```python
import ollama

client = ollama.Client(host="http://localhost:9655")
resp = client.chat(model="deepseek-reasoner", messages=[
    {"role": "user", "content": "Go tilida sodda HTTP server kodi"}
])
print(resp["message"]["content"])
```

### 6. Strukturalangan Natijalar (Structured Outputs & JSON Schema) 🎯
Natijani qat'iy JSON formatida yoki ko'rsatilgan JSON Schema asosida olish:

```bash
# 1. CLI orqali toza JSON olish (jq bilan quvurlash mumkin)
./bin/nexus "Eng mashhur 3 ta dasturlash tili va ularning asosiy sohasi" --format json | jq .

# 2. Ollama API orqali:
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "3 ta meva nomini JSON ro'yxat qilib ber"}],
  "format": "json",
  "stream": false
}'

# 3. OpenAI API orqali (response_format):
curl http://localhost:9655/v1/chat/completions -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Telefon ma'lumotlarini JSON qilib ber"}],
  "response_format": {"type": "json_object"}
}'
```

### 7. Universal Integratsiyalar (Launcher) 🚀
Claude Code, DeepSeek Harness, Codex, OpenCode, Goose, Aider kabi barcha AI vositalarini bitta terminal buyrug'i bilan ishga tushirish (Nexus fon rejimida serverni avtomatik ko'taradi):

```bash
# 1. Claude Code CLI (DeepSeek R1 / V3 quvvatida)
./bin/nexus launch claude
./bin/nexus launch claude -p "Ushbu Go kodini refaktoring qil"

# 2. Rasmiy DeepSeek Harness CLI (@deepseek-ai/dsh)
./bin/nexus launch dsh

# 3. OpenAI Codex CLI (avtomatik config va responses wire API)
./bin/nexus launch codex

# 4. OpenCode CLI (OPENCODE_CONFIG_CONTENT avtomatik inyeksiyasi bilan)
./bin/nexus launch opencode

# 5. Block Goose AI Agent (OpenAI provayderi orqali)
./bin/nexus launch goose

# 6. Aider Pair Programming CLI (DeepSeek R1 bilan)
./bin/nexus launch aider

# 7. ChatGPT Desktop (macOS) va mahalliy bridge
./bin/nexus launch chatgpt

# 8. Ko'p akkauntlar hovuzi monitori
./bin/nexus launch pool

# 9. DeepSeek Agentic Eval & Benchmark Harness
./bin/nexus launch harness
```

### 8. Ko'p Akkauntlar Hovuzi (Multi-Account Pool) 👥
Cheklovlarga (HTTP 429) tushmaslik uchun bir nechta DeepSeek akkauntlarini navbatma-navbat (Round-Robin) ishlatish:

```bash
# Yangi akkaunt qo'shish
./bin/nexus auth add acc2 --chrome

# Akkauntlar ro'yxati va holati
./bin/nexus auth list

# Faol asosiy akkauntni almashtirish
./bin/nexus auth switch acc2

# Barcha akkauntlarni tekshirish
./bin/nexus auth test
```

### 9. Tizim diagnostikasi (Doctor)
```bash
./bin/nexus doctor
```

---

## 📁 Loyiha Tuzilishi

```text
nexus-cli/
├── bin/
│   └── nexus                     # Kompilyatsiya qilingan binar fayl
├── cmd/
│   └── nexus/main.go             # Kirish nuqtasi va CLI buyruqlari dispetcheri
├── pkg/
│   ├── deepseek/
│   │   ├── pow.go                # Sof Go parallel DeepSeekHashV1 Keccak PoW solver
│   │   ├── pow_test.go           # PoW testlari va benchmark
│   │   ├── stream.go             # SSE JSON patch va OpenAI stream parser
│   │   ├── stream_test.go        # Stream testlari
│   │   ├── auth.go               # Token, sessiya va profil boshqaruvi
│   │   ├── chrome_auth.go        # Chrome CDP avtomatik autentifikatsiya
│   │   ├── client.go             # DeepSeek Web API mijozi (Chat, Sessions, Auto-retry)
│   │   └── types.go              # Model konfiguratsiyalari va tiplar
│   ├── server/
│   │   ├── server.go             # Mahalliy HTTP proxy server (Routing, CORS, Auth)
│   │   ├── server_test.go        # Server endpoint testlari
│   │   ├── openai.go             # OpenAI Chat Completions API adapteri
│   │   ├── anthropic.go          # Anthropic Messages API adapteri (Claude Code uchun)
│   │   └── session.go            # Agent sessiyalari izolyatsiyasi va tozalash
│   ├── launcher/
│   │   └── launcher.go           # Claude Code, Codex, Goose, Aider launcherlari
│   └── cli/
│       ├── terminal.go           # Ranglar, banner va animatsiyali spinner
│       ├── repl.go               # Interaktiv terminal chat va slash buyruqlar
│       ├── runner.go             # Bir martalik va quvurli (pipe) so'rovlar
│       └── doctor.go             # To'liq tizim diagnostikasi
├── go.mod                        # Go moduli
└── README.md                     # Asosiy hujjat (Ingliz tilida)
```

---

## 📄 Litsenziya

Ushbu loyiha MIT litsenziyasi ostida tarqatiladi.
