CREATE TABLE workspace_agent_script_timings
(
    display_name text not null,
    started_at   timestamp with time zone not null,
    ended_at     timestamp with time zone not null,
    exit_code    int not null
);

ALTER TABLE workspace_agent_scripts ADD COLUMN display_name text not null default '';
