ALTER TABLE quest_sessions
  ADD COLUMN IF NOT EXISTS session_id TEXT;

UPDATE quest_sessions
SET session_id = id::text
WHERE session_id IS NULL;

ALTER TABLE quest_sessions
  ALTER COLUMN session_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_quest_sessions_session_id
  ON quest_sessions(session_id);

ALTER TABLE quest_sessions
  ADD COLUMN IF NOT EXISTS tasks_completed INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS engagement_seconds INTEGER NOT NULL DEFAULT 0;

ALTER TABLE quest_sessions
  DROP CONSTRAINT IF EXISTS quest_sessions_tasks_completed_nonnegative,
  ADD CONSTRAINT quest_sessions_tasks_completed_nonnegative CHECK (tasks_completed >= 0),
  DROP CONSTRAINT IF EXISTS quest_sessions_engagement_seconds_nonnegative,
  ADD CONSTRAINT quest_sessions_engagement_seconds_nonnegative CHECK (engagement_seconds >= 0);

ALTER TABLE quest_arcs
  ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS quest_badges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  arc_id UUID NOT NULL REFERENCES quest_arcs(id) ON DELETE CASCADE,
  learner_id UUID NOT NULL REFERENCES learner_profile(id) ON DELETE CASCADE,
  genre TEXT NOT NULL,
  earned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (arc_id, learner_id)
);

ALTER TABLE quest_badges
  ADD COLUMN IF NOT EXISTS arc_id UUID REFERENCES quest_arcs(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS learner_id UUID REFERENCES learner_profile(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS genre TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS earned_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_quest_badges_learner_earned_at
  ON quest_badges (learner_id, earned_at DESC);

CREATE INDEX IF NOT EXISTS idx_quest_badges_arc_id
  ON quest_badges (arc_id);
