-- SQLite mirror of 000124_reading_question_kind.

ALTER TABLE reading_questions ADD COLUMN kind TEXT NOT NULL DEFAULT '';
ALTER TABLE reading_questions ADD COLUMN gap_type TEXT NOT NULL DEFAULT '';
