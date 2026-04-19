package claude

import "testing"

func TestParseChapterResponsePlainJSON(t *testing.T) {
	raw := `{"chapter":"A bright trail led Mia to the old oak where clues waited.","questions":[{"q":"Where did Mia go?","answer":"To the old oak.","hint":"Look for the place named in the chapter."}]}`

	got, err := parseChapterResponse(raw)
	if err != nil {
		t.Fatalf("parseChapterResponse returned error: %v", err)
	}
	if got.Chapter == "" {
		t.Fatal("chapter should not be empty")
	}
	if len(got.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(got.Questions))
	}
}

func TestParseChapterResponseWrappedJSON(t *testing.T) {
	raw := `Here is your story:
{"chapter":"Luca counted fireflies and saw seven near the pond.","questions":[{"q":"What did Luca count?","answer":"Fireflies.","hint":"Look at what was near the pond."}]}
Thanks!`

	got, err := parseChapterResponse(raw)
	if err != nil {
		t.Fatalf("parseChapterResponse returned error: %v", err)
	}
	if got.Questions[0].Answer != "Fireflies." {
		t.Fatalf("unexpected parsed answer: %q", got.Questions[0].Answer)
	}
}

func TestParseChapterResponseMalformedJSONFails(t *testing.T) {
	raw := `No JSON here`

	_, err := parseChapterResponse(raw)
	if err == nil {
		t.Fatal("expected parseChapterResponse to fail for malformed content")
	}
}
