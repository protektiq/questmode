package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxLearnerIDLength = 128
	weekDaysWindow     = 7
)

type ProgressHandler struct {
	DB *pgxpool.Pool
}

type progressSessionDay struct {
	Date              string `json:"date"`
	TasksCompleted    int    `json:"tasks_completed"`
	EngagementSeconds int    `json:"engagement_seconds"`
}

type progressSpellingWord struct {
	Word       string `json:"word"`
	Mastered   bool   `json:"mastered"`
	LastSeenAt string `json:"last_seen_at,omitempty"`
}

type progressSpellingData struct {
	Total    int                    `json:"total"`
	Mastered int                    `json:"mastered"`
	Words    []progressSpellingWord `json:"words"`
}

type progressMathByType struct {
	ProblemType string  `json:"problem_type"`
	Attempts    int     `json:"attempts"`
	CorrectRate float64 `json:"correct_rate"`
}

type progressMathData struct {
	TotalAttempts int                  `json:"total_attempts"`
	CorrectRate   float64              `json:"correct_rate"`
	ByProblemType []progressMathByType `json:"by_problem_type"`
}

type progressQuestArc struct {
	ArcID             string `json:"arc_id"`
	Title             string `json:"title"`
	Status            string `json:"status"`
	ChaptersCompleted int    `json:"chapters_completed"`
}

type progressBadge struct {
	BadgeCode string `json:"badge_code"`
	EarnedAt  string `json:"earned_at"`
}

type progressBrainCheck struct {
	RecordedAt string `json:"recorded_at"`
	Response   string `json:"response"`
	Emoji      string `json:"emoji"`
}

type ProgressData struct {
	LearnerID        string               `json:"learner_id"`
	Sessions         []progressSessionDay `json:"sessions"`
	Spelling         progressSpellingData `json:"spelling"`
	Math             progressMathData     `json:"math"`
	QuestArcs        []progressQuestArc   `json:"quest_arcs"`
	QuestBadges      []progressBadge      `json:"quest_badges"`
	WeeklyWordsTyped int                  `json:"weekly_words_typed"`
	BrainChecks      []progressBrainCheck `json:"brain_checks"`
}

type rawBrainCheckEvent struct {
	Response   string `json:"response"`
	RecordedAt string `json:"recordedAt"`
}

func NewProgressHandler(db *pgxpool.Pool) *ProgressHandler {
	return &ProgressHandler{DB: db}
}

func (h *ProgressHandler) GetProgress(c *gin.Context) {
	learnerID := strings.TrimSpace(c.Query("learner_id"))
	if learnerID == "" || len(learnerID) > maxLearnerIDLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id is required and must be <= 128 chars"})
		return
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "learner_id must be a valid uuid"})
		return
	}

	ctx := c.Request.Context()
	data := ProgressData{
		LearnerID: learnerID,
		Sessions:  make([]progressSessionDay, 0, weekDaysWindow),
		Spelling: progressSpellingData{
			Words: make([]progressSpellingWord, 0),
		},
		Math: progressMathData{
			ByProblemType: make([]progressMathByType, 0),
		},
		QuestArcs:   make([]progressQuestArc, 0),
		QuestBadges: make([]progressBadge, 0),
		BrainChecks: make([]progressBrainCheck, 0),
	}

	if err := h.loadSessions(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load session progress"})
		return
	}
	if err := h.loadSpelling(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load spelling progress"})
		return
	}
	if err := h.loadMath(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load math progress"})
		return
	}
	if err := h.loadArcs(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load quest arcs"})
		return
	}
	if err := h.loadBadges(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load quest badges"})
		return
	}
	if err := h.loadWeeklyWords(ctx, learnerID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load weekly words typed"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *ProgressHandler) loadSessions(ctx context.Context, learnerID string, data *ProgressData) error {
	rows, err := h.DB.Query(
		ctx,
		`SELECT
			to_char(day::date, 'YYYY-MM-DD') AS date,
			COALESCE(SUM(tasks_completed), 0) AS tasks_completed,
			COALESCE(SUM(engagement_seconds), 0) AS engagement_seconds,
			COALESCE(
				jsonb_agg(brain_checks_json) FILTER (WHERE brain_checks_json <> '[]'::jsonb),
				'[]'::jsonb
			) AS brain_checks_all
		FROM (
			SELECT generate_series(
				date_trunc('day', now()) - INTERVAL '6 days',
				date_trunc('day', now()),
				INTERVAL '1 day'
			) AS day
		) days
		LEFT JOIN quest_sessions qs
			ON qs.learner_id = $1::uuid
			AND date_trunc('day', qs.started_at) = days.day
		GROUP BY day
		ORDER BY day`,
		learnerID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var day progressSessionDay
		var brainChecksJSON []byte
		if scanErr := rows.Scan(&day.Date, &day.TasksCompleted, &day.EngagementSeconds, &brainChecksJSON); scanErr != nil {
			return scanErr
		}
		data.Sessions = append(data.Sessions, day)
		appendBrainChecks(brainChecksJSON, data)
	}
	return rows.Err()
}

func (h *ProgressHandler) loadSpelling(ctx context.Context, learnerID string, data *ProgressData) error {
	if err := h.DB.QueryRow(
		ctx,
		`SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE mastered_at IS NOT NULL) AS mastered
		FROM spelling_mastery
		WHERE learner_id = $1::uuid`,
		learnerID,
	).Scan(&data.Spelling.Total, &data.Spelling.Mastered); err != nil {
		return err
	}

	rows, err := h.DB.Query(
		ctx,
		`SELECT
			word,
			(mastered_at IS NOT NULL) AS mastered,
			COALESCE(to_char(last_seen_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM spelling_mastery
		WHERE learner_id = $1::uuid
		ORDER BY word ASC`,
		learnerID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var word progressSpellingWord
		if scanErr := rows.Scan(&word.Word, &word.Mastered, &word.LastSeenAt); scanErr != nil {
			return scanErr
		}
		data.Spelling.Words = append(data.Spelling.Words, word)
	}
	return rows.Err()
}

func (h *ProgressHandler) loadMath(ctx context.Context, learnerID string, data *ProgressData) error {
	if err := h.DB.QueryRow(
		ctx,
		`SELECT
			COUNT(*) AS total_attempts,
			COALESCE(AVG(CASE WHEN correct THEN 1.0 ELSE 0.0 END), 0.0) AS correct_rate
		FROM math_attempts
		WHERE learner_id = $1::uuid`,
		learnerID,
	).Scan(&data.Math.TotalAttempts, &data.Math.CorrectRate); err != nil {
		return err
	}

	rows, err := h.DB.Query(
		ctx,
		`SELECT
			COALESCE(NULLIF(problem_type, ''), 'general') AS problem_type,
			COUNT(*) AS attempts,
			COALESCE(AVG(CASE WHEN correct THEN 1.0 ELSE 0.0 END), 0.0) AS correct_rate
		FROM math_attempts
		WHERE learner_id = $1::uuid
		GROUP BY COALESCE(NULLIF(problem_type, ''), 'general')
		ORDER BY problem_type`,
		learnerID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var grouped progressMathByType
		if scanErr := rows.Scan(&grouped.ProblemType, &grouped.Attempts, &grouped.CorrectRate); scanErr != nil {
			return scanErr
		}
		data.Math.ByProblemType = append(data.Math.ByProblemType, grouped)
	}
	return rows.Err()
}

func (h *ProgressHandler) loadArcs(ctx context.Context, learnerID string, data *ProgressData) error {
	rows, err := h.DB.Query(
		ctx,
		`SELECT
			qa.id::text,
			qa.title,
			qa.status,
			COALESCE(MAX(qc.chapter_index), 0) AS chapters_completed
		FROM quest_arcs qa
		LEFT JOIN quest_chapters qc ON qc.arc_id = qa.id
		WHERE qa.learner_id = $1::uuid
		GROUP BY qa.id, qa.title, qa.status
		ORDER BY qa.created_at DESC`,
		learnerID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var arc progressQuestArc
		if scanErr := rows.Scan(&arc.ArcID, &arc.Title, &arc.Status, &arc.ChaptersCompleted); scanErr != nil {
			return scanErr
		}
		data.QuestArcs = append(data.QuestArcs, arc)
	}
	return rows.Err()
}

func (h *ProgressHandler) loadBadges(ctx context.Context, learnerID string, data *ProgressData) error {
	rows, err := h.DB.Query(
		ctx,
		`SELECT genre AS badge_code, to_char(earned_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM quest_badges
		WHERE learner_id = $1::uuid
		ORDER BY earned_at DESC`,
		learnerID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var badge progressBadge
		if scanErr := rows.Scan(&badge.BadgeCode, &badge.EarnedAt); scanErr != nil {
			return scanErr
		}
		data.QuestBadges = append(data.QuestBadges, badge)
	}
	return rows.Err()
}

func (h *ProgressHandler) loadWeeklyWords(ctx context.Context, learnerID string, data *ProgressData) error {
	err := h.DB.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(word_count), 0)
		FROM writing_logs
		WHERE learner_id = $1::uuid
		  AND logged_at >= date_trunc('week', now())
		  AND logged_at < date_trunc('week', now()) + INTERVAL '7 days'`,
		learnerID,
	).Scan(&data.WeeklyWordsTyped)
	if errors.Is(err, pgx.ErrNoRows) {
		data.WeeklyWordsTyped = 0
		return nil
	}
	return err
}

func appendBrainChecks(rawJSON []byte, data *ProgressData) {
	if len(rawJSON) == 0 {
		return
	}

	var nested []json.RawMessage
	if err := json.Unmarshal(rawJSON, &nested); err != nil {
		return
	}

	for _, chunk := range nested {
		var events []rawBrainCheckEvent
		if err := json.Unmarshal(chunk, &events); err != nil {
			continue
		}
		for _, event := range events {
			response := strings.TrimSpace(event.Response)
			if response == "" {
				continue
			}
			data.BrainChecks = append(data.BrainChecks, progressBrainCheck{
				Response:   response,
				RecordedAt: normalizeTimeString(event.RecordedAt),
				Emoji:      brainCheckEmoji(response),
			})
		}
	}
}

func normalizeTimeString(rawValue string) string {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.UTC().Format(time.RFC3339)
}

func brainCheckEmoji(response string) string {
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "frustrated", "hard", "sad", "angry":
		return "😟"
	case "ok", "fine", "normal", "neutral":
		return "😐"
	default:
		return "😊"
	}
}
