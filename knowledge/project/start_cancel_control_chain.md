# 【主题】
AgentOps 中 `StartTask` 与 `CancelTask` 的执行取消控制链

一句话概述：

L4 的 running cancel 不是简单把状态改成 `cancelled`，而是一条跨 `StartTask`、`CancelTask`、`CancelRegistry`、数据库事务和 executor context 的协作链；数据库行锁决定最终业务状态，context cancel 只负责通知正在运行的 runner 停止。

## 【所属分类】

项目实战 / 当前项目中的关键链路 / 后端并发控制

## 【核心结论】

- `CancelTask` 负责把业务状态收口成 `cancelled`。
- `StartTask` 负责执行 runner，并在收到 `context.Canceled` 后判断这是不是业务 cancel。
- `CancelRegistry` 负责把 `taskID` 和标准库生成的 `cancel` 函数连接起来。
- `context cancel` 是通知机制，不是数据库锁，也不是业务状态。
- `FOR UPDATE` / 数据库事务才是状态一致性的根。
- `CancelTask` 应先提交数据库 `cancelled`，再触发 `cancel()`，避免 `StartTask` 把业务取消误收口成 `failed`。

## 【展开解释】

### 1. 这条链路为什么复杂

`StartTask` 和 `CancelTask` 可能是两个不同 HTTP 请求：

```text
请求 A：StartTask 正在执行 runner
请求 B：CancelTask 同时过来取消 task
```

这两个请求会共享同一个任务状态，但它们不在同一个调用栈里。

因此必须分清三层机制：

- `CancelRegistry` 的 mutex
  - 只保护内存里的 `taskID -> cancel` map
  - 不保护数据库业务状态
- 数据库事务和 `FOR UPDATE`
  - 保护 `tasks` 这一行的状态迁移
  - 决定 `succeeded / failed / cancelled` 谁最终胜出
- `context cancel`
  - 只是通知 runner 停止
  - 不自动改变数据库状态

### 2. 标准控制链

当前目标链路应是：

```text
StartTask:
  1. pending -> running
  2. 创建 runCtx / cancel
  3. 注册 taskID -> cancel
  4. 调用 TaskRunner.Run(runCtx, ...)

CancelTask:
  1. 发现 task 是 running
  2. 校验 actor 是 assigned operator 或 admin
  3. task_executions.status -> cancelled
  4. tasks.status -> cancelled
  5. 写 task_status_histories
  6. 写 audit_logs
  7. commit
  8. 调用 ExecutionCancels.Cancel(taskID)

StartTask 被唤醒:
  1. runner 返回 context.Canceled
  2. 重新查 task 当前状态
  3. 如果已经是 cancelled，返回 cancelled
  4. 否则按普通执行失败处理
```

关键顺序是：

```text
CancelTask 先提交数据库 cancelled
再调用 cancel()
```

如果反过来，`StartTask` 可能先被 `cancel()` 唤醒，但数据库还没提交 `cancelled`，于是误以为 executor 失败，走 `FailTaskLogic`。

### 3. `StartTask` 为什么收到 cancel 后还要查数据库

`context.Canceled` 不一定等于业务取消。

它可能来自：

- 用户主动调用 `CancelTask`
- HTTP 请求被客户端断开
- 上游 context 被取消
- 服务内部主动调用 `cancel()`

所以 `StartTask` 不能只看到：

```go
errors.Is(runErr, context.Canceled)
```

就直接返回 `cancelled`。

它必须重新查数据库：

```go
cancelledTask, findErr := l.svcCtx.TaskModel.FindByID(l.ctx, taskID)
```

只有当前 task 已经是：

```go
TaskStatusCancelled
```

才能把这次 `context.Canceled` 解释成业务取消。

### 4. 为什么还要处理 `ErrTaskNotRunning`

即使 runner 正常成功，`StartTask` 后面调用 `SucceedTask` 时，也可能遇到状态竞争。

例如：

```text
runner 刚执行成功
CancelTask 抢先拿到 task 行锁
CancelTask 写入 cancelled 并提交
StartTask 再调用 SucceedTask
SucceedTask 发现 task 已经不是 running
返回 ErrTaskNotRunning
```

这时不能直接把 `ErrTaskNotRunning` 当成错误抛出。

更合理的是重新查当前 task：

```text
如果 current status 是 cancelled，返回 cancelled
如果 current status 是 failed / succeeded，也按当前终态返回
否则再返回原错误
```

这表示数据库最终状态已经由另一个合法状态迁移决定，`StartTask` 应尊重已经落库的终态。

### 5. 谁负责最终一致性

最终一致性不是靠 registry，也不是靠 context。

真正负责状态一致性的是：

- `FindByIDForUpdate`
- 数据库事务
- `SucceedTask / FailTask / CancelTask` 对状态的二次校验

直观理解：

```text
registry 负责通知
context 负责传播取消信号
数据库行锁负责决定最终状态
logic 层负责解释竞争后的结果
```

## 【代码/场景对应】

当前 AgentOps 中直接对应：

- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
  - 创建 `runCtx / cancel`
  - 注册 `ExecutionCancels.Register(taskID, cancel)`
  - runner 返回 `context.Canceled` 后重新查 task 状态
  - `SucceedTask` 遇到 `ErrTaskNotRunning` 后重新查当前终态
- [internal/logic/tasks/canceltasklogic.go](../../internal/logic/tasks/canceltasklogic.go)
  - running cancel 时先写 execution / task / history / audit
  - `tx.Commit()` 后再调用 `ExecutionCancels.Cancel(taskID)`
- [internal/executor/registry.go](../../internal/executor/registry.go)
  - 保存和触发 `taskID -> context.CancelFunc`
- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
  - `FindByIDForUpdate` 是状态竞争控制的关键
- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)
  - `Finish` 负责把 running execution 收口为 `succeeded / failed / cancelled`

当前阶段边界：

- 已有 registry 和数据库状态收口。
- 已能从 `StartTask` 注册 cancel，并由 `CancelTask` 触发 cancel。
- 仍需要测试验证长时间 runner 能被 cancel 打断。
- 真实 CLI provider 应使用 `exec.CommandContext` 才能做到进程级中断。

## 【易错点】

- 误以为 registry 的锁能保护任务状态。它只保护内存 map。
- 误以为 `context.Canceled` 一定来自业务取消。它也可能来自请求断开或上游 context 取消。
- 误以为 `cancel()` 会自动写数据库。数据库状态必须由 logic 明确写入。
- 误以为先调用 `cancel()` 再写数据库也没问题。这样可能导致 `StartTask` 抢先按失败收口。
- 误以为 `ErrTaskNotRunning` 一定是异常。它也可能是合法状态竞争后的结果，需要重新查当前终态。

## 【关联知识】

- [AgentOps 中 `context.CancelFunc` 与执行取消注册表的边界](./execution_cancel_registry_context.md)
- [AgentOps 中 `task`、`execution` 与 `Finish` 的边界](./task_execution_finish_boundary.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)
- Go `context`
- PostgreSQL `FOR UPDATE`
- 状态机并发控制
- executor runner / CLI 进程中断
