ALTER TABLE
    workspace_resources
ADD
    COLUMN module TEXT;

CREATE TABLE workspace_modules (
    id uuid NOT NULL,
    job_id uuid NOT NULL,
    transition workspace_transition NOT NULL,
    source TEXT NOT NULL,
    version TEXT NOT NULL,
    key TEXT NOT NULL,
    created_at timestamp with time zone NOT NULL
);
