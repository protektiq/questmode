package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	stateKeyPrefix      = "story:state:"
	stateTTL            = 24 * time.Hour
	maxIDLength         = 128
	maxResponseLength   = 64
	maxEngagementSecond = 86400
)

func Load(learnerID string, rdb *redis.Client) (StoryState, error) {
	learnerID = strings.TrimSpace(learnerID)
	if err := validateIdentifier("learnerID", learnerID); err != nil {
		return StoryState{}, err
	}
	if rdb == nil {
		return StoryState{}, errors.New("redis client is required")
	}

	raw, err := rdb.Get(context.Background(), storyStateKey(learnerID)).Result()
	if errors.Is(err, redis.Nil) {
		return StoryState{}, nil
	}
	if err != nil {
		return StoryState{}, fmt.Errorf("load story state from redis: %w", err)
	}

	var state StoryState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return StoryState{}, fmt.Errorf("unmarshal story state: %w", err)
	}
	return state, nil
}

func Save(state StoryState, rdb *redis.Client) error {
	state.LearnerID = strings.TrimSpace(state.LearnerID)
	if err := validateIdentifier("learner_id", state.LearnerID); err != nil {
		return err
	}
	if rdb == nil {
		return errors.New("redis client is required")
	}
	if state.EngagementSeconds < 0 || state.EngagementSeconds > maxEngagementSecond {
		return fmt.Errorf("engagement_seconds must be between 0 and %d", maxEngagementSecond)
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal story state: %w", err)
	}
	if err := rdb.Set(context.Background(), storyStateKey(state.LearnerID), payload, stateTTL).Err(); err != nil {
		return fmt.Errorf("save story state to redis: %w", err)
	}
	return nil
}

func FlushToDB(state StoryState, db *pgxpool.Pool) error {
	state.SessionID = strings.TrimSpace(state.SessionID)
	if err := validateIdentifier("session_id", state.SessionID); err != nil {
		return err
	}
	if db == nil {
		return errors.New("db pool is required")
	}
	if state.EngagementSeconds < 0 || state.EngagementSeconds > maxEngagementSecond {
		return fmt.Errorf("engagement_seconds must be between 0 and %d", maxEngagementSecond)
	}

	tasksCompleted := len(state.ChoicesMade)
	if _, err := db.Exec(
		context.Background(),
		`UPDATE quest_sessions
		 SET tasks_completed = $1, engagement_seconds = $2
		 WHERE session_id = $3`,
		tasksCompleted,
		state.EngagementSeconds,
		state.SessionID,
	); err != nil {
		return fmt.Errorf("flush story state to db: %w", err)
	}
	return nil
}

func ApplyBrainCheck(state *StoryState, response string) {
	if state == nil {
		return
	}
	response = strings.TrimSpace(response)
	if len(response) > maxResponseLength {
		response = response[:maxResponseLength]
	}
	state.BrainCheckFrustrated = response == "frustrated"
}

func storyStateKey(learnerID string) string {
	return stateKeyPrefix + learnerID
}

func validateIdentifier(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) > maxIDLength {
		return fmt.Errorf("%s exceeds maximum length of %d", name, maxIDLength)
	}
	return nil
}
