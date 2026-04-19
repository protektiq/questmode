-- Add progress tracking fields for spelling practice workflow.
ALTER TABLE spelling_mastery
  ADD COLUMN IF NOT EXISTS correct_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS hint_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS mastered_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_spelling_mastery_active_by_learner_last_seen
  ON spelling_mastery (learner_id, mastered_at, last_seen_at);
