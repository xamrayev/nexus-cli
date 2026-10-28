# Nexus-CLI (DeepSeek Web Bridge for Golang) ⚡

[English](README.md) | [Русский](README.ru.md) | [O'zbekcha](README.uz.md)

---

Высокоскоростная CLI-утилита на Golang и локальный API прокси-сервер, позволяющий любому приложению подключаться к моделям DeepSeek **V3** (`deepseek-chat`) и **R1** (`deepseek-reasoner` / Thinking) через бесплатный веб-аккаунт (`chat.deepseek.com`), без необходимости иметь официальный платный API-ключ.

Проект переписан с нуля на чистом Go (Golang) на основе детального анализа реализаций на Node.js (`FreeDeepseekAPI-EN`) и Python (`deepseek_bridge`), с **полным отсутствием внешних зависимостей времени выполнения (zero runtime dependencies)**.

---

## 🚀 Основные возможности

1. **Параллельный PoW (Proof-of-Work) солвер на чистом Go:**
   - Параллельное вычисление алгоритма `DeepSeekHashV1` (Keccak-f[1600] 1..23 раундов SHA3-256) по всем ядрам процессора.
   - Решение задач со сложностью 144,000 всего за **~40-50 миллисекунд** (без использования CGO, WASM или Python).
2. **Интерактивный терминальный чат (REPL):**
   - Стриминг цепочки рассуждений модели R1 (`reasoning_content`) в реальном времени с выделенным стилем (курсив/cyan).
   - Цветной и интуитивный UI, анимированные индикаторы загрузки (спиннеры).
   - Быстрые слэш-команды внутри чата: `/help`, `/new`, `/model`, `/thinking`, `/search`, `/whoami`, `/clear`, `/exit`.
3. **Одиночные запросы и поддержка Unix Pipe:**
   - Прямой запрос из терминала: `nexus "Что такое квантовая физика?"`
   - Работа с пайпами Linux/macOS: `cat main.go | nexus "Проанализируй этот код"`
4. **Совместимый API-сервер OpenAI, Anthropic и Ollama (`nexus serve`):**
   - `POST /v1/chat/completions` (формат OpenAI, SSE стриминг и JSON)
   - `POST /v1/messages` (Anthropic Messages API — 100% совместимость с **Claude Code** и Anthropic SDK!)
   - `POST /v1/responses` (OpenAI Responses API)
   - `GET /v1/models` и `GET /v1/model-capabilities`
   - `GET /v1/sessions` и `POST /v1/sessions/reset`
   - Бесшовная интеграция с Open WebUI, Continue.dev, Cursor, Cline и другими инструментами.
5. **Нативная эмуляция Ollama API 🦙:**
   - `GET /api/tags`, `POST /api/chat`, `POST /api/generate`, `GET /api/version`
   - Прямое подключение к Open WebUI, Dify, AnythingLLM, LlamaIndex и официальным SDK Ollama.
6. **Структурированный вывод (Structured Outputs & JSON Schema) 🎯:**
   - Получение строгого JSON или валидация по схеме через CLI, Ollama и OpenAI endpoints.
7. **Универсальные интеграции (Launcher) 🚀:**
   - Запуск Claude Code, DeepSeek Harness, Codex, OpenCode, Goose, Aider одной командой (Nexus автоматически поднимает сервер в фоновом режиме).
8. **Пул нескольких аккаунтов (Multi-Account Pool) 👥:**
   - Ротация нескольких аккаунтов DeepSeek по алгоритму Round-Robin для защиты от лимитов запросов (HTTP 429).
9. **Управление аутентификацией (`nexus auth`):**
   - Ручной ввод: `nexus auth login`
   - Автоматический сбор токена через Chrome: `nexus auth chrome` (извлечение токена из памяти браузера через CDP)
   - Проверка статуса аккаунта: `nexus auth whoami`
10. **Диагностика системы (Doctor):**
    - `nexus doctor` — мгновенная проверка валидности токена, доступности серверов, скорости PoW-солвера и создания сессий.

---

## 📦 Установка и сборка

Требуется установленный Go версии 1.24+:

```bash
# Клонируйте репозиторий и перейдите в папку:
git clone https://github.com/xamrayev/nexus-cli.git
cd nexus-cli

# Сборка бинарного файла:
go build -o bin/nexus ./cmd/nexus

# Глобальная установка в систему (по желанию):
go install github.com/xamrayev/nexus-cli/cmd/nexus@latest
```

---

## 🔑 Аутентификация (Подключение аккаунта)

Поддерживается 3 способа привязки аккаунта DeepSeek:

### 1. Интерактивная команда:
```bash
./bin/nexus auth login
```
Утилита запросит `userToken`:
1. Откройте в браузере `https://chat.deepseek.com`.
2. Нажмите `F12` (на Mac: `Cmd+Option+I`).
3. Перейдите во вкладку `Application` -> `Local Storage` -> `https://chat.deepseek.com` -> найдите и скопируйте значение `userToken`.

### 2. Автоматически через Chrome (CDP):
```bash
./bin/nexus auth chrome
```
Программа запустит окно Chrome, после вашего входа в аккаунт токен сохранится автоматически.

### 3. Через файл `.env`:
Создайте в корне проекта файл `.env`:
```env
DEEPSEEK_TOKEN=ваш_deepseek_token
```

---

## 💡 Примеры использования

### 1. Интерактивный чат (REPL)
```bash
./bin/nexus chat
```
Или просто:
```bash
./bin/nexus
```
Процесс рассуждений (R1 Thinking) транслируется в реальном времени:
```text
nexus ❯ Почему небо голубое?

💭 Процесс рассуждений (Thinking):
Пользователь интересуется физической причиной цвета неба...
Необходимо объяснить механизм рэлеевского рассеяния света...

🤖 DeepSeek (deepseek-reasoner):
Основная причина голубого цвета неба — рэлеевское рассеяние (Rayleigh scattering) солнечного света...
```

### 2. Одиночный запрос (One-off prompt)
```bash
./bin/nexus "Назови 3 интересных факта о космосе"
```

### 3. Анализ кода через Unix Pipe
```bash
cat main.go | ./bin/nexus "Проанализируй этот код на предмет безопасности и производительности"
```

### 4. Запуск локального API прокси-сервера
```bash
./bin/nexus serve --port 9655
```
Теперь любые клиенты (Cursor, Open WebUI, Claude Code) можно направить на адрес `http://localhost:9655/v1`!

**Запрос в формате OpenAI через cURL:**
```bash
curl http://localhost:9655/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-reasoner",
    "messages": [{"role": "user", "content": "Привет!"}],
    "stream": true
  }'
```

### 5. Нативная эмуляция Ollama API 🦙
Nexus полностью воспроизводит спецификацию Ollama API. Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex и официальные библиотеки Ollama работают «из коробки»:

```bash
# 1. Получение списка моделей (формат Ollama)
curl http://localhost:9655/api/tags

# 2. Ollama Chat запрос (NDJSON стриминг)
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Привет!"}],
  "stream": true
}'
```

**Использование через официальный Python Ollama SDK:**
```python
import ollama

client = ollama.Client(host="http://localhost:9655")
resp = client.chat(model="deepseek-reasoner", messages=[
    {"role": "user", "content": "Напиши простой HTTP-сервер на Go"}
])
print(resp["message"]["content"])
```

### 6. Структурированный вывод (Structured Outputs & JSON Schema) 🎯
Получение строгого JSON или генерация по заданной схеме:

```bash
# 1. Через CLI в чистом JSON (удобно передавать в jq)
./bin/nexus "3 самых популярных языка программирования" --format json | jq .

# 2. Через эмуляцию Ollama API:
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Выведи список из 3 фруктов в JSON"}],
  "format": "json",
  "stream": false
}'

# 3. Через OpenAI API (response_format):
curl http://localhost:9655/v1/chat/completions -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Сформируй информацию об устройстве в JSON"}],
  "response_format": {"type": "json_object"}
}'
```

### 7. Универсальные интеграции (Launcher) 🚀
Запуск Claude Code, DeepSeek Harness, Codex, OpenCode, Goose, Aider одной командой (Nexus сам поднимет прокси в фоне при необходимости):

```bash
# 1. Claude Code CLI (на базе DeepSeek R1 / V3)
./bin/nexus launch claude
./bin/nexus launch claude -p "Проведи рефакторинг этого Go-кода"

# 2. Официальный DeepSeek Harness CLI (@deepseek-ai/dsh)
./bin/nexus launch dsh

# 3. OpenAI Codex CLI (автоматическая конфигурация и Responses Wire API)
./bin/nexus launch codex

# 4. OpenCode CLI (с автоматической инъекцией конфигурации)
./bin/nexus launch opencode

# 5. Block Goose AI Agent (через OpenAI provider)
./bin/nexus launch goose

# 6. Aider Pair Programming CLI (с DeepSeek R1)
./bin/nexus launch aider

# 7. ChatGPT Desktop (macOS) и локальный мост
./bin/nexus launch chatgpt

# 8. Мониторинг пула аккаунтов
./bin/nexus launch pool

# 9. DeepSeek Agentic Eval & Benchmark Harness
./bin/nexus launch harness
```

#### Claude Code + официальный API DeepSeek (`--api`)
Использует официальный Anthropic-совместимый API DeepSeek (`https://api.deepseek.com/anthropic`) вместо web-моста: без локального прокси, без web-аккаунта и без PoW. Платно, но стабильно и без риска блокировки аккаунта.

```bash
export DEEPSEEK_API_KEY=sk-...          # https://platform.deepseek.com/api_keys
./bin/nexus launch claude --api         # deepseek-flash (контекст 1M)
./bin/nexus launch claude --api -m pro  # deepseek-v4-pro
```

Унаследованные переменные `ANTHROPIC_*` (например, настоящий ключ Anthropic) удаляются перед запуском Claude Code и не попадают в DeepSeek. Лучше задавать ключ через `DEEPSEEK_API_KEY`, а не `--api-key`: ключ в командной строке остаётся в истории shell.

### 8. Пул нескольких аккаунтов (Multi-Account Pool) 👥
Использование нескольких аккаунтов DeepSeek по очереди (Round-Robin) для устранения проблем с лимитами (HTTP 429):

```bash
# Добавить новый аккаунт
./bin/nexus auth add acc2 --chrome

# Список аккаунтов и их статус
./bin/nexus auth list

# Переключить активный аккаунт
./bin/nexus auth switch acc2

# Протестировать доступность всех аккаунтов
./bin/nexus auth test
```

### 9. Диагностика системы (Doctor)
```bash
./bin/nexus doctor
```

---

## 📁 Структура проекта

```text
nexus-cli/
├── bin/
│   └── nexus                     # Скомпилированный исполняемый файл
├── cmd/
│   └── nexus/main.go             # Точка входа и диспетчер команд CLI
├── pkg/
│   ├── deepseek/
│   │   ├── pow.go                # Параллельный Keccak-солвер DeepSeekHashV1 на чистом Go
│   │   ├── pow_test.go           # Тесты и бенчмарки PoW
│   │   ├── stream.go             # Парсер SSE JSON patch и стриминга OpenAI
│   │   ├── stream_test.go        # Тесты парсера стриминга
│   │   ├── auth.go               # Управление токенами, сессиями и профилями
│   │   ├── chrome_auth.go        # Автоматическая авторизация через Chrome CDP
│   │   ├── client.go             # Клиент DeepSeek Web API (чат, сессии, автоповторы)
│   │   └── types.go              # Конфигурации моделей и типы данных
│   ├── server/
│   │   ├── server.go             # Локальный HTTP прокси-сервер (маршрутизация, CORS, авторизация)
│   │   ├── server_test.go        # Тесты эндпоинтов сервера
│   │   ├── openai.go             # Адаптер OpenAI Chat Completions API
│   │   ├── anthropic.go          # Адаптер Anthropic Messages API (для Claude Code)
│   │   └── session.go            # Изоляция сессий агентов и очистка контекста
│   ├── launcher/
│   │   └── launcher.go           # Лаунчеры для Claude Code, Codex, Goose, Aider и др.
│   └── cli/
│       ├── terminal.go           # Стилизация, цвета, баннеры и спиннеры
│       ├── repl.go               # Интерактивный чат в терминале и слэш-команды
│       ├── runner.go             # Обработка одиночных и конвейерных (pipe) запросов
│       └── doctor.go             # Полная диагностика системы
├── go.mod                        # Go модуль
└── README.md                     # Документация проекта (на английском языке)
```

---

## 📄 Лицензия

Проект распространяется под лицензией MIT.
