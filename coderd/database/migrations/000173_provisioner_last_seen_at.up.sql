ALTER TABLE provisioner_daemons
	ADD COLUMN last_seen_at TIMESTAMP WITH TIME ZONE NULL,
	ADD COLUMN version TEXT NOT NULL DEFAULT ''::TEXT;
