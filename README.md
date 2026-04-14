# AgentOps

## What Is This

AgentOps 是一个基于 go-zero 的学习型后端项目，当前方向是把“代码任务 / 分析任务”纳入一个可控、可审计、可审批的任务平台骨架。

当前项目重点不是一次性做完整 AI 平台，而是围绕任务主链，逐步熟悉：

- `.api -> goctl -> types / handler / logic`
- go-zero 的分层职责
- PostgreSQL 接入与主链落库
- 状态流转、审计、执行记录
- Git 上下文感知

## Current Scope

当前已经具备这些接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/approve`
- `POST /api/v1/tasks/:id/cancel`
- `GET /api/v1/tasks/:id/logs`
- `GET /api/v1/tasks/:id/executions`

## What Is Real Now

当前这些能力已经接入 PostgreSQL：

- `CreateTask`
  - 插入 `tasks`
  - 插入 `task_policies`
  - 插入最小 `audit_logs`
- `ListTasks`
  - 真实查询 `tasks`
- `GetTask`
  - 真实查询 `tasks + task_policies`
- `ApproveTask`
  - 基于真实状态更新
  - 写 `approval_records`
  - 写 `audit_logs`
- `CancelTask`
  - 基于真实状态更新
  - 写 `audit_logs`
- `GetTaskLogs`
  - 真实查询 `audit_logs`
- `GetTaskExecutions`
  - 真实查询 `task_executions`
- Git 上下文
  - 创建任务时采集 `branch / head commit / dirty`
  - 存入 `tasks`
  - 列表和详情接口可返回这些字段

## Current Data Model

当前最核心的几张表：

- `tasks`
  任务本体
- `task_policies`
  任务策略快照
- `task_executions`
  执行记录
- `approval_records`
  审批记录
- `audit_logs`
  审计日志

关系可以先这样理解：

- `tasks` 是中心
- `task_policies` 当前基本是 `1:1`
- `audit_logs` 是 `1:N`
- `task_executions` 是 `1:N`
- `approval_records` 设计上允许 `1:N`

## Tech Stack

- Go
- go-zero
- goctl
- PostgreSQL
- Git

## Project Structure

- `agentops.api`
  接口契约源头，定义路由、请求结构、响应结构。

- `internal/types`
  由 `.api` 生成的请求/响应结构体，不作为主维护入口。

- `internal/handler`
  HTTP 请求入口，负责接收请求、解析参数、调用 logic、返回响应。

- `internal/logic`
  业务流程层，负责参数校验、状态判断、状态流转、组织响应。

- `internal/model`
  数据访问层，负责 SQL 和数据库读写。

- `internal/svc`
  `ServiceContext` 所在位置，负责共享依赖注入。

- `internal/gitctx`
  Git 上下文读取辅助包，负责识别仓库状态并采集 Git 信息。

- `migrations`
  建表 SQL / migration 文件。

- `etc`
  配置文件目录。

## Current Status Design

当前任务状态：

- `pending`
- `waiting_approval`
- `running`
- `succeeded`
- `failed`
- `cancelled`

当前任务模式：

- `analyze`
- `patch`

含义：

- `analyze`
  分析型任务，偏只读理解
- `patch`
  修改型任务，偏代码改动

## What Is Not Done Yet

当前还没有完全做完的部分：

- 统一错误码与更细的 HTTP 状态映射
- 真实执行器接入
- `task_executions` 的完整写入链
- 更完整的审批人 / 操作者身份
- 更细的策略模型拆分
- CLI 主链接入
- README / 运行说明继续细化

## How To Run

### 1. 启动 PostgreSQL

如果你本地是用 Docker：

```bash
docker start agentops-postgres
