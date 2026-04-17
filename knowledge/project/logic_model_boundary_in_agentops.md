# 【主题】
AgentOps 中 logic 层与 model 层的边界

一句话概述：

当前项目里，`logic` 负责业务规则和事务编排，`model` 负责 SQL 与字段落库；边界不是按类型名字切，而是按语义责任切。

## 【所属分类】
项目实战 / 设计取舍

## 【核心结论】

- `logic` 负责“为什么这样做”“先做什么后做什么”“这次动作是否合法”。
- `model` 负责“这张表怎么查”“这几列怎么写”“结果怎么 scan 出来”。
- `sql.NullString`、`sql.NullTime` 穿透到 logic，并不代表它们属于 logic，而是当前 logic 直接消费了 model struct。
- 如果一个规则换数据库后仍然成立，它更像 logic；如果它强依赖 SQL 写法，它更像 model。

## 【展开解释】

### logic 层通常负责什么

- 参数校验
- 状态机校验
- 角色校验
- 开事务
- 协调多张表一起更新
- 决定最终返回什么响应

例如 `StartTaskLogic` 的职责是：

- 校验 `id` 与 `operatorId`
- 查 task
- 判断当前状态是不是 `pending`
- 判断 operator 是否匹配
- 依次更新 task、history、execution、audit
- 提交事务

这些步骤都属于业务流程控制。

### model 层通常负责什么

- 单张表的 SQL
- 可空字段映射
- `QueryRowContext` / `ExecContext`
- `RETURNING`
- `Scan`

它不负责解释业务语义，只负责把数据读写到数据库。

### 为什么 `sql.NullString` 会出现在 logic

这是当前项目的实用主义取舍。

logic 直接使用了：

- `model.Task`
- `model.TaskExecution`
- `model.TaskStatusHistory`

而这些结构体本来就是数据库映射对象，所以 `sql.NullString`、`sql.NullTime` 会跟着一起上浮到 logic。

这不是立刻要修的 bug，但说明当前分层是“够用版”，不是“最纯版”。

### 一个很有用的判断方法

看到一段逻辑时，问自己：

- 如果以后数据库从 PostgreSQL 换成别的存储，这个规则还成立吗？

如果还成立，它更像业务规则，应放在 logic。

例如：

- 只有 `pending` 才能 `start`
- reviewer 只能 approve 自己的任务
- `running` 不能 cancel

如果这段逻辑依赖：

- `FOR UPDATE`
- `RETURNING`
- `NOW()`
- `sql.NullString`

那它更像存储实现细节，应放在 model。

## 【代码/场景对应】

当前项目中直接对应：

- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
- [internal/logic/tasks/succeedtasklogic.go](../../internal/logic/tasks/succeedtasklogic.go)
- [internal/logic/tasks/failtasklogic.go](../../internal/logic/tasks/failtasklogic.go)
- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)

## 【易错点】

- 以为出现 `sql.NullString` 就一定是分层错误。
- 把 `updated_at` 误认为业务事件时间。
- 把 model 方法签名里没被 SQL 真正消费的参数也当成“合理预留”。

## 【关联知识】

- [事务、Rollback/Commit 与 FOR UPDATE](../database/transactions_and_row_locks.md)
- [时间字段、`NOW()`、`time.Time` 与 RFC3339 的区别](../database/time_fields_now_timestamptz_and_rfc3339.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)

