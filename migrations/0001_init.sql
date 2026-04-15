CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    repo_path TEXT NOT NULL,
    prompt TEXT NOT NULL,
    mode VARCHAR(32) NOT NULL CHECK (mode IN ('analyze', 'patch')),
    status VARCHAR(32) NOT NULL CHECK (status IN (
        'waiting_approval',
        'pending',
        'running',
        'succeeded',
        'failed',
        'cancelled'
    )),
    approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    max_steps BIGINT NOT NULL CHECK (max_steps > 0),
    created_by VARCHAR(64) NOT NULL DEFAULT 'system',

    git_branch VARCHAR(255) NOT NULL DEFAULT '',
    git_head_commit VARCHAR(64) NOT NULL DEFAULT '',
    git_dirty BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS task_policies (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
    allowed_paths TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    denied_paths TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS task_executions (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL CHECK (status IN (
        'pending',
        'running',
        'succeeded',
        'failed',
        'cancelled'
    )),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    result_summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS approval_records (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    approved_by VARCHAR(64) NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    step BIGINT NOT NULL DEFAULT 0,
    level VARCHAR(32) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    tool_name VARCHAR(64) NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at
    ON tasks(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_task_executions_task_id
    ON task_executions(task_id);

CREATE INDEX IF NOT EXISTS idx_approval_records_task_id
    ON approval_records(task_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_task_id_occurred_at
    ON audit_logs(task_id, occurred_at DESC);
