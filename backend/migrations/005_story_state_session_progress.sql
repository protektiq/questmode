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
