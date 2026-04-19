package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	maxRawResponseLength = 20000
	maxChapterLength     = 5000
	maxQuestionCount     = 6
)

type Question struct {
	Q      string `json:"q"`
	Answer string `json:"answer"`
	Hint   string `json:"hint"`
}

type ChapterResponse struct {
	Chapter   string     `json:"chapter"`
	Questions []Question `json:"questions"`
}

func parseChapterResponse(raw string) (ChapterResponse, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ChapterResponse{}, errors.New("empty response body")
	}
	if len(raw) > maxRawResponseLength {
		return ChapterResponse{}, fmt.Errorf("response body exceeds max length of %d", maxRawResponseLength)
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return ChapterResponse{}, errors.New("no json object found in response")
	}

	candidate := raw[start : end+1]

	var parsed ChapterResponse
	if err := json.Unmarshal([]byte(candidate), &parsed); err != nil {
		return ChapterResponse{}, fmt.Errorf("unmarshal chapter response: %w", err)
	}

	parsed.Chapter = strings.TrimSpace(parsed.Chapter)
	if parsed.Chapter == "" {
		return ChapterResponse{}, errors.New("chapter is required")
	}
	if len(parsed.Chapter) > maxChapterLength {
		return ChapterResponse{}, fmt.Errorf("chapter exceeds max length of %d", maxChapterLength)
	}
	if len(parsed.Questions) == 0 || len(parsed.Questions) > maxQuestionCount {
		return ChapterResponse{}, fmt.Errorf("questions length must be between 1 and %d", maxQuestionCount)
	}

	for i := range parsed.Questions {
		parsed.Questions[i].Q = strings.TrimSpace(parsed.Questions[i].Q)
		parsed.Questions[i].Answer = strings.TrimSpace(parsed.Questions[i].Answer)
		parsed.Questions[i].Hint = strings.TrimSpace(parsed.Questions[i].Hint)
		if parsed.Questions[i].Q == "" || parsed.Questions[i].Answer == "" || parsed.Questions[i].Hint == "" {
			return ChapterResponse{}, fmt.Errorf("question at index %d is missing required fields", i)
		}
	}

	return parsed, nil
}
