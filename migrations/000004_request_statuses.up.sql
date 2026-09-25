-- "done" becomes "issued": a request is now ready first, then issued.
ALTER TABLE requests DROP CONSTRAINT requests_status_check;
UPDATE requests SET status = 'issued' WHERE status = 'done';
ALTER TABLE requests ADD CONSTRAINT requests_status_check
    CHECK (status IN ('new', 'in_progress', 'ready', 'issued', 'cancelled'));
