-- Story API schema alignment for runtime arc/chapter/session handlers.

-- Compatibility alias so handlers can query learner_profiles while the base table remains learner_profile.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_views
    WHERE schemaname = 'public' AND viewname = 'learner_profiles'
  ) THEN
    EXECUTE 'CREATE VIEW learner_profiles AS SELECT * FROM learner_profile';
  END IF;
END $$;

ALTER TABLE quest_arcs
  ADD COLUMN IF NOT EXISTS learner_id UUID REFERENCES learner_profile(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS genre TEXT NOT NULL DEFAULT 'general',
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

CREATE INDEX IF NOT EXISTS idx_quest_arcs_learner_id ON quest_arcs(learner_id);

ALTER TABLE quest_chapters
  ADD COLUMN IF NOT EXISTS chapter_index INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS content_text TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS questions_json JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_quest_chapters_arc_chapter_index
  ON quest_chapters(arc_id, chapter_index);

ALTER TABLE quest_sessions
  ADD COLUMN IF NOT EXISTS arc_id UUID REFERENCES quest_arcs(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS brain_checks_json JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_quest_sessions_arc_id ON quest_sessions(arc_id);
