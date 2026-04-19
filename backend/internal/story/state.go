package story

const (
	PhaseChapter  = "chapter"
	PhaseQuestion = "question"
	PhaseSpelling = "spelling"
	PhaseMath     = "math"
)

type StoryState struct {
	LearnerID            string   `json:"learner_id"`
	SessionID            string   `json:"session_id"`
	ArcID                string   `json:"arc_id"`
	Genre                string   `json:"genre"`
	ChapterIndex         int      `json:"chapter_index"`
	ChoicesMade          []string `json:"choices_made"`
	ActiveSpellingWord   string   `json:"active_spelling_word"`
	BrainCheckFrustrated bool     `json:"brain_check_frustrated"`
	EngagementSeconds    int      `json:"engagement_seconds"`
	BreakOffered         bool     `json:"break_offered"`
	LastChapterSummary   string   `json:"last_chapter_summary"`
	Phase                string   `json:"phase"`
}
