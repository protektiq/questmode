package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"questmode/backend/internal/claude"
	"questmode/backend/internal/story"
)

const (
	maxGenreLength      = 64
	maxAnswerLength     = 500
	maxResponseLength   = 64
	minWritingTextChars = 20
	maxWritingTextChars = 20000
	maxQuestionAttempts = 2
	defaultArcLength    = 10
)

type StoryHandler struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Claude *claude.ClaudeClient
}

type StartArcRequest struct {
	Genre string `json:"genre"`
}

type AnswerRequest struct {
	ChapterID     string `json:"chapter_id"`
	QuestionIndex int    `json:"question_index"`
	Answer        string `json:"answer"`
}

type BrainCheckRequest struct {
	LearnerID string `json:"learner_id"`
	Response  string `json:"response"`
}

type WritingLogRequest struct {
	LearnerID string `json:"learner_id"`
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

type chapterQuestionPublic struct {
	Q    string `json:"q"`
	Hint string `json:"hint"`
}

type questBadge struct {
	ID        string `json:"id"`
	ArcID     string `json:"arc_id"`
	LearnerID string `json:"learner_id"`
	Genre     string `json:"genre"`
	EarnedAt  string `json:"earned_at"`
}

type chapterResponse struct {
	ChapterID   string                  `json:"chapter_id"`
	Content     string                  `json:"content"`
	Questions   []chapterQuestionPublic `json:"questions"`
	ArcComplete bool                    `json:"arc_complete"`
	Badge       *questBadge             `json:"badge"`
}

type answerMeta struct {
	ChapterID            string      `json:"chapter_id"`
	CurrentQuestionIndex int         `json:"current_question_index"`
	HintUsageByQuestion  map[int]int `json:"hint_usage_by_question"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

func NewStoryHandler(db *pgxpool.Pool, rdb *redis.Client, claudeClient *claude.ClaudeClient) *StoryHandler {
	return &StoryHandler{
		DB:     db,
		Redis:  rdb,
		Claude: claudeClient,
	}
}

func (h *StoryHandler) RegisterStoryRoutes(group *gin.RouterGroup) {
	group.POST("/arc/start", h.StartArc)
	group.GET("/chapter", h.GetChapter)
	group.POST("/answer", h.SubmitAnswer)
	group.POST("/brain-check", h.BrainCheck)
	group.POST("/writing-log", h.LogWriting)
	group.GET("/status", h.GetStatus)
}

func (h *StoryHandler) StartArc(c *gin.Context) {
	var req StartArcRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.Genre = strings.TrimSpace(req.Genre)
	if req.Genre == "" || len(req.Genre) > maxGenreLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "genre is required and must be <= 64 characters"})
		return
	}

	ctx := c.Request.Context()
	var learnerID string
	if err := h.DB.QueryRow(ctx, `SELECT id::text FROM learner_profiles LIMIT 1`).Scan(&learnerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no learner profile found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load learner profile"})
		return
	}

	var arcID string
	if err := h.DB.QueryRow(
		ctx,
		`INSERT INTO quest_arcs (learner_id, genre, status, title, sort_order) VALUES ($1::uuid, $2, 'active', 'Dynamic Arc', 0) RETURNING id::text`,
		learnerID,
		req.Genre,
	).Scan(&arcID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create quest arc"})
		return
	}

	defaultState := story.StoryState{
		LearnerID:    learnerID,
		ArcID:        arcID,
		Genre:        req.Genre,
		ChapterIndex: 0,
		ChoicesMade:  []string{},
		Phase:        story.PhaseChapter,
	}

	var sessionID string
	generatedSessionID := uuid.NewString()
	if err := h.DB.QueryRow(
		ctx,
		`INSERT INTO quest_sessions (learner_id, arc_id, started_at, status, session_id) VALUES ($1::uuid, $2::uuid, now(), 'active', $3) RETURNING session_id`,
		learnerID,
		arcID,
		generatedSessionID,
	).Scan(&sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create quest session"})
		return
	}

	defaultState.SessionID = sessionID
	if err := story.Save(defaultState, h.Redis); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize story state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"arc_id":     arcID,
		"genre":      req.Genre,
		"learner_id": learnerID,
		"session_id": sessionID,
	})
}

func (h *StoryHandler) GetChapter(c *gin.Context) {
	learnerID := strings.TrimSpace(c.Query("learner_id"))
	if learnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id is required"})
		return
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id must be a valid uuid"})
		return
	}

	state, err := story.Load(learnerID, h.Redis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load story state"})
		return
	}
	if state.LearnerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "story state not found for learner"})
		return
	}

	chapterData := claude.FALLBACK_CHAPTER
	if h.Claude != nil {
		generated, genErr := h.Claude.GenerateChapter(c.Request.Context(), state)
		if genErr == nil {
			chapterData = generated
		}
	}

	questionsJSON, err := json.Marshal(chapterData.Questions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize chapter questions"})
		return
	}

	nextChapterIndex := state.ChapterIndex + 1
	arcLength := getArcLength()
	var chapterID string
	err = h.DB.QueryRow(
		c.Request.Context(),
		`INSERT INTO quest_chapters (arc_id, chapter_index, content_text, questions_json, title, sort_order) VALUES ($1::uuid, $2, $3, $4::jsonb, $5, $6) RETURNING id::text`,
		state.ArcID,
		nextChapterIndex,
		chapterData.Chapter,
		questionsJSON,
		"Dynamic Chapter",
		nextChapterIndex,
	).Scan(&chapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist chapter"})
		return
	}

	state.ChapterIndex = nextChapterIndex
	state.LastChapterSummary = chapterData.Chapter
	state.Phase = story.PhaseQuestion

	meta := answerMeta{
		ChapterID:            chapterID,
		CurrentQuestionIndex: 0,
		HintUsageByQuestion:  map[int]int{},
		UpdatedAt:            time.Now().UTC(),
	}
	if err := h.saveAnswerMeta(c.Request.Context(), learnerID, meta); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare question state"})
		return
	}

	arcComplete := state.ChapterIndex >= arcLength
	var earnedBadge *questBadge
	if arcComplete {
		earnedBadge, err = h.completeArc(c.Request.Context(), state)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete arc"})
			return
		}
		if err := story.Delete(learnerID, h.Redis); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear story state"})
			return
		}
		if err := h.Redis.Del(c.Request.Context(), h.answerMetaKey(learnerID)).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear answer state"})
			return
		}
	} else {
		if err := story.Save(state, h.Redis); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save story state"})
			return
		}
	}

	publicQuestions := make([]chapterQuestionPublic, 0, len(chapterData.Questions))
	for _, q := range chapterData.Questions {
		publicQuestions = append(publicQuestions, chapterQuestionPublic{
			Q:    q.Q,
			Hint: q.Hint,
		})
	}

	c.JSON(http.StatusOK, chapterResponse{
		ChapterID:   chapterID,
		Content:     chapterData.Chapter,
		Questions:   publicQuestions,
		ArcComplete: arcComplete,
		Badge:       earnedBadge,
	})
}

func (h *StoryHandler) SubmitAnswer(c *gin.Context) {
	var req AnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.ChapterID = strings.TrimSpace(req.ChapterID)
	req.Answer = strings.TrimSpace(req.Answer)
	if req.ChapterID == "" || req.Answer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chapter_id and answer are required"})
		return
	}
	if len(req.Answer) > maxAnswerLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "answer exceeds max length"})
		return
	}
	if _, err := uuid.Parse(req.ChapterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chapter_id must be a valid uuid"})
		return
	}
	if req.QuestionIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question_index must be non-negative"})
		return
	}

	ctx := c.Request.Context()
	var rawQuestions []byte
	var arcID string
	if err := h.DB.QueryRow(
		ctx,
		`SELECT questions_json, arc_id::text FROM quest_chapters WHERE id = $1::uuid`,
		req.ChapterID,
	).Scan(&rawQuestions, &arcID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load chapter"})
		return
	}

	var learnerID string
	if err := h.DB.QueryRow(
		ctx,
		`SELECT learner_id::text FROM quest_sessions WHERE arc_id = $1::uuid ORDER BY started_at DESC LIMIT 1`,
		arcID,
	).Scan(&learnerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve learner session"})
		return
	}

	var questions []claude.Question
	if err := json.Unmarshal(rawQuestions, &questions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid chapter questions"})
		return
	}
	if req.QuestionIndex >= len(questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question_index out of range"})
		return
	}

	meta, _ := h.loadAnswerMeta(ctx, learnerID)
	if meta.HintUsageByQuestion == nil {
		meta.HintUsageByQuestion = map[int]int{}
	}
	meta.ChapterID = req.ChapterID
	meta.UpdatedAt = time.Now().UTC()

	current := questions[req.QuestionIndex]
	normalizedInput := strings.ToLower(strings.TrimSpace(req.Answer))
	normalizedAnswer := strings.ToLower(strings.TrimSpace(current.Answer))

	state, err := story.Load(learnerID, h.Redis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load story state"})
		return
	}
	if state.LearnerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "story state not found for learner"})
		return
	}

	correct := normalizedInput == normalizedAnswer
	revealed := false
	hint := ""

	if correct {
		meta.HintUsageByQuestion[req.QuestionIndex] = 0
	} else {
		meta.HintUsageByQuestion[req.QuestionIndex] = meta.HintUsageByQuestion[req.QuestionIndex] + 1
		if meta.HintUsageByQuestion[req.QuestionIndex] < maxQuestionAttempts {
			hint = current.Hint
			if err := h.saveAnswerMeta(ctx, learnerID, meta); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save answer state"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"correct":    false,
				"hint":       hint,
				"revealed":   false,
				"next_phase": state.Phase,
			})
			return
		}
		revealed = true
		hint = current.Answer
	}

	meta.CurrentQuestionIndex = req.QuestionIndex + 1
	if meta.CurrentQuestionIndex >= len(questions) {
		state.Phase = story.PhaseSpelling
	} else {
		state.Phase = story.PhaseQuestion
	}

	if err := story.Save(state, h.Redis); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save story phase"})
		return
	}
	if err := h.saveAnswerMeta(ctx, learnerID, meta); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist answer progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"correct":    correct,
		"hint":       hint,
		"revealed":   revealed,
		"next_phase": state.Phase,
	})
}

func (h *StoryHandler) BrainCheck(c *gin.Context) {
	var req BrainCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.LearnerID = strings.TrimSpace(req.LearnerID)
	req.Response = strings.TrimSpace(req.Response)
	if _, err := uuid.Parse(req.LearnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id must be a valid uuid"})
		return
	}
	if req.Response == "" || len(req.Response) > maxResponseLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "response is required and must be <= 64 chars"})
		return
	}

	state, err := story.Load(req.LearnerID, h.Redis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load story state"})
		return
	}
	if state.LearnerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "story state not found for learner"})
		return
	}

	story.ApplyBrainCheck(&state, req.Response)
	if err := story.Save(state, h.Redis); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save story state"})
		return
	}

	event := map[string]string{
		"response":   req.Response,
		"recordedAt": time.Now().UTC().Format(time.RFC3339),
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize brain check event"})
		return
	}

	cmdTag, err := h.DB.Exec(
		c.Request.Context(),
		`UPDATE quest_sessions
		 SET brain_checks_json = COALESCE(brain_checks_json, '[]'::jsonb) || $1::jsonb
		 WHERE id = (
			SELECT id FROM quest_sessions
			WHERE learner_id = $2::uuid
			ORDER BY started_at DESC
			LIMIT 1
		 )`,
		"["+string(eventJSON)+"]",
		req.LearnerID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to append brain check"})
		return
	}
	if cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active session found for learner"})
		return
	}

	nextDifficulty := "normal"
	if state.BrainCheckFrustrated {
		nextDifficulty = "easier"
	}

	c.JSON(http.StatusOK, gin.H{
		"acknowledged":    true,
		"next_difficulty": nextDifficulty,
	})
}

func (h *StoryHandler) GetStatus(c *gin.Context) {
	learnerID := strings.TrimSpace(c.Query("learner_id"))
	if learnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id is required"})
		return
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id must be a valid uuid"})
		return
	}

	state, err := story.Load(learnerID, h.Redis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load story state"})
		return
	}
	if state.LearnerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "story state not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"learner_id":             state.LearnerID,
		"session_id":             state.SessionID,
		"arc_id":                 state.ArcID,
		"genre":                  state.Genre,
		"chapter_index":          state.ChapterIndex,
		"active_spelling_word":   state.ActiveSpellingWord,
		"brain_check_frustrated": state.BrainCheckFrustrated,
		"engagement_seconds":     state.EngagementSeconds,
		"break_offered":          state.BreakOffered,
		"last_chapter_summary":   state.LastChapterSummary,
		"phase":                  state.Phase,
	})
}

func (h *StoryHandler) LogWriting(c *gin.Context) {
	var req WritingLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.LearnerID = strings.TrimSpace(req.LearnerID)
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Text = strings.TrimSpace(req.Text)

	if req.LearnerID == "" || req.SessionID == "" || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id, session_id, and text are required"})
		return
	}
	if len(req.Text) < minWritingTextChars || len(req.Text) > maxWritingTextChars {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text must be between 20 and 20000 characters"})
		return
	}
	if _, err := uuid.Parse(req.LearnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id must be a valid uuid"})
		return
	}
	if _, err := uuid.Parse(req.SessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id must be a valid uuid"})
		return
	}

	wordCount := len(strings.Fields(req.Text))
	if wordCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text must contain at least one word"})
		return
	}

	cmdTag, err := h.DB.Exec(
		c.Request.Context(),
		`INSERT INTO writing_logs (learner_id, session_id, text, word_count, logged_at)
		 SELECT
		   $1::uuid,
		   qs.id,
		   $3,
		   $4,
		   now()
		 FROM quest_sessions qs
		 WHERE qs.learner_id = $1::uuid
		   AND (qs.id = $2::uuid OR qs.session_id = $2::text)
		 ORDER BY qs.started_at DESC
		 LIMIT 1`,
		req.LearnerID,
		req.SessionID,
		req.Text,
		wordCount,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save writing log"})
		return
	}
	if cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found for learner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"saved":      true,
		"word_count": wordCount,
	})
}

func (h *StoryHandler) answerMetaKey(learnerID string) string {
	return "story:answer-meta:" + learnerID
}

func (h *StoryHandler) loadAnswerMeta(ctx context.Context, learnerID string) (answerMeta, error) {
	raw, err := h.Redis.Get(ctx, h.answerMetaKey(learnerID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return answerMeta{}, nil
		}
		return answerMeta{}, err
	}

	var meta answerMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return answerMeta{}, err
	}
	return meta, nil
}

func (h *StoryHandler) saveAnswerMeta(ctx context.Context, learnerID string, meta answerMeta) error {
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return h.Redis.Set(ctx, h.answerMetaKey(learnerID), payload, 24*time.Hour).Err()
}

func getArcLength() int {
	raw := strings.TrimSpace(os.Getenv("ARC_LENGTH"))
	if raw == "" {
		return defaultArcLength
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultArcLength
	}
	return parsed
}

func (h *StoryHandler) completeArc(ctx context.Context, state story.StoryState) (*questBadge, error) {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`UPDATE quest_arcs
		 SET status = 'completed', completed_at = now()
		 WHERE id = $1::uuid`,
		state.ArcID,
	); err != nil {
		return nil, err
	}

	badge := &questBadge{}
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO quest_badges (arc_id, learner_id, genre, earned_at)
		 VALUES ($1::uuid, $2::uuid, $3, now())
		 RETURNING id::text, arc_id::text, learner_id::text, genre, to_char(earned_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		state.ArcID,
		state.LearnerID,
		state.Genre,
	).Scan(&badge.ID, &badge.ArcID, &badge.LearnerID, &badge.Genre, &badge.EarnedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return badge, nil
}
