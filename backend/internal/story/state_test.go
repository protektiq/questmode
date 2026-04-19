package story

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStoryStateJSONRoundTrip(t *testing.T) {
	initial := StoryState{
		LearnerID:            "learner-123",
		SessionID:            "session-123",
		ArcID:                "arc-456",
		Genre:                "fantasy",
		ChapterIndex:         2,
		ChoicesMade:          []string{"left", "talk"},
		ActiveSpellingWord:   "lantern",
		BrainCheckFrustrated: true,
		EngagementSeconds:    312,
		BreakOffered:         true,
		LastChapterSummary:   "The hero found a map.",
		Phase:                PhaseQuestion,
	}

	payload, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded StoryState
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(initial, decoded) {
		t.Fatalf("story state mismatch after round-trip: got=%+v want=%+v", decoded, initial)
	}
}

func TestApplyBrainCheckFrustratedSetsFlag(t *testing.T) {
	state := StoryState{BrainCheckFrustrated: false}
	ApplyBrainCheck(&state, "frustrated")
	if !state.BrainCheckFrustrated {
		t.Fatalf("expected BrainCheckFrustrated to be true")
	}
}

func TestApplyBrainCheckGreatClearsFlag(t *testing.T) {
	state := StoryState{BrainCheckFrustrated: true}
	ApplyBrainCheck(&state, "great")
	if state.BrainCheckFrustrated {
		t.Fatalf("expected BrainCheckFrustrated to be false")
	}
}
