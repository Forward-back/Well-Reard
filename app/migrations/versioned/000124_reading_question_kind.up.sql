-- 000124: question kind and gap type for the reading question quality gate.

ALTER TABLE reading_questions ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE reading_questions ADD COLUMN gap_type VARCHAR(24) NOT NULL DEFAULT '';
