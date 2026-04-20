# AgentOps 可扩展性审查

## 结论

按严苛可扩展性标准，这个库目前更像学习型单机任务 API，还不是能横向扩展的任务平台。主要瓶颈集中在无界查询、状态机并发模型、索引设计、执行器抽象和审计日志增长。

## 主要问题

### 1. 高风险：所有列表型接口基本无分页

`GET /tasks` 只有 `Items`，没有 `limit/cursor/filter`，见 `agentops.api:49`。底层直接 `SELECT ... FROM tasks ORDER BY id DESC` 全量读，见 `internal/model/taskmodel.go:106`。

日志、执行记录、状态历史也类似：

- `internal/model/auditlogmodel.go:80`
- `internal/model/taskexecutionmodel.go:139`
- `internal/model/taskstatushistorymodel.go:70`

数据一多会拖垮 DB、内存和 HTTP 响应。

建议：

- 所有列表接口改成 cursor 分页。
- `GET /tasks` 至少支持 `limit/status/createdBefore/cursor`。
- 日志接口按 `(task_id, step, id)` 或 `(task_id, occurred_at, id)` 游标读取。

### 2. 高风险：索引和查询模式不匹配

`tasks` 建的是 `(status, created_at DESC)`，见 `migrations/0001_init.sql:97`，但列表按 `id DESC` 查，见 `internal/model/taskmodel.go:128`。

`audit_logs` 建的是 `(task_id, occurred_at DESC)`，见 `migrations/0001_init.sql:109`，但查询按 `step ASC, id ASC` 排，见 `internal/model/auditlogmodel.go:93`。

这会导致大表排序和慢查询。

建议：

- 如果任务列表主排序是 `id DESC`，补 `CREATE INDEX ON tasks (id DESC)` 或改成按 `created_at DESC, id DESC` 并配套索引。
- 日志如果以 step 展示，补 `CREATE INDEX ON audit_logs (task_id, step ASC, id ASC)`。
- 状态历史若按 `created_at ASC, id ASC` 查，也应补匹配索引。

### 3. 高风险：状态流转靠 `FOR UPDATE` 串行化

`FindByIDForUpdate` 锁整行任务，见 `internal/model/taskmodel.go:229`。`start/succeed/fail/cancel` 都在锁内做多次写入，例如 `internal/logic/tasks/starttasklogic.go:59`。

小并发没问题，但多 worker、多实例下会形成热点。更可扩展的做法是条件更新：

```sql
UPDATE tasks
SET status = 'running', operator_id = $2, updated_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING id, status, updated_at;
```

建议：

- 把状态校验和状态更新合并成一条条件更新。
- 用 affected rows / returning 结果判断并发失败。
- 审计日志和状态历史作为事件追加写入，不要把大量逻辑堆在同一个锁窗口内。

### 4. 高风险：没有真正的 worker lease / 队列语义

API 是手动 `start/succeed/fail`，见 `agentops.api:183`。`task_executions` 没有 `attempt_no / lease_until / heartbeat_at / worker_id`，见 `migrations/0001_init.sql:43`。

这意味着多执行器无法安全抢占任务，也无法恢复卡死的 `running` 任务。

建议：

- 引入 `task_claims` 或扩展 `task_executions`。
- 增加 `attempt_no`、`worker_id`、`lease_until`、`heartbeat_at`。
- 领取任务使用 `FOR UPDATE SKIP LOCKED` 或条件更新批量 claim。
- 超时 running 任务应可被重新投递或标记 failed。

### 5. 中高风险：审计日志 step 用 `MAX(step)+1`

每次状态变更都查最大 step，见 `internal/model/auditlogmodel.go:64`，表上也没有 `(task_id, step)` 唯一约束，见 `migrations/0001_init.sql:86`。

一旦日志写入变成异步或多来源，很容易重复 step 或变成热点查询。

建议：

- 增加 `UNIQUE(task_id, step)`。
- 更推荐用自增 `id` 作为稳定顺序，`step` 只作为业务展示字段。
- 如果必须连续 step，应使用单独计数器表或在任务行上维护 `next_log_step`，并明确锁边界。

### 6. 中风险：创建任务同步执行多个 `git` 命令且无超时

`CreateTask` 请求路径里直接读 Git，见 `internal/logic/tasks/createtasklogic.go:65`。内部用 `exec.Command`，不是 `CommandContext`，见 `internal/gitctx/gitctx.go:47`。

大仓库、网络盘、损坏仓库会拖住请求 goroutine。

建议：

- 改用 `exec.CommandContext` 并设置短超时。
- 对 Git 信息采集做缓存或异步化。
- 创建任务时只做最小校验，复杂仓库扫描交给后台 worker。

### 7. 中风险：表结构还没有多租户、分区、保留策略

核心表只有全局 `tasks/audit_logs`，无 workspace/tenant 维度、无日志分区、无归档字段，见 `migrations/0001_init.sql:1`。

后续要做团队隔离、冷热数据、按租户限流会比较痛。

建议：

- 早期就加入 `workspace_id` 或 `tenant_id`。
- 大表尤其是 `audit_logs` 设计保留策略。
- 日志表考虑按时间或 tenant 分区。
- 常用查询索引都应包含 tenant 前缀。

### 8. 中风险：认证和配额还是占位

登录直接返回固定 token，见 `internal/logic/auth/loginlogic.go:35`。路由侧也没有看到鉴权、限流、配额中间件。

可扩展系统不只是能处理流量，还要能限制滥用和隔离资源。

建议：

- 接入真实认证。
- 所有写接口加身份和权限校验。
- 按用户、tenant、repo 加限流和任务并发上限。

## 推荐整改顺序

1. 补分页和匹配索引。
2. 把状态流转改成条件更新，减少 `FOR UPDATE` 锁窗口。
3. 设计 worker claim / lease / heartbeat 模型。
4. 调整审计日志顺序模型和唯一约束。
5. 给 Git 命令加 context 超时，评估异步化。
6. 增加 tenant/workspace 维度、日志保留策略和鉴权限流。
