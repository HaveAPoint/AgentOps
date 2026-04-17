# Knowledge Index

这是知识仓库的总索引。先按分类入口找，再进入具体主题笔记。

## 分类入口

- [Go](./go/README.md)
- [Database](./database/README.md)
- [Project](./project/README.md)

## 主题笔记

### Go

- [Go 请求绑定、导出字段与 tag](./go/request_binding_and_tags.md)
- [Go test、GOCACHE 与 goctl 生成边界](./go/go_test_gocache_and_goctl_generation.md)

### Database

- [事务、Rollback/Commit 与 FOR UPDATE](./database/transactions_and_row_locks.md)
- [NULL、默认值与空字符串](./database/null_default_and_empty_string.md)
- [`sql.NullString`、`*string` 和普通 `string` 的区别](./database/sql_nullstring_vs_ptr_string.md)
- [时间字段、`NOW()`、`TIMESTAMPTZ`、`time.Time` 与 RFC3339 的区别](./database/time_fields_now_timestamptz_and_rfc3339.md)

### Project

- [AgentOps L2 approve 链路笔记](./project/agentops_l2_approve_flow.md)
- [approval_records 当前语义与 L2 目标](./project/approval_records_semantics.md)
- [AgentOps L2 任务生命周期图](./project/task_lifecycle_map.md)
- [create / approve / start 的角色语义](./project/create_approve_start_roles.md)
- [StartTask 的最小实现为什么先只做 `pending -> running`](./project/starttask_minimal_impl.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./project/task_snapshot_execution_history_audit.md)
- [AgentOps 中 logic 层与 model 层的边界](./project/logic_model_boundary_in_agentops.md)
- [AgentOps L2 集成测试为什么这样写](./project/l2_integration_testing_notes.md)
- [AgentOps 中 `task`、`execution` 与 `Finish` 的边界](./project/task_execution_finish_boundary.md)
- [开发期为什么直接重写 schema 基线，而不是继续做兼容迁移](./project/development_phase_schema_baseline_rewrite.md)
- [AgentOps 里 `.api -> goctl -> types/handler/logic` 为什么必须同步](./project/api_goctl_sync_boundary.md)
