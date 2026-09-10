-- +migrate Up

-- comment_summaries holds one cached AI summary per activity, built
-- from that activity's approved comments. It is its own table, not
-- new columns on activities, on purpose:
--   - activities already uses optimistic locking (version). A
--     background summary refresh should never fight a real edit
--     to the activity for that same version counter.
--   - it keeps the AI-summary concern entirely inside the Comment
--     bounded context, which is the thing that actually knows when
--     a summary needs regenerating (a comment got approved).
CREATE TABLE comment_summaries (
    activity_id    UUID PRIMARY KEY REFERENCES activities(id) ON DELETE CASCADE,
    summary        TEXT NOT NULL DEFAULT '',
    -- how many approved comments this summary was built from, so a
    -- caller can tell "no comments yet" apart from "not generated yet".
    comment_count  INT NOT NULL DEFAULT 0,
    generated_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +migrate Down

DROP TABLE IF EXISTS comment_summaries;
