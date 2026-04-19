package claude

import (
	"strings"
	"testing"

	"questmode/backend/internal/story"
)

func TestBuildChapterPromptIncludesRequiredFields(t *testing.T) {
	state := story.StoryState{
		Genre:              "mystery",
		ChapterIndex:       5,
		ChoicesMade:        []string{"left", "climb"},
		ActiveSpellingWord: "adventure",
		LastChapterSummary: "The heroes found a hidden map under the old library stairs.",
	}

	prompt := buildChapterPrompt(state)

	requiredSubstrings := []string{
		"Genre: mystery",
		"Lexile target:",
		"Spelling focus word: adventure",
		"Math context to weave naturally:",
		"Prior chapter summary: The heroes found a hidden map under the old library stairs.",
		"Write exactly one chapter between 100-150 words.",
	}
	for _, expected := range requiredSubstrings {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing expected content: %q", expected)
		}
	}
}

func TestBuildChapterPromptHasExactOutputContractEnding(t *testing.T) {
	prompt := buildChapterPrompt(story.StoryState{})
	expectedEnding := "Output ONLY this JSON: {\"chapter\":\"...\",\"questions\":[{\"q\":\"...\",\"answer\":\"...\",\"hint\":\"...\"}]}"

	if !strings.HasSuffix(prompt, expectedEnding) {
		t.Fatalf("prompt does not end with required output contract.\nexpected suffix: %q\ngot: %q", expectedEnding, prompt)
	}
}
