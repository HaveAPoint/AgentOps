# 【主题】
开发期为什么直接重写 schema 基线，而不是继续做兼容迁移

一句话概述：

当项目仍在开发阶段、历史数据没有保留价值时，直接把初始化 migration 改成目标语义，通常比保留旧字段再层层兼容更清晰、更便宜。

## 【所属分类】
项目实战 / 当前项目中的设计取舍

## 【核心结论】

- 开发期如果没有历史包袱，优先重写 `0001_init.sql` 这种基线文件，而不是继续堆 `ALTER TABLE`。
- 一旦决定切到 `creator_id / reviewer_id / operator_id` 这类目标字段，就不要再在代码里依赖旧列。
- “兼容更多”不等于“更稳”；在开发期，双轨语义往往会放大复杂度。
- schema、model、logic、API 的语义必须一起收敛，不能只改其中一层。

## 【展开解释】

线上项目之所以常用：

- `0001` 先建旧结构
- `0002`
- `0003`
- 再逐步兼容和回填

是因为线上已有真实数据，迁移策略首先要保证不丢数据、不停业务。

但当前项目的前提不同：

- 还在开发阶段
- 数据库里的历史数据没有保留价值
- 目标语义已经比较明确

这时继续保留旧字段，例如：

- `created_by`
- `approved_by` 的旧借位语义
- 旧的 `comment`

只会带来额外维护成本：

- SQL 要写兼容分支
- model 要做双字段映射
- logic 要区分“当前写法”和“最终写法”
- API 响应会出现新旧字段混杂

这类复杂度不会帮助当前 L2 主链落地，反而会拖慢推进。

因此更适合的做法是：

1. 直接重写 `migrations/0001_init.sql`
2. 让 `tasks`、`task_executions`、`approval_records` 一次切到 L2 目标语义
3. 把旧迁移文件退化为废弃说明，避免误导后续开发

## 【代码/场景对应】

当前项目中直接对应：

- [migrations/0001_init.sql](../../migrations/0001_init.sql)
- [migrations/0002_l2_task_lifecycle.sql](../../migrations/0002_l2_task_lifecycle.sql)
- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)
- [internal/model/approvalrecordmodel.go](../../internal/model/approvalrecordmodel.go)

当前会话里最终收敛出的目标语义包括：

- `tasks.creator_id / reviewer_id / operator_id`
- `tasks.approved_by / approved_at / cancelled_by / cancelled_at`
- `task_executions.operator_id / error_message`
- `approval_records.reviewer_id / decision / reason`

## 【易错点】

- schema 已经删掉旧列，SQL 里却还写 `COALESCE(new_col, old_col)`。
- 以为“先兼容着，后面再说”不会增加成本。实际上它会渗透到 model、logic、API 和测试里。
- 误把开发期项目当成线上系统来设计迁移路径。
- 基线已经重写，却还在知识和代码里沿用旧术语。

## 【关联知识】

- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [create / approve / start 的角色语义](./create_approve_start_roles.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)
