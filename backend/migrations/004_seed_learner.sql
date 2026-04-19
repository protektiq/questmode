-- Seed data for local/dev; idempotent via ON CONFLICT.

INSERT INTO learner_profile (id, display_name)
VALUES ('00000000-0000-4000-8000-000000000001'::uuid, 'Demo Learner')
ON CONFLICT (id) DO NOTHING;

INSERT INTO quest_arcs (id, title, sort_order)
VALUES ('00000000-0000-4000-8000-000000000002'::uuid, 'Arc One', 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO quest_chapters (id, arc_id, title, sort_order)
VALUES (
  '00000000-0000-4000-8000-000000000003'::uuid,
  '00000000-0000-4000-8000-000000000002'::uuid,
  'Chapter One',
  0
)
ON CONFLICT (id) DO NOTHING;
