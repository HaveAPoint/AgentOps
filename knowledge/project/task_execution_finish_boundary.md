# 【主题】
AgentOps 中 `task`、`execution` 与 `Finish` 的边界

一句话概述：

这条链路最容易混的不是某一行 SQL，而是“任务实例”和“执行实例”没有分开；一旦把 `tasks`、`task_executions`、`Finish(...)`、`TaskExecution`、`FinishExecutionParams` 的边界分清，L2 主链就会顺很多。

## 【所属分类】
项目实战 / 当前项目中的关键链路

## 【核心结论】

- 当前代码已经把 L2 主链推进到了 `create -> approve -> start -> succeed`，但还没整体收口完成。
- `task` 和 `execution` 不是一层语义：`task` 是任务当前快照，`execution` 是一次执行实例。
- `TaskExecutionModel.Finish(...)` 只负责收口一条 execution 记录，不负责判断业务是否合法。
- `TaskExecution` 对应数据库整行结构，`FinishExecutionParams` 对应 finish 这个业务动作的输入。
- `execution.ID` 是执行记录自己的主键，`TaskID` 是它归属的任务 id；finish 应按 `execution.ID` 精确更新。

## 【展开解释】

### 1. 当前 L2 代码推进到了哪

基于当前仓库代码，主链已经不是文档阶段，而是已经有了明确实现：

- `create`
  - 根据 `approvalRequired` 进入 `pending` 或 `waiting_approval`
- `approve`
  - 已经修正成 `waiting_approval -> pending`
- `start`
  - `pending -> running`
  - 同时插入一条 `task_executions`
- `succeed`
  - 在事务内同时更新 `tasks`、`task_executions`、`audit_logs`

但当前还没有完全收口：

- `fail` 还没有正式实现
- `cancel` 与当前 L2 文档约束还有偏差
- `task_status_histories` 只有表，还没有真正写入链路
- 目前没有行为测试覆盖这些迁移

所以更准确的判断是：

- L2 主链基本打通
- 但还没有达到“闭环已完成”的状态

### 2. `task` 和 `execution` 为什么一定要分开

当前链路里：

- `tasks`
  - 记录任务当前快照
  - 例如当前状态、当前 operator、审批信息
- `task_executions`
  - 记录一次具体执行实例
  - 例如谁执行、何时开始、何时结束、结果摘要、错误信息

最容易混的点是：

- `pending` 是 `tasks.status`
- 不是 `task_executions.status`

当前实现里，任务创建后即使进入 `pending`，也还没有 execution 记录。

只有 `start` 的时候，才会真正插入一条 `task_executions`。因此：

- “任务已经创建好”
- 和“某次执行已经开始”

这是两个不同阶段，不能混成一件事。

### 3. `Finish(...)` 的职责边界

`TaskExecutionModel.Finish(...)` 本质上只是：

- 根据 execution 主键定位一条记录
- 更新 `status / finished_at / result_summary / error_message`

它不负责：

- 校验 task 当前是否处于 `running`
- 校验 operator 是否匹配
- 校验 running execution 是否存在

这些都应该放在 logic 层处理。

也就是说：

- logic 负责“这次 finish 合不合法”
- model 负责“这条 SQL 怎么写”

这也是为什么 `Finish(...)` 接收 `DBTX`：

- 这样它可以和 `tasks`、`audit_logs` 的更新放在同一个事务里
- 保证 task 状态和 execution 状态不分裂

### 4. 为什么有 `FinishExecutionParams`

这里有两个不同层次的 struct：

- `TaskExecution`
  - 描述数据库里一整行 execution 长什么样
- `FinishExecutionParams`
  - 描述 finish 这个动作需要什么输入

这两个概念不能强行合并。

因为 `Finish(...)` 实际只会更新：

- `status`
- `finished_at`
- `result_summary`
- `error_message`

它并不需要：

- `TaskID`
- `OperatorId`
- `StartedAt`
- `CreatedAt`

如果直接把整个 `TaskExecution` 传进去，方法签名会变宽，调用方也更容易误解“是不是整行字段都参与了更新”。

另外，两个类型在时间字段上的语义也不同：

- `TaskExecution.FinishedAt` 用 `sql.NullTime`
  - 表达数据库列可空
- `FinishExecutionParams.FinishedAt` 用 `time.Time`
  - 表达 finish 动作里结束时间必须给

因此当前更合适的边界是：

- 插入整行时用 `TaskExecution`
- 按动作更新时用 `FinishExecutionParams`

### 5. `execution.ID`、`TaskID`、`CreatedAt`、`StartedAt` 的区别

`TaskExecution` 里最容易混的字段是：

- `ID`
  - `task_executions` 这张表自己的主键
- `TaskID`
  - 外键，指向它归属的 `tasks.id`
- `CreatedAt`
  - execution 这条数据库记录被插入的时间
- `StartedAt`
  - 这次执行在业务语义上的开始时间

其中：

- `CreatedAt` 不是 task 的创建时间
- `ID` 也不是 task 的 id

当前 finish 链路先用 `taskID` 找到“这个任务当前那条 running execution”，再使用 `execution.ID` 做精确更新。

这是因为：

- `taskID` 只能说明“属于哪个任务”
- `execution.ID` 才能说明“是哪一条 execution 记录”

短期内如果一个 task 只有一条 execution，按 `task_id` 更新看上去似乎也能跑；
但从数据建模上，它仍然是范围定位，不是实例定位。

因此更稳的做法是：

- 用 `taskID` 缩小查找范围
- 用 `execution.ID` 精确落更新

### 6. `created_at` 和 `started_at` 为什么不要视为同一个字段

它们通常很接近，但语义不同：

- `started_at`
  - 是应用层以业务语义写进去的开始时间
- `created_at`
  - 是 execution 行落库时生成的记录创建时间

在当前实现里，两者大概率接近，但并不保证完全相等。

因此：

- `started_at` 更像业务事件时间
- `created_at` 更像数据库记录时间

二者不要混成一个概念。

## 【代码/场景对应】

当前项目中直接对应：

- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)
- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
- [internal/logic/tasks/succeedtasklogic.go](../../internal/logic/tasks/succeedtasklogic.go)
- [internal/logic/tasks/failtasklogic.go](../../internal/logic/tasks/failtasklogic.go)
- [phase/taskL2.md](../../phase/taskL2.md)

## 【易错点】

- 把 `task` 和 `execution` 当成同一层状态。
- 误以为任务进入 `pending` 时就已经有 execution 记录。
- 误以为 `Finish(...)` 应该顺手做业务状态机校验。
- 误以为 “统一复用一个 struct” 一定比“实体 struct + 动作 params struct”更好。
- 把 `execution.ID` 和 `TaskID` 混用，导致更新定位不精确。
- 把 `created_at` 和 `started_at` 当成完全等价的时间字段。

## 【关联知识】

- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)
