# Project

AgentOps 项目实战知识的分类入口。

适合放这里的内容：

- 当前项目中的接口
- 当前项目中的表结构
- 当前项目中的关键链路
- 当前项目中的错误案例
- 当前项目中的设计取舍

当前笔记：

- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
- [approval_records 当前语义与 L2 目标](./approval_records_semantics.md)
- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [create / approve / start 的角色语义](./create_approve_start_roles.md)
- [StartTask 的最小实现为什么先只做 `pending -> running`](./starttask_minimal_impl.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)
- [AgentOps L2 集成测试为什么这样写](./l2_integration_testing_notes.md)
- [AgentOps 中 `task`、`execution` 与 `Finish` 的边界](./task_execution_finish_boundary.md)
- [开发期为什么直接重写 schema 基线，而不是继续做兼容迁移](./development_phase_schema_baseline_rewrite.md)
- [AgentOps 里 `.api -> goctl -> types/handler/logic` 为什么必须同步](./api_goctl_sync_boundary.md)
