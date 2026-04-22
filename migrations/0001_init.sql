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

    creator_id VARCHAR(64) NOT NULL,
    reviewer_id VARCHAR(64),
    operator_id VARCHAR(64),
    approved_by VARCHAR(64),
    approved_at TIMESTAMPTZ,
    cancelled_by VARCHAR(64),
    cancelled_at TIMESTAMPTZ,

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
    operator_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN (
        'running',
        'succeeded',
        'failed',
        'cancelled'
    )),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    result_summary TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS approval_records (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    reviewer_id VARCHAR(64) NOT NULL,
    decision VARCHAR(32) NOT NULL CHECK (decision IN ('approved')),
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS task_status_histories (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    from_status VARCHAR(32),
    to_status VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor_id VARCHAR(64) NOT NULL,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN (
        'creator',
        'reviewer',
        'operator',
        'admin',
        'system'
    )),
    reason TEXT NOT NULL DEFAULT '',
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

CREATE INDEX IF NOT EXISTS idx_task_status_histories_task_id_created_at
    ON task_status_histories(task_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_task_id_occurred_at
    ON audit_logs(task_id, occurred_at DESC);
