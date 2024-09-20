ALTER TABLE workspace_agent_scripts ADD COLUMN id uuid unique not null default gen_random_uuid();
ALTER TABLE workspace_agent_scripts ADD COLUMN display_name text not null default '';

CREATE TYPE workspace_agent_script_timing_stage AS ENUM (
    'start',
    'stop',
    'cron'
    );

CREATE TABLE workspace_agent_script_timings
(
    script_id     uuid                                not null references workspace_agent_scripts (id) on delete cascade,
    display_name  text                                not null,
    started_at    timestamp with time zone            not null,
    ended_at      timestamp with time zone            not null,
    exit_code     int                                 not null,
    stage         workspace_agent_script_timing_stage not null,
    timed_out     bool                                not null
);
