package core

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// TaskType classifies a user request according to the CCR strategy table.
type TaskType string

const (
	TaskTypeUnknown       TaskType = "unknown"
	TaskTypeReasoning     TaskType = "reasoning"     // murakkab arxitektura / chuqur tahlil
	TaskTypeCoding        TaskType = "coding"        // kod yozish / tahrirlash
	TaskTypeSmall         TaskType = "small"         // kichkina vazifa (commit, sarhisob)
	TaskTypeLargeContext  TaskType = "large_context" // katta kod bazasi / hujjat (100k+ token)
	TaskTypeImageOrSearch TaskType = "image_or_search" // rasm generatsiya / veb-qidiruv
)

// RouteDecision is the CCR router's output: which provider/model to use and why.
type RouteDecision struct {
	TaskType  TaskType
	Provider  string
	Model     string
	Reason    string
	Fallback  []string // ordered provider names to try if the primary fails
}

// RuleRouter implements CCR with keyword-based task analysis and the spec's
// fallback chain: Gemini OAuth -> DeepSeek Web -> GLM Web -> Qwen Web.
type RuleRouter struct {
	registry       *Registry
	preferredModel map[TaskType]string
	fallbackChain  []string
}

// NewRuleRouter creates a CCR router wired to the given registry.
func NewRuleRouter(registry *Registry) *RuleRouter {
	return &RuleRouter{
		registry: registry,
		preferredModel: map[TaskType]string{
			TaskTypeReasoning:     "deepseek-reasoner",
			TaskTypeCoding:        "gemini-2.5-pro",
			TaskTypeSmall:         "gemini-2.5-flash",
			TaskTypeLargeContext:  "kimi-k1.5",
			TaskTypeImageOrSearch: "gemini-hybrid",
		},
		// Spec: "Avtomatik Zaxira" chain — Gemini OAuth -> DeepSeek Web -> GLM Web -> Qwen Web.
		fallbackChain: []string{"gemini", "deepseek", "glm", "qwen"},
	}
}

// Classify analyzes the prompt and returns the task type.
func (r *RuleRouter) Classify(prompt string) TaskType {
	t := strings.ToLower(prompt)
	n := utf8.RuneCountInString(prompt)

	// Real-time web search / image generation.
	if containsAny(t, []string{"image", "rasm", "generate image", "qidiruv", "search", "yangilik", "news", "jaring", "real-time", "real time"}) {
		return TaskTypeImageOrSearch
	}

	// Complex architecture / deep reasoning.
	if containsAny(t, []string{"architect", "tahlil", "murakkab", "chuqur", "reason", "reasoning", "mantiq", "fikrlash", "algorithm", "dizayn"}) {
		return TaskTypeReasoning
	}

	// Very long prompt = large-context analysis.
	if n > 6000 {
		return TaskTypeLargeContext
	}

	// Everyday coding / editing.
	if containsAny(t, []string{"code", "kod", "function", "class", "refactor", "bug", "git commit", "sarhisob", "dastur", "edit", "fix"}) {
		return TaskTypeCoding
	}

	return TaskTypeSmall
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// Route picks the provider and model for the prompt.
func (r *RuleRouter) Route(_ context.Context, prompt string) (RouteDecision, error) {
	task := r.Classify(prompt)
	idealModel := r.preferredModel[task]

	// 1. Ideal provider/model from the CCR table.
	if p, _, ok := r.registry.FindModel(idealModel); ok && p.AuthStatus().Ready {
		return RouteDecision{
			TaskType: task,
			Provider: p.Name(),
			Model:    idealModel,
			Reason:   idealReason(task),
			Fallback: r.readyProvidersExcluding(p.Name()),
		}, nil
	}

	// 2. Ideal unavailable: best registered & ready provider for this task.
	if best := r.bestAvailable(task); best.Provider != "" {
		return RouteDecision{
			TaskType: task,
			Provider: best.Provider,
			Model:    best.Model,
			Reason:   fallbackReason(task, idealModel),
			Fallback: r.readyProvidersExcluding(best.Provider),
		}, nil
	}

	// 3. No provider ready at all: best-effort decision with an auth hint.
	return RouteDecision{
		TaskType: task,
		Provider: "",
		Model:    idealModel,
		Reason:   "Hech qanday provayder autentifikatsiya qilinmagan. Avval akkaunt ulang: nexus auth login",
		Fallback: r.fallbackChain,
	}, nil
}

// modelForTask maps a provider + task type to a concrete model id the provider serves.
// Only providers implemented in this repo have entries here; others return "".
func modelForTask(providerName string, task TaskType) string {
	if providerName != "deepseek" {
		return ""
	}
	switch task {
	case TaskTypeReasoning:
		return "deepseek-reasoner"
	case TaskTypeCoding:
		return "deepseek-chat"
	case TaskTypeSmall:
		return "deepseek-chat"
	case TaskTypeLargeContext:
		return "deepseek-reasoner"
	case TaskTypeImageOrSearch:
		return "deepseek-reasoner-search"
	}
	return ""
}

type candidate struct {
	Provider string
	Model    string
}

func (r *RuleRouter) bestAvailable(task TaskType) candidate {
	for _, name := range r.fallbackChain {
		p, ok := r.registry.Provider(name)
		if !ok || !p.AuthStatus().Ready {
			continue
		}
		model := modelForTask(name, task)
		if model == "" {
			// Offer the provider's first model as a generic fallback.
			if models := p.Models(); len(models) > 0 {
				model = models[0].ID
			}
		}
		if model != "" {
			return candidate{Provider: name, Model: model}
		}
	}
	return candidate{}
}

func (r *RuleRouter) readyProvidersExcluding(name string) []string {
	var out []string
	for _, n := range r.fallbackChain {
		if n == name {
			continue
		}
		if p, ok := r.registry.Provider(n); ok && p.AuthStatus().Ready {
			out = append(out, n)
		}
	}
	return out
}

func idealReason(task TaskType) string {
	switch task {
	case TaskTypeReasoning:
		return "Murakkab arxitektura va chuqur tahlil uchun deepseek-reasoner (R1) fikrlash zanjiri ishlatiladi."
	case TaskTypeCoding:
		return "Kundalik kod yozish uchun gemini-2.5-pro (OAuth kanali) tavsiya etiladi."
	case TaskTypeSmall:
		return "Kichkina fon vazifasi uchun kam-kechiklikli gemini-2.5-flash / glm-4-plus."
	case TaskTypeLargeContext:
		return "Katta kontekstli kod bazasi / hujjat tahlili uchun kimi-k1.5 / gemini-2.5-pro."
	case TaskTypeImageOrSearch:
		return "Rasm generatsiya va real-vaqt veb-qidiruv uchun gemini-hybrid (Web Cookie kanali)."
	}
	return "CCR standart qoidasi bo'yicha model tanlandi."
}

func fallbackReason(task TaskType, idealModel string) string {
	return fmt.Sprintf("Ideal provayder/model (%s) mavjud emas yoki autentifikatsiya qilinmagan, shuning uchun mavjud provayder orqali bajariladi.", idealModel)
}
