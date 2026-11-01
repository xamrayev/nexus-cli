package cli

import (
	"log"

	"github.com/xamrayev/go-deepseek-web/server"
)

// ServiceName labels the proxy in its /health payload.
const ServiceName = "Nexus-CLI FreeDeepSeek Proxy"

// serverLogTag prefixes the startup lines.
const serverLogTag = "Nexus Server"

// ServerListenLogger returns the startup logger for server.ServerOptions.OnListen.
// The library's own lines are English, so Nexus supplies these instead.
func ServerListenLogger() func(server.ListenInfo) {
	return func(info server.ListenInfo) {
		log.Printf("[%s] Tinglamoqda: http://%s (Port: %d)", serverLogTag, info.Addr, info.Port)
		if info.ReadyAccounts > 0 {
			log.Printf("[%s] Akkauntlar hovuzi: %d ta faol hisob yuklandi (Round-Robin & 429 Failover faol)", serverLogTag, info.ReadyAccounts)
		}
		log.Printf("[%s] OpenAI Chat: POST http://localhost:%d/v1/chat/completions", serverLogTag, info.Port)
		log.Printf("[%s] Anthropic Messages (Claude Code): POST http://localhost:%d/v1/messages", serverLogTag, info.Port)
		log.Printf("[%s] Modellarni ko'rish: GET http://localhost:%d/v1/models", serverLogTag, info.Port)
	}
}
