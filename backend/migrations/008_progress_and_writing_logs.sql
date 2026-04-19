-- Progress dashboard support: writing logs, badges, and math problem grouping.

CREATE TABLE IF NOT EXISTS writing_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  session_id UUID NOT NULL REFERENCES quest_sessions(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  word_count INTEGER NOT NULL,
  logged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT writing_logs_word_count_nonnegative CHECK (word_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_writing_logs_learner_logged_at
  ON writing_logs (learner_id, logged_at DESC);

CREATE INDEX IF NOT EXISTS idx_writing_logs_session_id
  ON writing_logs (session_id);

ALTER TABLE math_attempts
  ADD COLUMN IF NOT EXISTS problem_type TEXT NOT NULL DEFAULT 'general';

UPDATE math_attempts
SET problem_type = 'general'
WHERE problem_type IS NULL OR btrim(problem_type) = '';

CREATE INDEX IF NOT EXISTS idx_math_attempts_learner_problem_type
  ON math_attempts (learner_id, problem_type);

CREATE TABLE IF NOT EXISTS quest_badges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  badge_code TEXT NOT NULL,
  badge_name TEXT NOT NULL DEFAULT '',
  earned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (learner_id, badge_code)
);

CREATE INDEX IF NOT EXISTS idx_quest_badges_learner_earned_at
  ON quest_badges (learner_id, earned_at DESC);
