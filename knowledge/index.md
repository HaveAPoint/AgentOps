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

### Project

- [AgentOps L2 approve 链路笔记](./project/agentops_l2_approve_flow.md)
- [approval_records 当前语义与 L2 目标](./project/approval_records_semantics.md)
- [AgentOps L2 任务生命周期图](./project/task_lifecycle_map.md)
- [create / approve / start 的角色语义](./project/create_approve_start_roles.md)
- [StartTask 的最小实现为什么先只做 `pending -> running`](./project/starttask_minimal_impl.md)
