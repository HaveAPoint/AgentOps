# 【主题】
AgentOps 中 `context.CancelFunc` 与执行取消注册表的边界

一句话概述：

`context.WithTimeout` 生成的 `cancel` 函数是 Go 标准库提供的取消入口；AgentOps 的 `CancelRegistry` 不负责实现取消，只负责把运行中任务的 `taskID -> cancel` 保存起来，让 `CancelTask` 能找到并触发正在执行的 runner 停止。

## 【所属分类】

项目实战 / Go context / 后端并发控制

## 【核心结论】

- `cancel` 函数来自 Go 标准库 `context` 包，不是业务代码手写出来的。
- `context.WithTimeout(parent, timeout)` 会返回一个新的 `Context` 和一个 `context.CancelFunc`。
- `CancelRegistry` 保存的是 `taskID -> cancel function`，不是保存完整 task，也不是保存完整 context。
- `StartTask` 负责创建 `runCtx / cancel` 并注册 cancel。
- `CancelTask` 负责根据 `taskID` 找到 cancel 并调用。
- executor runner 必须尊重 `ctx.Done()`，否则调用 cancel 也不能真正中断执行。
- 当前内存 registry 只适合单进程 MVP；多实例 / 多 worker 场景需要更强的协调机制。

## 【展开解释】

### 1. `cancel` 从哪里来

Go 标准库的 `context.WithTimeout` 会返回两个值：

```go
runCtx, cancel := context.WithTimeout(parentCtx, timeout)
```

其中：

- `runCtx` 是一个新的 context，带有超时时间。
- `cancel` 是一个 `context.CancelFunc`。

`CancelFunc` 的类型本质是：

```go
type CancelFunc func()
```

但它不是普通的业务函数。它内部绑定了由 `WithTimeout` 创建出来的 `runCtx`。

调用：

```go
cancel()
```

会让 `runCtx.Done()` 变成可读，并让：

```go
runCtx.Err()
```

返回：

```go
context.Canceled
```

如果不是手动取消，而是超过 timeout，则通常返回：

```go
context.DeadlineExceeded
```

### 2. 为什么需要 `CancelRegistry`

`StartTask` 创建 `runCtx / cancel` 的位置，和 `CancelTask` 收到取消请求的位置，不在同一个函数调用栈里。

如果没有共享机制：

- `StartTask` 手里有 `cancel`
- `CancelTask` 不知道这个 `cancel` 在哪里

因此需要一个运行中任务表：

```text
taskID -> cancel function
```

这就是 `CancelRegistry` 的职责。

它不实现取消逻辑，只保存和取出标准库生成的 cancel 函数。

### 3. 标准链路

理想链路是：

```text
StartTask 创建 runCtx / cancel
StartTask 注册 taskID -> cancel
StartTask 调用 TaskRunner.Run(runCtx, ...)
CancelTask 根据 taskID 找到 cancel 并调用
runner 监听 ctx.Done() 后停止
StartTask 结束后注销 taskID -> cancel
```

代码形态通常是：

```go
runCtx, cancel := context.WithTimeout(l.ctx, executionTimeout)
l.svcCtx.ExecutionCancels.Register(taskID, cancel)
defer l.svcCtx.ExecutionCancels.Unregister(taskID)
defer cancel()
```

`defer cancel()` 负责释放 context 资源。

`defer Unregister(...)` 负责避免已经结束的任务继续留在 registry 里。

### 4. 为什么 registry 需要锁

`CancelRegistry` 内部通常是：

```go
map[int64]context.CancelFunc
```

但 Go 的 map 不是并发安全的。

在 AgentOps 里，至少存在两类并发访问：

- `StartTask` 注册 / 注销 cancel
- `CancelTask` 查找 / 调用 cancel

这些可能发生在不同 goroutine，所以 registry 需要 `sync.Mutex` 保护 map。

### 5. `cancel()` 不等于强制杀死进程

调用 `cancel()` 只是在 context 层面发出取消信号。

真正能不能停止，取决于 runner 是否尊重 context。

例如 mock runner 可以这样检查：

```go
select {
case <-ctx.Done():
	return executor.Result{}, ctx.Err()
default:
}
```

真实 CLI runner 则应使用：

```go
exec.CommandContext(ctx, ...)
```

这样 context 被取消时，命令进程才有机会被终止。

如果 runner 完全不检查 `ctx.Done()`，也不用 `CommandContext`，那么 registry 调用 cancel 也只能改变 context 状态，不能真正停止执行。

## 【代码/场景对应】

当前 AgentOps 中直接对应：

- [internal/executor/registry.go](../../internal/executor/registry.go)
  - 保存 `taskID -> context.CancelFunc`
  - 提供 `Register / Cancel / Unregister`
- [internal/svc/servicecontext.go](../../internal/svc/servicecontext.go)
  - 持有 `ExecutionCancels *executor.CancelRegistry`
  - 在 `NewServiceContext` 中初始化 registry
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
  - 当前已创建 `runCtx, cancel := context.WithTimeout(...)`
  - 后续应在这里注册和注销 cancel
- [internal/logic/tasks/canceltasklogic.go](../../internal/logic/tasks/canceltasklogic.go)
  - 当前已支持 `running -> cancelled` 的数据库状态收口
  - 后续应在 running cancel 分支调用 registry 触发 cancel

当前阶段边界：

- 已有 registry 地基。
- `running -> cancelled` 已能做数据库状态收口。
- 还没有把 `StartTask` 的 cancel 注册进 registry。
- 还没有让 `CancelTask` 调用 registry。
- 还没有真实 CLI provider，因此还没有完整进程级中断能力。

## 【易错点】

- 误以为 `cancel` 是业务代码自己实现的。实际它来自标准库 `context.WithTimeout / context.WithCancel`。
- 误以为 registry 保存的是完整 context。实际它保存的是 cancel 函数。
- 误以为调用 `cancel()` 就一定能杀死执行。实际 runner 必须检查 `ctx.Done()` 或使用 `exec.CommandContext`。
- 误以为数据库写成 `cancelled` 就等于真实执行已经中断。数据库收口和执行中断是两层能力。
- 误把 registry 放到 `StartTaskLogic` 局部变量里。局部变量无法被另一个 `CancelTask` 请求找到。
- 忘记 `Unregister`。执行结束后不注销会留下旧 cancel 函数，造成内存泄漏和误导。

## 【关联知识】

- Go 标准库 `context`
- `context.WithTimeout`
- `context.CancelFunc`
- `ctx.Done()` / `ctx.Err()`
- `context.Canceled` / `context.DeadlineExceeded`
- `sync.Mutex`
- `exec.CommandContext`
- AgentOps `StartTask` / `CancelTask` 执行生命周期
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)
- [AgentOps 中 `task`、`execution` 与 `Finish` 的边界](./task_execution_finish_boundary.md)
