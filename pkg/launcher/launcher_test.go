package launcher

import (
	"bytes"
	"strings"
	"testing"
)

// Every platform handled by `nexus launch` must be listed in the help menu.
func TestPrintLaunchMenuListsAllPlatforms(t *testing.T) {
	var buf bytes.Buffer
	PrintLaunchMenu(&buf)
	out := buf.String()

	for _, p := range []string{"claude", "dsh", "codex", "opencode", "goose", "aider", "chatgpt", "harness", "pool"} {
		if !strings.Contains(out, p) {
			t.Errorf("launch menu does not mention %q", p)
		}
	}
	if strings.Contains(out, "%!") {
		t.Errorf("launch menu has a fmt verb mismatch:\n%s", out)
	}
}
