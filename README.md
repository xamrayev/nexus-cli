# Nexus-CLI (DeepSeek Web Bridge for Golang) ⚡

[English](README.md) | [Русский](README.ru.md) | [O'zbekcha](README.uz.md)

---

A high-performance Golang CLI utility and local API proxy server that enables any application or coding agent to connect to DeepSeek **V3** (`deepseek-chat`) and **R1** (`deepseek-reasoner` / Thinking) models via a free web account (`chat.deepseek.com`), without requiring an official paid DeepSeek API key.

Rewritten from scratch in pure Go (Golang) based on in-depth analysis of Node.js (`FreeDeepseekAPI-EN`) and Python (`deepseek_bridge`) projects, featuring **zero runtime dependencies**.

---

## 🚀 Key Features

1. **Pure Go Parallel PoW (Proof-of-Work) Solver:**
   - Solves DeepSeek server challenges using the `DeepSeekHashV1` algorithm (Keccak-f[1600] 1..23 rounds SHA3-256) parallelized across CPU cores.
   - Solves difficulty 144,000 challenges in just **~40–50 milliseconds** (no CGO, WASM, or Python dependencies required).
2. **Interactive Terminal Chat (REPL):**
   - Streams the R1 model's thought process (`reasoning_content`) in real-time with dedicated styling (italic/cyan).
   - Polished ANSI UI with colorful prompts and animated loading spinners.
   - Built-in slash commands: `/help`, `/new`, `/model`, `/thinking`, `/search`, `/whoami`, `/clear`, `/exit`.
3. **One-Off & Unix Pipe Support:**
   - Direct CLI queries: `nexus "What is quantum computing?"`
   - Linux/macOS pipe workflows: `cat main.go | nexus "Review this code for security and performance"`
4. **OpenAI, Anthropic & Ollama Compatible Proxy Server (`nexus serve`):**
   - `POST /v1/chat/completions` (OpenAI format, SSE streaming, and JSON)
   - `POST /v1/messages` (Anthropic Messages API — 100% compatible with **Claude Code** and the Anthropic SDK!)
   - `POST /v1/responses` (OpenAI Responses API)
   - `GET /v1/models` and `GET /v1/model-capabilities`
   - `GET /v1/sessions` and `POST /v1/sessions/reset`
   - Connects seamlessly with Open WebUI, Continue.dev, Cursor, Cline, and any other OpenAI/Anthropic client.
5. **Native Ollama API Emulation 🦙:**
   - Emulates `GET /api/tags`, `POST /api/chat`, `POST /api/generate`, `GET /api/version`.
   - Direct plug-and-play compatibility with Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex, and official Ollama SDKs.
6. **Structured Outputs & JSON Schema 🎯:**
   - Enforce strict JSON output or validate against JSON Schemas across CLI, Ollama, and OpenAI endpoints.
7. **Universal Integrations (Launcher) 🚀:**
   - One-command launcher for Claude Code, DeepSeek Harness, Codex, OpenCode, Goose, and Aider (Nexus automatically boots the proxy in the background).
8. **Multi-Account Pool 👥:**
   - Rotate multiple DeepSeek accounts using Round-Robin scheduling to eliminate rate-limit roadblocks (HTTP 429).
9. **Authentication Management (`nexus auth`):**
   - Manual login: `nexus auth login` (extract `userToken` from browser LocalStorage)
   - Automatic via Chrome: `nexus auth chrome` (reads session token from browser memory via CDP)
   - Account inspection: `nexus auth whoami`
10. **System Diagnostics (Doctor):**
    - `nexus doctor` — instantly checks token validity, server connectivity, PoW solver performance, and session creation.

---

## 📦 Installation & Build

Requires Go 1.24+:

```bash
# Clone repository and navigate to directory:
git clone https://github.com/xamrayev/nexus-cli.git
cd nexus-cli

# Build the executable binary:
go build -o bin/nexus ./cmd/nexus

# Install globally to system PATH (optional):
go install github.com/xamrayev/nexus-cli/cmd/nexus@latest
```

---

## 🔑 Authentication (Account Linking)

There are 3 ways to connect your DeepSeek account:

### 1. Interactive CLI prompt:
```bash
./bin/nexus auth login
```
You will be prompted for your `userToken`:
1. Open `https://chat.deepseek.com` in your browser.
2. Press `F12` (or `Cmd+Option+I` on macOS).
3. Navigate to `Application` -> `Local Storage` -> `https://chat.deepseek.com` -> copy the `userToken` value and paste it into the prompt.

### 2. Automatic via Chrome (CDP):
```bash
./bin/nexus auth chrome
```
This automatically launches a Chrome window. Once you log in, Nexus securely captures and stores the session token.

### 3. Via `.env` file:
Create a `.env` file in the project root:
```env
DEEPSEEK_TOKEN=your_deepseek_token_here
```

---

## 💡 Usage Examples

### 1. Interactive Chat (REPL)
```bash
./bin/nexus chat
```
Or simply:
```bash
./bin/nexus
```
Real-time R1 reasoning stream:
```text
nexus ❯ Why is the sky blue?

💭 Thinking Process:
The user is asking about the physical cause behind the color of the sky...
Need to explain the mechanism of Rayleigh scattering of sunlight by atmospheric molecules...

🤖 DeepSeek (deepseek-reasoner):
The primary reason the sky appears blue is due to a phenomenon called Rayleigh scattering...
```

### 2. One-off prompt
```bash
./bin/nexus "Provide 3 key facts about general relativity"
```

### 3. Code analysis via Unix Pipe
```bash
cat main.go | ./bin/nexus "Analyze this code for security vulnerabilities and performance bottlenecks"
```

### 4. Running the Local API Proxy Server
```bash
./bin/nexus serve --port 9655
```
You can now route any tool (Cursor, Open WebUI, Claude Code) to `http://localhost:9655/v1`!

**Sample cURL request in OpenAI format:**
```bash
curl http://localhost:9655/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-reasoner",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### 5. Native Ollama API Emulation 🦙
Nexus implements the Ollama API specification. Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex, and official Ollama libraries can connect directly to `http://localhost:9655`:

```bash
# 1. Fetch available models (Ollama format)
curl http://localhost:9655/api/tags

# 2. Ollama Chat request (NDJSON streaming)
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Hello!"}],
  "stream": true
}'
```

**Using the official Python Ollama SDK:**
```python
import ollama

client = ollama.Client(host="http://localhost:9655")
resp = client.chat(model="deepseek-reasoner", messages=[
    {"role": "user", "content": "Write a simple HTTP server in Go"}
])
print(resp["message"]["content"])
```

### 6. Structured Outputs & JSON Schema 🎯
Enforce strict JSON output or conform to a specified schema:

```bash
# 1. Direct JSON from CLI (pipeable to jq)
./bin/nexus "Top 3 programming languages and their primary domains" --format json | jq .

# 2. Via Ollama API:
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "List 3 fruits in JSON"}],
  "format": "json",
  "stream": false
}'

# 3. Via OpenAI API (response_format):
curl http://localhost:9655/v1/chat/completions -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Return device specs as JSON"}],
  "response_format": {"type": "json_object"}
}'
```

### 7. Universal Integrations (Launcher) 🚀
Launch coding tools and agents with a single command — Nexus automatically boots the proxy server in the background:

```bash
# 1. Claude Code CLI (powered by DeepSeek R1 / V3)
./bin/nexus launch claude
./bin/nexus launch claude -p "Refactor this Go code"

# 2. Official DeepSeek Harness CLI (@deepseek-ai/dsh)
./bin/nexus launch dsh

# 3. OpenAI Codex CLI (auto-configured with responses wire API)
./bin/nexus launch codex

# 4. OpenCode CLI (with automated config injection)
./bin/nexus launch opencode

# 5. Block Goose AI Agent (via OpenAI provider)
./bin/nexus launch goose

# 6. Aider Pair Programming CLI (with DeepSeek R1)
./bin/nexus launch aider

# 7. ChatGPT Desktop (macOS) & local bridge
./bin/nexus launch chatgpt

# 8. Multi-Account Pool Monitor
./bin/nexus launch pool

# 9. DeepSeek Agentic Eval & Benchmark Harness
./bin/nexus launch harness
```

#### Claude Code + DeepSeek official API (`--api`)
Uses DeepSeek's official Anthropic-compatible API (`https://api.deepseek.com/anthropic`) instead of the web bridge: no local proxy, no web account, no PoW. It is paid, but stable and free of web-account ban risk.

```bash
export DEEPSEEK_API_KEY=sk-...          # https://platform.deepseek.com/api_keys
./bin/nexus launch claude --api         # deepseek-flash (1M context)
./bin/nexus launch claude --api -m pro  # deepseek-v4-pro
```

Inherited `ANTHROPIC_*` variables (e.g. a real Anthropic key) are removed before Claude Code starts, so they never reach DeepSeek. Prefer `DEEPSEEK_API_KEY` over `--api-key`: a key on the command line stays in shell history.

### 8. Multi-Account Pool 👥
Avoid rate limits (HTTP 429) by cycling through multiple DeepSeek accounts using Round-Robin rotation:

```bash
# Add a new account
./bin/nexus auth add acc2 --chrome

# List accounts and check their status
./bin/nexus auth list

# Switch active account
./bin/nexus auth switch acc2

# Validate all accounts in the pool
./bin/nexus auth test
```

### 9. System Diagnostics (Doctor)
```bash
./bin/nexus doctor
```

---

## 📁 Project Structure

```text
nexus-cli/
├── bin/
│   └── nexus                     # Compiled binary executable
├── cmd/
│   └── nexus/main.go             # Application entry point & CLI command dispatcher
├── pkg/
│   ├── deepseek/
│   │   ├── pow.go                # Pure Go parallel DeepSeekHashV1 Keccak PoW solver
│   │   ├── pow_test.go           # PoW test suite & benchmarks
│   │   ├── stream.go             # SSE JSON patch & OpenAI stream parser
│   │   ├── stream_test.go        # Stream tests
│   │   ├── auth.go               # Token, session, and profile manager
│   │   ├── chrome_auth.go        # Automated Chrome CDP authentication
│   │   ├── client.go             # DeepSeek Web API client (Chat, Sessions, Auto-retry)
│   │   └── types.go              # Model configs and type definitions
│   ├── server/
│   │   ├── server.go             # Local HTTP proxy server (Routing, CORS, Auth)
│   │   ├── server_test.go        # Server endpoint unit tests
│   │   ├── openai.go             # OpenAI Chat Completions API adapter
│   │   ├── anthropic.go          # Anthropic Messages API adapter (for Claude Code)
│   │   └── session.go            # Agent session isolation and cleanup
│   ├── launcher/
│   │   └── launcher.go           # Launchers for Claude Code, Codex, Goose, Aider, etc.
│   └── cli/
│       ├── terminal.go           # ANSI styling, banners, and spinners
│       ├── repl.go               # Interactive terminal chat & slash commands
│       ├── runner.go             # Single-prompt and Unix pipe runner
│       └── doctor.go             # End-to-end system diagnostics
├── go.mod                        # Go module definition
├── README.md                     # Main documentation (English)
├── README.ru.md                  # Russian documentation (Русский)
└── README.uz.md                  # Uzbek documentation (O'zbekcha)
```

---

## 📄 License

Distributed under the MIT License.
