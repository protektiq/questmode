package math

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	wrongAttempts int
	err           error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 1 {
		return errors.New("expected one scan destination")
	}
	ptr, ok := dest[0].(*int)
	if !ok {
		return errors.New("expected *int destination")
	}
	*ptr = r.wrongAttempts
	return nil
}

type fakeAttemptStore struct {
	execCorrectValues []bool
	wrongAttempts     int
}

func (s *fakeAttemptStore) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	if len(args) != 3 {
		return pgconn.CommandTag{}, errors.New("expected 3 args")
	}
	correct, ok := args[2].(bool)
	if !ok {
		return pgconn.CommandTag{}, errors.New("expected bool correct arg")
	}
	s.execCorrectValues = append(s.execCorrectValues, correct)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (s *fakeAttemptStore) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return fakeRow{wrongAttempts: s.wrongAttempts}
}

func TestProblemBankSize(t *testing.T) {
	if len(ProblemBank) < 30 {
		t.Fatalf("expected at least 30 math problems, got %d", len(ProblemBank))
	}
}

func TestGetProblemFrustratedDowngradesDifficulty(t *testing.T) {
	problem := GetProblem(DiffHard, "adventure", true)
	if problem.ID == "" {
		t.Fatal("expected a problem")
	}
	if problem.Difficulty != DiffMedium {
		t.Fatalf("expected downgraded difficulty %q, got %q", DiffMedium, problem.Difficulty)
	}
}

func TestCheckAnswerLogsEveryAttempt(t *testing.T) {
	ctx := context.Background()
	learnerID := "0f1fddf3-df53-4bc3-a163-90ff0f4f31f1"
	problem := MathProblem{
		ID:     "adv-001",
		Answer: 36,
		Hint1:  "hint one",
		Hint2:  "hint two",
	}

	wrongStore := &fakeAttemptStore{wrongAttempts: 1}
	wrongRes, err := checkAnswerWithStore(ctx, learnerID, problem, 0, wrongStore)
	if err != nil {
		t.Fatalf("wrong answer call failed: %v", err)
	}
	if len(wrongStore.execCorrectValues) != 1 || wrongStore.execCorrectValues[0] {
		t.Fatalf("expected one logged wrong attempt, got %+v", wrongStore.execCorrectValues)
	}
	if wrongRes.Hint != problem.Hint1 {
		t.Fatalf("expected first hint %q, got %q", problem.Hint1, wrongRes.Hint)
	}

	correctStore := &fakeAttemptStore{}
	correctRes, err := checkAnswerWithStore(ctx, learnerID, problem, 36, correctStore)
	if err != nil {
		t.Fatalf("correct answer call failed: %v", err)
	}
	if len(correctStore.execCorrectValues) != 1 || !correctStore.execCorrectValues[0] {
		t.Fatalf("expected one logged correct attempt, got %+v", correctStore.execCorrectValues)
	}
	if !correctRes.Correct {
		t.Fatalf("expected correct result, got %+v", correctRes)
	}
}

func TestPublicProblemResponseOmitsAnswerField(t *testing.T) {
	problem := MathProblem{
		ID:            "adv-001",
		NarrativeHook: "hook",
		Question:      "question",
		Hint1:         "hint1",
		Hint2:         "hint2",
		Genre:         "adventure",
		Answer:        36,
		Difficulty:    DiffMedium,
	}
	public := PublicProblem(problem)

	raw, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, exists := payload["answer"]; exists {
		t.Fatalf("expected response to omit answer field, got payload: %v", payload)
	}
}
