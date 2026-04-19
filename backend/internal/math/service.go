package math

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxGenreLength      = 64
	maxProblemIDLength  = 64
	minAnswerValue      = -1000000
	maxAnswerValue      = 1000000
	firstHintThreshold  = 1
	secondHintThreshold = 2
)

type MathResult struct {
	Correct      bool   `json:"correct"`
	Hint         string `json:"hint,omitempty"`
	Reveal       bool   `json:"reveal"`
	RevealAnswer string `json:"reveal_answer,omitempty"`
}

type attemptStore interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func GetProblem(difficulty Difficulty, genre string, frustrated bool) MathProblem {
	normalizedDifficulty := normalizeDifficulty(difficulty)
	normalizedGenre := normalizeGenre(genre)
	if frustrated {
		normalizedDifficulty = Downgrade(normalizedDifficulty)
	}

	filtered := make([]MathProblem, 0, len(ProblemBank))
	for _, problem := range ProblemBank {
		if problem.Difficulty != normalizedDifficulty {
			continue
		}
		if normalizedGenre != "" && problem.Genre != normalizedGenre {
			continue
		}
		filtered = append(filtered, problem)
	}

	if len(filtered) == 0 {
		for _, problem := range ProblemBank {
			if problem.Difficulty == normalizedDifficulty {
				filtered = append(filtered, problem)
			}
		}
	}

	if len(filtered) == 0 {
		filtered = ProblemBank
	}
	if len(filtered) == 0 {
		return MathProblem{}
	}

	idx := time.Now().Second() % len(filtered)
	return filtered[idx]
}

func CheckAnswer(
	ctx context.Context,
	learnerID, sessionID, problemID string,
	answer int,
	db *pgxpool.Pool,
) (MathResult, error) {
	learnerID = strings.TrimSpace(learnerID)
	sessionID = strings.TrimSpace(sessionID)
	problemID = strings.TrimSpace(problemID)

	if learnerID == "" || sessionID == "" || problemID == "" {
		return MathResult{}, errors.New("learner_id, session_id, and problem_id are required")
	}
	if len(problemID) > maxProblemIDLength {
		return MathResult{}, fmt.Errorf("problem_id exceeds max length of %d", maxProblemIDLength)
	}
	if answer < minAnswerValue || answer > maxAnswerValue {
		return MathResult{}, fmt.Errorf("answer must be between %d and %d", minAnswerValue, maxAnswerValue)
	}
	if _, err := uuid.Parse(learnerID); err != nil {
		return MathResult{}, errors.New("learner_id must be a valid uuid")
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return MathResult{}, errors.New("session_id must be a valid uuid")
	}
	if db == nil {
		return MathResult{}, errors.New("db is required")
	}

	problem, found := problemByID(problemID)
	if !found {
		return MathResult{}, errors.New("problem not found")
	}

	return checkAnswerWithStore(ctx, learnerID, problem, answer, db)
}

func checkAnswerWithStore(
	ctx context.Context,
	learnerID string,
	problem MathProblem,
	answer int,
	store attemptStore,
) (MathResult, error) {
	correct := answer == problem.Answer
	if _, err := store.Exec(ctx, `
		INSERT INTO math_attempts (learner_id, problem, correct)
		VALUES ($1::uuid, $2, $3)
	`, learnerID, problem.ID, correct); err != nil {
		return MathResult{}, err
	}

	if correct {
		return MathResult{
			Correct: true,
			Reveal:  false,
		}, nil
	}

	var wrongAttempts int
	if err := store.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM math_attempts
		WHERE learner_id = $1::uuid AND problem = $2 AND correct = FALSE
	`, learnerID, problem.ID).Scan(&wrongAttempts); err != nil {
		return MathResult{}, err
	}

	switch {
	case wrongAttempts >= 3:
		return MathResult{
			Correct:      false,
			Reveal:       true,
			RevealAnswer: fmt.Sprintf("%d", problem.Answer),
		}, nil
	case wrongAttempts >= secondHintThreshold:
		return MathResult{
			Correct: false,
			Hint:    problem.Hint2,
			Reveal:  false,
		}, nil
	case wrongAttempts >= firstHintThreshold:
		return MathResult{
			Correct: false,
			Hint:    problem.Hint1,
			Reveal:  false,
		}, nil
	default:
		return MathResult{
			Correct: false,
			Reveal:  false,
		}, nil
	}
}

func normalizeDifficulty(d Difficulty) Difficulty {
	switch Difficulty(strings.ToLower(strings.TrimSpace(string(d)))) {
	case DiffEasy:
		return DiffEasy
	case DiffMedium:
		return DiffMedium
	case DiffHard:
		return DiffHard
	default:
		return DiffEasy
	}
}

func normalizeGenre(genre string) string {
	normalized := strings.ToLower(strings.TrimSpace(genre))
	if normalized == "" {
		return ""
	}
	if len(normalized) > maxGenreLength {
		return ""
	}
	switch normalized {
	case "adventure", "mystery", "fantasy":
		return normalized
	default:
		return ""
	}
}

func problemByID(problemID string) (MathProblem, bool) {
	for _, problem := range ProblemBank {
		if problem.ID == problemID {
			return problem, true
		}
	}
	return MathProblem{}, false
}
