package core

import (
	"context"
	"strings"
	"testing"
)

// stubProvider is a registry-safe test double for Provider.
type stubProvider struct {
	name   string
	models []ModelInfo
	ready  bool
}

func (s stubProvider) Name() string { return s.name }

func (s stubProvider) Models() []ModelInfo {
	out := make([]ModelInfo, len(s.models))
	copy(out, s.models)
	return out
}

func (s stubProvider) Capability(model string) Capability {
	for _, m := range s.models {
		if m.ID == model {
			return m.Caps
		}
	}
	return Capability{}
}

func (s stubProvider) AuthStatus() AuthStatus {
	return AuthStatus{Provider: s.name, Ready: s.ready}
}

func (s stubProvider) ChatStream(context.Context, ChatRequest) (<-chan ChatChunk, <-chan error) {
	chunkCh := make(chan ChatChunk, 1)
	errCh := make(chan error, 1)
	close(chunkCh)
	close(errCh)
	return chunkCh, errCh
}

func TestClassify(t *testing.T) {
	r := &RuleRouter{}
	cases := []struct {
		prompt string
		want   TaskType
	}{
		{"Explain the architecture of this distributed system", TaskTypeReasoning},
		{"chuqur tahlil va murakkab mantiq kerak", TaskTypeReasoning},
		{"Write a Go function to sort a slice", TaskTypeCoding},
		{"git commit barcha o'zgarishlarni sarhisob qiling", TaskTypeSmall},
		{"generate an image of a cat", TaskTypeImageOrSearch},
		{"qidiruv yangiliklar bo'yicha", TaskTypeImageOrSearch},
		{"hello", TaskTypeSmall},
		{strings.Repeat("x ", 1000), TaskTypeLargeContext},
		{"", TaskTypeSmall},
	}
	for _, c := range cases {
		if got := r.Classify(c.prompt); got != c.want {
			t.Errorf("Classify(%q) = %v, want %v", c.prompt, got, c.want)
		}
	}
}

func TestRoutePrefersReadyIdealProvider(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{name: "deepseek", ready: true,
		models: []ModelInfo{{ID: "deepseek-reasoner", Caps: Capability{Reasoning: true}}}})
	r := NewRuleRouter(reg)

	d, err := r.Route(context.Background(), "Explain the architecture of this system")
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "deepseek" || d.Model != "deepseek-reasoner" {
		t.Errorf("Route = %+v, want provider deepseek model deepseek-reasoner", d)
	}
}

func TestRouteFallsBackWhenIdealMissing(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{name: "deepseek", ready: true,
		models: []ModelInfo{{ID: "deepseek-chat"}}})
	r := NewRuleRouter(reg)

	d, err := r.Route(context.Background(), "Write a Go function")
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "deepseek" || d.Model != "deepseek-chat" {
		t.Errorf("Route = %+v, want deepseek/deepseek-chat fallback", d)
	}
	if !strings.Contains(d.Reason, "mavjud emas") {
		t.Errorf("Reason should mention fallback, got: %s", d.Reason)
	}
}

func TestRouteNoProviderReady(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{name: "deepseek", ready: false})
	r := NewRuleRouter(reg)

	d, err := r.Route(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "" {
		t.Errorf("Route.Provider = %q, want empty when no provider ready", d.Provider)
	}
	if !strings.Contains(d.Reason, "autentifikatsiya qilinmagan") {
		t.Errorf("Reason should hint at auth, got: %s", d.Reason)
	}
}
