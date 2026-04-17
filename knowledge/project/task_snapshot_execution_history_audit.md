# 【主题】
`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别

一句话概述：

这四张表都会“记录东西”，但它们记录的不是同一种语义：有的是当前快照，有的是执行实例，有的是状态轨迹，有的是文本事件。

## 【所属分类】
项目实战 / 当前项目中的表结构

## 【核心结论】

- `tasks` 记录任务当前快照，尤其是当前状态。
- `task_executions` 记录一次执行尝试的明细。
- `task_status_histories` 记录状态机迁移轨迹。
- `audit_logs` 记录给人读的文本事件，不适合作为主查询依据。

## 【展开解释】

这四张表最容易混淆的地方，是它们都和“记录任务发生了什么”有关。

但它们回答的问题不同：

- `tasks`
  - 这个任务现在是什么状态？
  - 当前 creator / reviewer / operator 是谁？
  - 当前任务元数据是什么？

- `task_executions`
  - 这次执行是谁跑的？
  - 什么时间开始？
  - 什么时间结束？
  - 成功还是失败？
  - 失败信息是什么？

- `task_status_histories`
  - 状态从哪变到哪？
  - 是谁触发的？
  - 触发人的角色是什么？
  - 动作是什么？
  - 变更原因是什么？

- `audit_logs`
  - 给开发者或操作者展示一条可读事件描述
  - 比如“task started by operator: x”

因此：

- `tasks.status` 只能表达“当前状态”
- 它不能回答“这个任务一共执行过几次”
- 也不能回答“这个任务曾经从哪些状态迁移过来”

如果以后出现：

- `running -> failed -> running -> succeeded`

那么：

- `tasks` 最终只能看到 `succeeded`
- `task_executions` 可以看到至少两次执行
- `task_status_histories` 可以看到完整状态迁移链

## 【代码/场景对应】

当前项目中直接对应：

- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)
- [internal/model/taskstatushistorymodel.go](../../internal/model/taskstatushistorymodel.go)
- [internal/model/auditlogmodel.go](../../internal/model/auditlogmodel.go)
- [migrations/0001_init.sql](../../migrations/0001_init.sql)

## 【易错点】

- 误以为 `task_executions` 就是日志。
- 误以为 `audit_logs` 可以完全替代结构化状态历史。
- 误以为只要有 `tasks.status`，就足够还原任务全过程。

## 【关联知识】

- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)

