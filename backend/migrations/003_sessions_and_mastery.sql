-- Inferred schema: sessions, spelling progress, and math attempts.

CREATE TABLE IF NOT EXISTS quest_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  chapter_id UUID REFERENCES quest_chapters(id) ON DELETE SET NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS spelling_mastery (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  word TEXT NOT NULL,
  mastery_level SMALLINT NOT NULL DEFAULT 0,
  last_practiced_at TIMESTAMPTZ,
  UNIQUE (learner_id, word)
);

CREATE TABLE IF NOT EXISTS math_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  problem TEXT NOT NULL,
  correct BOOLEAN NOT NULL,
  attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_quest_sessions_learner_id ON quest_sessions(learner_id);
CREATE INDEX IF NOT EXISTS idx_spelling_mastery_learner_id ON spelling_mastery(learner_id);
CREATE INDEX IF NOT EXISTS idx_math_attempts_learner_id ON math_attempts(learner_id);
