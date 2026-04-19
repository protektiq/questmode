package claude

import (
	"fmt"
	"strings"

	"questmode/backend/internal/story"
)

const (
	maxPromptFieldLength = 500
)

func buildChapterPrompt(state story.StoryState) string {
	genre := sanitizePromptField(state.Genre, "adventure")
	lexile := deriveLexileBand(state)
	spellingWord := sanitizePromptField(state.ActiveSpellingWord, "curious")
	mathContext := deriveMathContext(state)
	priorSummary := sanitizePromptField(state.LastChapterSummary, "This is the opening chapter.")

	return fmt.Sprintf(
		"You are writing a child-friendly interactive story chapter.\n"+
			"Genre: %s\n"+
			"Lexile target: %s\n"+
			"Spelling focus word: %s\n"+
			"Math context to weave naturally: %s\n"+
			"Prior chapter summary: %s\n"+
			"Write exactly one chapter between 100-150 words.\n"+
			"Keep language age-appropriate and engaging.\n"+
			"Include 2 comprehension questions with concise answers and helpful hints.\n"+
			"Output ONLY this JSON: {\"chapter\":\"...\",\"questions\":[{\"q\":\"...\",\"answer\":\"...\",\"hint\":\"...\"}]}",
		genre,
		lexile,
		spellingWord,
		mathContext,
		priorSummary,
	)
}

func sanitizePromptField(value, fallback string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return fallback
	}
	if len(clean) > maxPromptFieldLength {
		return clean[:maxPromptFieldLength]
	}
	return clean
}

func deriveLexileBand(state story.StoryState) string {
	if state.ChapterIndex >= 8 {
		return "600L-700L"
	}
	if state.ChapterIndex >= 4 {
		return "500L-600L"
	}
	return "450L-550L"
}

func deriveMathContext(state story.StoryState) string {
	count := len(state.ChoicesMade)
	if count <= 0 {
		return "Use simple counting up to 10."
	}
	if count <= 3 {
		return "Use addition and subtraction within 20."
	}
	return "Use multiplication patterns with small numbers."
}
