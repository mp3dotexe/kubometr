ALTER TABLE requests DROP CONSTRAINT requests_status_check;
UPDATE requests SET status = 'done' WHERE status IN ('ready', 'issued', 'cancelled');
ALTER TABLE requests ADD CONSTRAINT requests_status_check
    CHECK (status IN ('new', 'in_progress', 'done'));
