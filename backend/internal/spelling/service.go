package spelling

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	masteryThreshold = 3
	revealThreshold  = 2
	maxWordLength    = 64
	maxAnswerLength  = 256
	maxSeedWords     = 500
)

type SpellingWord struct {
	ID           string     `json:"id"`
	LearnerID    string     `json:"learner_id"`
	Word         string     `json:"word"`
	CorrectCount int        `json:"correct_count"`
	HintCount    int        `json:"hint_count"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	MasteredAt   *time.Time `json:"mastered_at,omitempty"`
}

type SpellingResult struct {
	Correct   bool   `json:"correct"`
	Mastered  bool   `json:"mastered"`
	Hint      string `json:"hint,omitempty"`
	HintsUsed int    `json:"hints_used,omitempty"`
	Reveal    bool   `json:"reveal"`
}

func GetActiveWord(ctx context.Context, learnerID string, db *pgxpool.Pool) (SpellingWord, error) {
	learnerID = strings.TrimSpace(learnerID)
	if learnerID == "" {
		return SpellingWord{}, errors.New("learner_id is required")
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		return SpellingWord{}, errors.New("learner_id must be a valid uuid")
	}
	if db == nil {
		return SpellingWord{}, errors.New("db is required")
	}

	var word SpellingWord
	var lastSeen sql.NullTime
	var mastered sql.NullTime
	err := db.QueryRow(ctx, `
		SELECT id::text, learner_id::text, word, correct_count, hint_count, last_seen_at, mastered_at
		FROM spelling_mastery
		WHERE learner_id = $1::uuid AND mastered_at IS NULL
		ORDER BY last_seen_at ASC NULLS FIRST
		LIMIT 1
	`, learnerID).Scan(
		&word.ID,
		&word.LearnerID,
		&word.Word,
		&word.CorrectCount,
		&word.HintCount,
		&lastSeen,
		&mastered,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SpellingWord{}, nil
		}
		return SpellingWord{}, err
	}

	if lastSeen.Valid {
		word.LastSeenAt = &lastSeen.Time
	}
	if mastered.Valid {
		word.MasteredAt = &mastered.Time
	}
	return word, nil
}

func CheckAnswer(ctx context.Context, learnerID, wordID, answer string, db *pgxpool.Pool) (SpellingResult, error) {
	learnerID = strings.TrimSpace(learnerID)
	wordID = strings.TrimSpace(wordID)
	answer = strings.TrimSpace(answer)

	if learnerID == "" || wordID == "" || answer == "" {
		return SpellingResult{}, errors.New("learner_id, word_id, and answer are required")
	}
	if len(answer) > maxAnswerLength {
		return SpellingResult{}, fmt.Errorf("answer exceeds max length of %d", maxAnswerLength)
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		return SpellingResult{}, errors.New("learner_id must be a valid uuid")
	}
	if _, err := uuid.Parse(wordID); err != nil {
		return SpellingResult{}, errors.New("word_id must be a valid uuid")
	}
	if db == nil {
		return SpellingResult{}, errors.New("db is required")
	}

	var storedWord string
	err := db.QueryRow(ctx, `
		SELECT word
		FROM spelling_mastery
		WHERE id = $1::uuid AND learner_id = $2::uuid
	`, wordID, learnerID).Scan(&storedWord)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SpellingResult{}, errors.New("word not found for learner")
		}
		return SpellingResult{}, err
	}

	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	normalizedWord := strings.ToLower(strings.TrimSpace(storedWord))
	if normalizedAnswer == normalizedWord {
		var correctCount int
		var mastered bool
		if err := db.QueryRow(ctx, `
			UPDATE spelling_mastery
			SET
				correct_count = correct_count + 1,
				last_seen_at = NOW(),
				mastered_at = CASE
					WHEN correct_count + 1 >= $3 THEN COALESCE(mastered_at, NOW())
					ELSE mastered_at
				END
			WHERE id = $1::uuid AND learner_id = $2::uuid
			RETURNING correct_count, mastered_at IS NOT NULL
		`, wordID, learnerID, masteryThreshold).Scan(&correctCount, &mastered); err != nil {
			return SpellingResult{}, err
		}
		return SpellingResult{
			Correct:  true,
			Mastered: mastered || correctCount >= masteryThreshold,
		}, nil
	}

	var hintsUsed int
	if err := db.QueryRow(ctx, `
		UPDATE spelling_mastery
		SET hint_count = hint_count + 1
		WHERE id = $1::uuid AND learner_id = $2::uuid
		RETURNING hint_count
	`, wordID, learnerID).Scan(&hintsUsed); err != nil {
		return SpellingResult{}, err
	}

	return SpellingResult{
		Correct:   false,
		Hint:      SyllableSplit(storedWord),
		HintsUsed: hintsUsed,
		Reveal:    hintsUsed >= revealThreshold,
	}, nil
}

func SyllableSplit(word string) string {
	trimmed := strings.TrimSpace(word)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= 2 {
		return trimmed
	}

	bestIdx := -1
	bestScore := -1 << 30
	for i := 1; i < len(runes); i++ {
		left := runes[:i]
		right := runes[i:]
		if len(left) == 0 || len(right) == 0 {
			continue
		}
		if !containsVowel(right) {
			continue
		}

		score := 0
		lastLeft := left[len(left)-1]
		firstRight := right[0]
		if isVowel(lastLeft) && !isVowel(firstRight) {
			score += 4
		}
		if !isVowel(lastLeft) && isVowel(firstRight) {
			score += 3
		}
		if containsVowel(left) {
			score += 1
		}

		edgeDistance := i
		if len(runes)-i < edgeDistance {
			edgeDistance = len(runes) - i
		}
		score += minInt(edgeDistance, 3)

		rightLower := strings.ToLower(string(right))
		if strings.HasPrefix(rightLower, "cause") ||
			strings.HasPrefix(rightLower, "tion") ||
			strings.HasPrefix(rightLower, "sure") ||
			strings.HasPrefix(rightLower, "ing") {
			score += 2
		}

		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	if bestIdx <= 0 || bestIdx >= len(runes) {
		mid := len(runes) / 2
		return string(runes[:mid]) + " – " + string(runes[mid:])
	}
	return string(runes[:bestIdx]) + " – " + string(runes[bestIdx:])
}

func SeedWordList(ctx context.Context, learnerID string, words []string, db *pgxpool.Pool) error {
	learnerID = strings.TrimSpace(learnerID)
	if learnerID == "" {
		return errors.New("learner_id is required")
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		return errors.New("learner_id must be a valid uuid")
	}
	if db == nil {
		return errors.New("db is required")
	}
	if len(words) == 0 {
		return nil
	}
	if len(words) > maxSeedWords {
		return fmt.Errorf("word list exceeds max size of %d", maxSeedWords)
	}

	seen := make(map[string]struct{}, len(words))
	batch := &pgx.Batch{}
	for _, rawWord := range words {
		w := strings.TrimSpace(rawWord)
		if w == "" {
			continue
		}
		if len(w) > maxWordLength {
			return fmt.Errorf("word exceeds max length of %d", maxWordLength)
		}
		key := strings.ToLower(w)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		batch.Queue(`
			INSERT INTO spelling_mastery (learner_id, word)
			VALUES ($1::uuid, $2)
			ON CONFLICT (learner_id, word) DO NOTHING
		`, learnerID, w)
	}

	if batch.Len() == 0 {
		return nil
	}

	results := db.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func isVowel(r rune) bool {
	l := unicode.ToLower(r)
	return l == 'a' || l == 'e' || l == 'i' || l == 'o' || l == 'u' || l == 'y'
}

func containsVowel(rs []rune) bool {
	for _, r := range rs {
		if isVowel(r) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
