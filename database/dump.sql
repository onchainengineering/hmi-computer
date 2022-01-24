-- Code generated by 'make database/generate'. DO NOT EDIT.

CREATE TYPE log_level AS ENUM (
    'trace',
    'debug',
    'info',
    'warn',
    'error',
    'fatal'
);

CREATE TYPE login_type AS ENUM (
    'built-in',
    'saml',
    'oidc'
);

CREATE TYPE parameter_type_system AS ENUM (
    'hcl'
);

CREATE TYPE project_storage_method AS ENUM (
    'inline-archive'
);

CREATE TYPE provisioner_type AS ENUM (
    'terraform',
    'cdr-basic'
);

CREATE TYPE userstatus AS ENUM (
    'active',
    'dormant',
    'decommissioned'
);

CREATE TYPE workspace_transition AS ENUM (
    'create',
    'start',
    'stop',
    'delete'
);

CREATE TABLE api_keys (
    id text NOT NULL,
    hashed_secret bytea NOT NULL,
    user_id text NOT NULL,
    application boolean NOT NULL,
    name text NOT NULL,
    last_used timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    login_type login_type NOT NULL,
    oidc_access_token text DEFAULT ''::text NOT NULL,
    oidc_refresh_token text DEFAULT ''::text NOT NULL,
    oidc_id_token text DEFAULT ''::text NOT NULL,
    oidc_expiry timestamp with time zone DEFAULT '0001-01-01 00:00:00+00'::timestamp with time zone NOT NULL,
    devurl_token boolean DEFAULT false NOT NULL
);

CREATE TABLE licenses (
    id integer NOT NULL,
    license jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE organization_members (
    organization_id text NOT NULL,
    user_id text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    roles text[] DEFAULT '{organization-member}'::text[] NOT NULL
);

CREATE TABLE organizations (
    id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    "default" boolean DEFAULT false NOT NULL,
    auto_off_threshold bigint DEFAULT '28800000000000'::bigint NOT NULL,
    cpu_provisioning_rate real DEFAULT 4.0 NOT NULL,
    memory_provisioning_rate real DEFAULT 1.0 NOT NULL,
    workspace_auto_off boolean DEFAULT false NOT NULL
);

CREATE TABLE project (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id text NOT NULL,
    name character varying(64) NOT NULL,
    provisioner provisioner_type NOT NULL,
    active_version_id uuid
);

CREATE TABLE project_history (
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(64) NOT NULL,
    description character varying(1048576) NOT NULL,
    storage_method project_storage_method NOT NULL,
    storage_source bytea NOT NULL,
    import_job_id uuid NOT NULL
);

CREATE TABLE project_parameter (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    project_history_id uuid NOT NULL,
    name character varying(64) NOT NULL,
    description character varying(8192) DEFAULT ''::character varying NOT NULL,
    default_source text,
    allow_override_source boolean NOT NULL,
    default_destination text,
    allow_override_destination boolean NOT NULL,
    default_refresh text NOT NULL,
    redisplay_value boolean NOT NULL,
    validation_error character varying(256) NOT NULL,
    validation_condition character varying(512) NOT NULL,
    validation_type_system parameter_type_system NOT NULL,
    validation_value_type character varying(64) NOT NULL
);

CREATE TABLE users (
    id text NOT NULL,
    email text NOT NULL,
    name text NOT NULL,
    revoked boolean NOT NULL,
    login_type login_type NOT NULL,
    hashed_password bytea NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    temporary_password boolean DEFAULT false NOT NULL,
    avatar_hash text DEFAULT ''::text NOT NULL,
    ssh_key_regenerated_at timestamp with time zone DEFAULT now() NOT NULL,
    username text DEFAULT ''::text NOT NULL,
    dotfiles_git_uri text DEFAULT ''::text NOT NULL,
    roles text[] DEFAULT '{site-member}'::text[] NOT NULL,
    status userstatus DEFAULT 'active'::public.userstatus NOT NULL,
    relatime timestamp with time zone DEFAULT now() NOT NULL,
    gpg_key_regenerated_at timestamp with time zone DEFAULT now() NOT NULL,
    _decomissioned boolean DEFAULT false NOT NULL,
    shell text DEFAULT ''::text NOT NULL
);

CREATE TABLE workspace (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    owner_id text NOT NULL,
    project_id uuid NOT NULL,
    project_history_id uuid NOT NULL,
    name character varying(64) NOT NULL
);

CREATE TABLE workspace_agent (
    id uuid NOT NULL,
    workspace_resource_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    instance_metadata jsonb NOT NULL,
    resource_metadata jsonb NOT NULL
);

CREATE TABLE workspace_history (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    workspace_id uuid NOT NULL,
    project_history_id uuid NOT NULL,
    before_id uuid,
    after_id uuid,
    transition workspace_transition NOT NULL,
    initiator character varying(255) NOT NULL,
    provisioner_state bytea,
    provision_job_id uuid NOT NULL
);

CREATE TABLE workspace_log (
    workspace_id uuid NOT NULL,
    workspace_history_id uuid NOT NULL,
    created timestamp with time zone NOT NULL,
    logged_by character varying(255),
    level log_level NOT NULL,
    log jsonb NOT NULL
);

CREATE TABLE workspace_resource (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    workspace_history_id uuid NOT NULL,
    type character varying(256) NOT NULL,
    name character varying(64) NOT NULL,
    workspace_agent_token character varying(128) NOT NULL,
    workspace_agent_id uuid
);

ALTER TABLE ONLY project_history
    ADD CONSTRAINT project_history_id_key UNIQUE (id);

ALTER TABLE ONLY project_history
    ADD CONSTRAINT project_history_project_id_name_key UNIQUE (project_id, name);

ALTER TABLE ONLY project
    ADD CONSTRAINT project_id_key UNIQUE (id);

ALTER TABLE ONLY project
    ADD CONSTRAINT project_organization_id_name_key UNIQUE (organization_id, name);

ALTER TABLE ONLY project_parameter
    ADD CONSTRAINT project_parameter_id_key UNIQUE (id);

ALTER TABLE ONLY project_parameter
    ADD CONSTRAINT project_parameter_project_history_id_name_key UNIQUE (project_history_id, name);

ALTER TABLE ONLY workspace_agent
    ADD CONSTRAINT workspace_agent_id_key UNIQUE (id);

ALTER TABLE ONLY workspace_history
    ADD CONSTRAINT workspace_history_id_key UNIQUE (id);

ALTER TABLE ONLY workspace
    ADD CONSTRAINT workspace_id_key UNIQUE (id);

ALTER TABLE ONLY workspace_resource
    ADD CONSTRAINT workspace_resource_id_key UNIQUE (id);

ALTER TABLE ONLY workspace_resource
    ADD CONSTRAINT workspace_resource_workspace_agent_token_key UNIQUE (workspace_agent_token);

ALTER TABLE ONLY workspace_resource
    ADD CONSTRAINT workspace_resource_workspace_history_id_name_key UNIQUE (workspace_history_id, name);

CREATE INDEX workspace_log_index ON workspace_log USING btree (workspace_id, workspace_history_id);

ALTER TABLE ONLY project_history
    ADD CONSTRAINT project_history_project_id_fkey FOREIGN KEY (project_id) REFERENCES project(id);

ALTER TABLE ONLY project_parameter
    ADD CONSTRAINT project_parameter_project_history_id_fkey FOREIGN KEY (project_history_id) REFERENCES project_history(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace_agent
    ADD CONSTRAINT workspace_agent_workspace_resource_id_fkey FOREIGN KEY (workspace_resource_id) REFERENCES workspace_resource(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace_history
    ADD CONSTRAINT workspace_history_project_history_id_fkey FOREIGN KEY (project_history_id) REFERENCES project_history(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace_history
    ADD CONSTRAINT workspace_history_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspace(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace_log
    ADD CONSTRAINT workspace_log_workspace_history_id_fkey FOREIGN KEY (workspace_history_id) REFERENCES workspace_history(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace_log
    ADD CONSTRAINT workspace_log_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspace(id) ON DELETE CASCADE;

ALTER TABLE ONLY workspace
    ADD CONSTRAINT workspace_project_history_id_fkey FOREIGN KEY (project_history_id) REFERENCES project_history(id);

ALTER TABLE ONLY workspace
    ADD CONSTRAINT workspace_project_id_fkey FOREIGN KEY (project_id) REFERENCES project(id);

ALTER TABLE ONLY workspace_resource
    ADD CONSTRAINT workspace_resource_workspace_history_id_fkey FOREIGN KEY (workspace_history_id) REFERENCES workspace_history(id) ON DELETE CASCADE;

