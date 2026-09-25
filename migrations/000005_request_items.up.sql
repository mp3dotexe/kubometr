-- The consultant's final list the client agreed to by making the request.
ALTER TABLE requests ADD COLUMN IF NOT EXISTS items TEXT NOT NULL DEFAULT '';
