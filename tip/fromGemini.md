# Code Review & Architecture Tips from Gemini

*Date: 2026-04-18*

This document summarizes a set of "Bad Smells" and architectural critiques regarding the current Go backend codebase, specifically focusing on domain modeling boundaries and engineering details.

### 1. 业务逻辑层严重“被数据库绑架”（抽象泄漏）
在 `internal/logic/tasks/createtasklogic.go` 和 `starttasklogic.go` 里，业务逻辑层直接引入了 `database/sql`，并大量充斥着这样的代码：
```go
tx, err := l.svcCtx.DB.BeginTx(l.ctx, &sql.TxOptions{})
// ...
ReviewerId: sql.NullString{
    String: reviewerID,
    Valid:  reviewerID != "",
},
```
* **问题**：`logic` 层是纯粹的业务领域层，它**不应该知道什么是 SQL，更不应该知道什么是 `sql.NullString` 或 `sql.Tx`**。
* **影响**：如果以后想把底层数据库换成 MongoDB，或者想把某个存储切到 Redis，`logic` 层要重写一半。
* **改进建议**：应该在 `model` 层实现 `UnitOfWork`（工作单元模式）或者提供一个 `RunInTx(ctx, fn func(ctx context.Context) error)` 闭包，让 `logic` 层只需要处理纯 Go 的基本类型（如 `*string`），把 `sql.NullString` 的转换等数据库特定逻辑封装在 `model` 层里。

### 2. “魔法字符串（Magic Strings）”满天飞
在 `createtasklogic.go` 和 `starttasklogic.go` 中，给 `TaskStatus` 定义了常量（如 `TaskStatusPending`），这很好。但是，在写入 `TaskStatusHistory` 和 `AuditLog` 时，却用了硬编码：
```go
Action:     "create",       // 魔法字符串
ActorRole:  "creator",      // 魔法字符串
Level:      "info",         // 魔法字符串
ToolName:   "api",          // 魔法字符串
```
* **问题**：在底层 `migrations/0001_init.sql` 里写了死板的 `CHECK (actor_role IN ('creator', ...))`，但代码里却用硬编码字符串。
* **影响**：如果哪天手抖敲成了 `"Creator"`（大写），编译不会报错，运行时直接数据库报错宕机。
* **改进建议**：在 `internal/logic/tasks/taskconsts.go` 里把这些字典值全部定义为 `const`（甚至自定义类型 `type ActorRole string`），利用 Go 的类型系统杜绝手抖。

### 3. 事务回滚的样板代码过于原始
事务都是这么写的：
```go
tx, err := l.svcCtx.DB.BeginTx(l.ctx, &sql.TxOptions{})
defer func() { _ = tx.Rollback() }()
// ...一堆 insert...
if err = tx.Commit(); err != nil { return nil, err }
```
* **问题**：这是最基础的 Go 事务写法。虽然 `Commit` 成功后 `Rollback` 会返回 `sql.ErrTxDone` 被忽略，但这是一种“防御性妥协”。它没有处理 `panic` 的情况——如果中间的代码抛出了 panic，虽然 `defer` 会触发回滚，但整个进程如果没有 `recover` 就会直接崩溃，且无法在回滚时记录相关的错误日志。
* **改进建议**：像这种大量需要跨表写入的场景，最好的做法是封装一个事务执行器（Transaction Runner / UoW），把 `defer`、`panic recover`、`rollback`、日志统一包起来。

### 4. 贫血的错误处理 (Anemic Error Handling)
`taskerrors.go` 里面全是扁平的 `errors.New(...)`：
```go
ErrTaskNotFound = errors.New("task not found")
```
* **问题**：这种扁平的错误定义适合写底层库，但不适合写业务 API。当 `l.svcCtx.TaskModel.Insert` 因为数据库连接断开报错时，直接 `return nil, err`。
* **影响**：最外层的 Handler 拿到的只是一个冰冷的 `connection refused`。如果是复杂的并发调用，根本不知道这个错误是在哪一步（插入 Task 还是插入 Policy）发生的。
* **改进建议**：业务中应该使用类似 `fmt.Errorf("failed to insert task: %w", err)` 进行错误包装（Error Wrapping），保留完整的错误上下文链路，方便排查定位问题。

---
**总体评价**：项目的基础底子（测试、数据库约束）非常硬，但在代码实现上带有明显的“按部就班写 CRUD”的痕迹，缺少为了**长期可维护性**而做的优雅封装。
