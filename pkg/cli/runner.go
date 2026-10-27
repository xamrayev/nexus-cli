package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// RunSinglePrompt executes a one-off prompt or piped input and streams to stdout.
func RunSinglePrompt(ctx context.Context, client *deepseek.Client, prompt string, model string, thinking, search bool, format string) error {
	// Check if stdin has piped data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeBytes, err := io.ReadAll(os.Stdin)
		if err == nil && len(pipeBytes) > 0 {
			pipeContent := string(pipeBytes)
			if prompt == "" {
				prompt = pipeContent
			} else {
				prompt = fmt.Sprintf("%s\n\n[Piped Input]:\n%s", prompt, pipeContent)
			}
		}
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt kiritilmadi")
	}

	isJSON := strings.EqualFold(strings.TrimSpace(format), "json")
	if isJSON {
		prompt += "\n\n[CRITICAL OUTPUT REQUIREMENT]: Respond ONLY with valid, raw JSON. Do NOT wrap in markdown code blocks like ```json ... ```. No preamble or explanations outside the JSON."
	}

	cfg := deepseek.ResolveModelConfig(model)
	if thinking {
		cfg.ThinkingEnabled = true
	}
	if search {
		cfg.SearchEnabled = true
	}

	spinner := NewSpinner("DeepSeek PoW hisoblanmoqda va serverga ulanmoqda...")
	isTerminal := (stat.Mode() & os.ModeCharDevice) != 0
	if isTerminal {
		spinner.Start()
	}

	chunkCh, errCh := client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           cfg.ID,
		ThinkingEnabled: &cfg.ThinkingEnabled,
		SearchEnabled:   &cfg.SearchEnabled,
	})

	hasStarted := false
	inThinking := false
	hasThinkingHeader := false

	var jsonBuffer strings.Builder

	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				if isTerminal {
					spinner.Stop()
				}
				return err
			}
		case chunk, ok := <-chunkCh:
			if !ok {
				if isTerminal {
					spinner.Stop()
				}
				if inThinking {
					fmt.Fprintln(os.Stderr)
				}
				// Check if there was an error
				select {
				case err := <-errCh:
					if err != nil {
						return err
					}
				default:
				}

				if isJSON {
					rawText := jsonBuffer.String()
					clean, _ := ExtractCleanJSONText(rawText)
					fmt.Println(clean)
				}
				return nil
			}

			if !hasStarted {
				if isTerminal {
					spinner.Stop()
				}
				hasStarted = true
			}

			if chunk.ReasoningContent != "" {
				if isTerminal {
					if !hasThinkingHeader {
						fmt.Fprintf(os.Stderr, "%s💭 Fikrlash jarayoni (Thinking):%s\n%s%s", Cyan, Reset, Gray, Italic)
						hasThinkingHeader = true
						inThinking = true
					}
					fmt.Fprint(os.Stderr, chunk.ReasoningContent)
				}
			}

			if chunk.Content != "" {
				if inThinking {
					if isTerminal {
						fmt.Fprintf(os.Stderr, "%s\n\n", Reset)
					}
					inThinking = false
				}
				if isJSON {
					jsonBuffer.WriteString(chunk.Content)
				} else {
					fmt.Print(chunk.Content)
				}
			}
		}
	}
}

// ExtractCleanJSONText extracts clean JSON from raw text or markdown code blocks.
func ExtractCleanJSONText(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}", nil
	}

	// Try removing markdown code block
	if strings.Contains(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		var extracted strings.Builder
		inBlock := false
		for _, line := range lines {
			lineTrimmed := strings.TrimSpace(line)
			if strings.HasPrefix(lineTrimmed, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				extracted.WriteString(line)
				extracted.WriteString("\n")
			}
		}
		cand := strings.TrimSpace(extracted.String())
		if cand != "" {
			return cand, nil
		}
	}

	startObj := strings.Index(trimmed, "{")
	endObj := strings.LastIndex(trimmed, "}")
	startArr := strings.Index(trimmed, "[")
	endArr := strings.LastIndex(trimmed, "]")

	if startObj != -1 && endObj != -1 && endObj > startObj {
		return strings.TrimSpace(trimmed[startObj : endObj+1]), nil
	}
	if startArr != -1 && endArr != -1 && endArr > startArr {
		return strings.TrimSpace(trimmed[startArr : endArr+1]), nil
	}

	return trimmed, nil
}
