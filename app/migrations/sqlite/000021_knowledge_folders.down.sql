DROP INDEX IF EXISTS idx_knowledges_folder_id;
ALTER TABLE knowledges DROP COLUMN folder_id;
DROP TABLE IF EXISTS knowledge_folders;
