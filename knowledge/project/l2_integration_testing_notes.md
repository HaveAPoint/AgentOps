# 【主题】
AgentOps L2 集成测试为什么这样写

一句话概述：

L2 的关键不是“能编译”，而是事务、状态机、行锁和多表一致性能不能在真实 PostgreSQL 下成立，所以测试采用了集成测试而不是纯单元测试。

## 【所属分类】
项目实战 / 测试

## 【核心结论】

- L2 当前最关键的测试是集成测试，不是纯单元测试。
- 测试文件放在被测 package 下，用 `*_test.go` 命名，是 Go 的常见做法。
- 这轮测试直接调用 logic，并连真实 PostgreSQL，目的是验证事务、状态迁移和并发行为。
- 并发 approve / start 的测试，核心是在验证 `FOR UPDATE` 和单事务语义是否真生效。

## 【展开解释】

### 为什么不只测编译

L2 的核心复杂度在：

- `approve / start / succeed / fail / cancel`
- 状态机是否合法
- `task / execution / history / audit` 是否一致
- 并发时会不会出现两个动作都成功

这些问题只靠：

- `go test ./...`
  - 编译通过

是发现不了的。

### 为什么选择集成测试

当前测试文件：

- [internal/logic/tasks/l2_integration_test.go](../../internal/logic/tasks/l2_integration_test.go)

做了这些事：

- 为每个测试创建独立临时数据库
- 执行 `migrations/0001_init.sql`
- 组装测试用 `ServiceContext`
- 直接调用 logic

这样可以把：

- PostgreSQL
- SQL
- 事务
- 行锁
- 多表一致性

一起测到。

### 为什么不优先走 HTTP

这轮的重点不是：

- 路由
- JSON 解析
- go-zero 框架本身

而是业务主链本身。

所以测试直接调 logic，更能集中验证：

- 参数校验
- 状态校验
- 事务编排
- 表更新顺序

### 测了哪些核心场景

- create 成功
- 非 git 仓库失败
- approve / start / succeed / fail / cancel 的合法与非法状态
- 重复 approve、重复 start、重复 finish
- 并发 approve
- 并发 start

### 测试文件应该放在哪里

Go 的常见实践是：

- 被测目录下放 `*_test.go`

例如：

- `internal/logic/tasks/*.go`
- `internal/logic/tasks/*_test.go`

不是必须单独起一个独立“测试工程”。

## 【代码/场景对应】

当前项目中直接对应：

- [internal/logic/tasks/l2_integration_test.go](../../internal/logic/tasks/l2_integration_test.go)
- [README.md](../../README.md)

## 【易错点】

- 误以为测试一定要独立到单独工程才算正规。
- 误以为编译通过就代表主链行为正确。
- 误以为并发问题可以等后面再测。

## 【关联知识】

- [事务、Rollback/Commit 与 FOR UPDATE](../database/transactions_and_row_locks.md)
- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [`tasks`、`task_executions`、`task_status_histories`、`audit_logs` 的职责区别](./task_snapshot_execution_history_audit.md)

