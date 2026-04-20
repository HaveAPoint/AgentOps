CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    system_role VARCHAR(32) NOT NULL CHECK (system_role IN (
        'admin',
        'reviewer',
        'operator',
        'viewer'
    )),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_creator_user
    FOREIGN KEY (creator_id) REFERENCES users(id);

ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_reviewer_user
    FOREIGN KEY (reviewer_id) REFERENCES users(id);

ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_operator_user
    FOREIGN KEY (operator_id) REFERENCES users(id);

ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_approved_by_user
    FOREIGN KEY (approved_by) REFERENCES users(id);

ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_cancelled_by_user
    FOREIGN KEY (cancelled_by) REFERENCES users(id);

ALTER TABLE task_executions
    ADD CONSTRAINT fk_task_executions_operator_user
    FOREIGN KEY (operator_id) REFERENCES users(id);

ALTER TABLE approval_records
    ADD CONSTRAINT fk_approval_records_reviewer_user
    FOREIGN KEY (reviewer_id) REFERENCES users(id);

ALTER TABLE task_status_histories
    ADD CONSTRAINT fk_task_status_histories_actor_user
    FOREIGN KEY (actor_id) REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_users_username
    ON users(username);

CREATE INDEX IF NOT EXISTS idx_users_system_role
    ON users(system_role);

