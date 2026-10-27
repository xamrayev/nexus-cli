package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExtractCleanJSONText(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"empty", "   ", "{}"},
		{"plain object", `{"a":1}`, `{"a":1}`},
		{"markdown block", "Natija:\n```json\n{\"a\":1}\n```\nrahmat", `{"a":1}`},
		{"object inside prose", `Mana: {"a":{"b":2}} tamom`, `{"a":{"b":2}}`},
		{"array inside prose", `ro'yxat: [1,2,3] tamom`, `[1,2,3]`},
		{"no json", "salom dunyo", "salom dunyo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ExtractCleanJSONText(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseBoolArg(t *testing.T) {
	for _, s := range []string{"1", "true", "TRUE", "yes", "on", "ha"} {
		if !parseBoolArg(s) {
			t.Errorf("parseBoolArg(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"0", "false", "no", "off", "yo'q", ""} {
		if parseBoolArg(s) {
			t.Errorf("parseBoolArg(%q) = true, want false", s)
		}
	}
}

func TestPrintBanner(t *testing.T) {
	var buf bytes.Buffer
	PrintBanner(&buf)
	if strings.TrimSpace(buf.String()) == "" {
		t.Fatal("banner is empty")
	}
}
