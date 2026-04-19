package claude

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	hintTimeout         = 3 * time.Second
	maxHintFieldLength  = 64
	maxFallbackHintSize = 128
)

func GenerateHint(word, wrong string, client *ClaudeClient) string {
	word = sanitizeHintField(word)
	wrong = sanitizeHintField(wrong)
	if word == "" {
		return ""
	}

	if client == nil || client.httpClient == nil {
		return fallbackSyllableHint(word)
	}

	ctx, cancel := context.WithTimeout(context.Background(), hintTimeout)
	defer cancel()

	payload := anthropicMessageRequest{
		Model:     chapterModel,
		MaxTokens: 80,
		System:    "You are a literacy tutor. Return one sentence only.",
		Messages: []anthropicRequestPrompt{
			{
				Role:    "user",
				Content: fmt.Sprintf("Give a one-sentence hint for spelling: %s. Wrong attempt: %s.", word, wrong),
			},
		},
	}

	text, err := client.requestAnthropicText(ctx, payload)
	if err != nil {
		return fallbackSyllableHint(word)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fallbackSyllableHint(word)
	}
	return text
}

func sanitizeHintField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > maxHintFieldLength {
		value = value[:maxHintFieldLength]
	}
	return value
}

func fallbackSyllableHint(word string) string {
	runes := []rune(word)
	if len(runes) <= 3 {
		return "Try sounding it out slowly: " + string(runes)
	}

	segments := make([]string, 0, 4)
	for i := 0; i < len(runes); i += 3 {
		end := i + 3
		if end > len(runes) {
			end = len(runes)
		}
		segments = append(segments, string(runes[i:end]))
	}

	hint := "Try this sound split: " + strings.Join(segments, "-")
	if len(hint) > maxFallbackHintSize {
		return hint[:maxFallbackHintSize]
	}
	return hint
}
