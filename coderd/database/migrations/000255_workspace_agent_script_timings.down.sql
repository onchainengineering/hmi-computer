DROP TYPE IF EXISTS workspace_agent_script_timing_stage CASCADE;
DROP TABLE IF EXISTS workspace_agent_script_timings;

ALTER TABLE workspace_agent_scripts DROP COLUMN display_name;
ALTER TABLE workspace_agent_scripts DROP COLUMN id;
