# AgentOps

## What Is This

AgentOps 是一个基于 go-zero 的学习型后端项目，当前方向是把“代码任务 / 分析任务”纳入一个可控、可审计、可审批的任务平台骨架。

当前项目重点不是一次性做完整 AI 平台，而是围绕任务主链，逐步熟悉：

- `.api -> goctl -> types / handler / logic`
- go-zero 的分层职责
- PostgreSQL 接入与主链落库
- 状态流转、审计、执行记录
- Git 上下文感知
- JWT 登录、真实 actor 与权限控制

## Current Scope

当前已经具备这些接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/approve`
- `POST /api/v1/tasks/:id/start`
- `POST /api/v1/tasks/:id/succeed`
- `POST /api/v1/tasks/:id/fail`
- `POST /api/v1/tasks/:id/cancel`
- `GET /api/v1/tasks/:id/logs`
- `GET /api/v1/tasks/:id/executions`
- `GET /api/v1/tasks/:id/status-histories`

## What Is Real Now

当前这些能力已经接入 PostgreSQL：

- `Login`
  - 真实查询 `users`
  - 校验 `password_hash`
  - 签发 JWT
- `CreateTask`
  - 插入 `tasks`
  - 插入 `task_policies`
  - 插入最小 `audit_logs`
  - `creator_id` 来自当前登录用户
- `ListTasks`
  - 真实查询 `tasks`
  - 按当前用户可见性过滤
- `GetTask`
  - 真实查询 `tasks + task_policies`
  - 不可见任务返回不存在语义
- `ApproveTask`
  - `waiting_approval -> pending`
  - 写 `approval_records`
  - 写 `task_status_histories`
  - 写 `audit_logs`
  - reviewer 来自当前登录用户和任务归属
- `StartTask`
  - `pending -> running`
  - 写 `task_executions`
  - 写 `task_status_histories`
  - 写 `audit_logs`
  - operator 来自当前登录用户和任务归属
  - 校验 repo 白名单与任务路径策略
  - 同步调用 mock executor
  - executor 成功后自动收口到 `succeeded`
  - executor 失败或新增越权变更文件后自动收口到 `failed`
  - execution result 会记录 executor summary / stdout / stderr 与 Git changed files 摘要
- `SucceedTask`
  - `running -> succeeded`
  - 收口 `task_executions`
  - 写 `task_status_histories`
  - 写 `audit_logs`
  - operator 来自当前登录用户和执行记录归属
- `FailTask`
  - `running -> failed`
  - 收口 `task_executions`
  - 写 `task_status_histories`
  - 写 `audit_logs`
  - 失败原因写入执行记录、状态历史和审计日志
- `CancelTask`
  - 允许 `waiting_approval / pending / running -> cancelled`
  - 写 `cancelled_by / cancelled_at`
  - running cancel 会同步把 running execution 收口为 `cancelled`
  - 写 `task_status_histories`
  - 写 `audit_logs`
  - `waiting_approval / pending` 由 creator / assigned reviewer / admin 取消
  - `running` 由 assigned operator / admin 取消
  - 当前只是数据库状态收口，还没有真实中断正在运行的 executor / CLI 进程
- `GetTaskLogs`
  - 真实查询 `audit_logs`
  - 按任务可见性控制
- `GetTaskExecutions`
  - 真实查询 `task_executions`
  - 按任务可见性控制
- `GetTaskStatusHistories`
  - 真实查询 `task_status_histories`
  - 按任务可见性控制
- Git 上下文
  - 创建任务时采集 `branch / head commit / dirty`
  - 存入 `tasks`
  - 列表和详情接口可返回这些字段
  - 执行时可采集 changed files 的 before / after / new 摘要

## Current Data Model

当前最核心的几张表：

- `tasks`
  任务本体
- `users`
  真实用户、登录名、密码哈希和系统角色
- `task_policies`
  任务策略快照
- `task_executions`
  执行记录
- `approval_records`
  审批记录
- `task_status_histories`
  结构化状态迁移历史
- `audit_logs`
  审计日志

关系可以先这样理解：

- `tasks` 是中心
- `users` 约束任务归属、审批、执行和状态历史 actor
- `task_policies` 当前基本是 `1:1`
- `audit_logs` 是 `1:N`
- `task_executions` 是 `1:N`
- `approval_records` 设计上允许 `1:N`
- `task_status_histories` 是 `1:N`

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

当前固定语义：

- `approve` 只表示“允许执行”，不会直接进入 `running`
- `start` 才是真正开始执行；当前 L4 MVP 是同步 mock 执行，接口可能直接返回 `succeeded / failed`
- `cancel` 支持 `waiting_approval / pending / running -> cancelled`；running cancel 当前只保证数据库收口，不代表真实进程中断
- 当前用户来自 JWT auth context，不来自请求体
- `creator_id / reviewer_id / operator_id` 表达任务归属
- 系统角色包括 `admin / reviewer / operator / viewer`
- 查询不可见任务返回不存在语义，避免暴露任务存在性

## What Is Not Done Yet

当前还没有完全做完的部分：

- 统一错误码与更细的 HTTP 状态映射
- 真实 `codex / claudecode` 执行器接入
- 更细的策略模型拆分
- reviewer 审批语义升级为 operator / mode / repo / path 范围审批
- running 状态下的 cancel 中断执行器
- 工业级 Git diff / patch 范围校验
- CLI 主链接入
- README / 运行说明继续细化

## L3 Verification

当前已经有一组最小行为测试，覆盖：

- create 成功 / 非 git 仓库失败
- approve / start / succeed / fail / cancel 的核心合法与非法状态
- 重复 approve、重复 start、重复 finish
- 并发 approve、并发 start
- 密码哈希与校验
- actor context 读取

测试入口：

```bash
GOCACHE=/tmp/agentops-gocache go test ./internal/auth -count=1
GOCACHE=/tmp/agentops-gocache go test ./internal/logic/tasks -run TestL2 -count=1
```

说明：

- tasks 集成测试会连接本机 PostgreSQL
- 会为每个测试创建独立临时数据库并执行 `migrations/0001_init.sql` 和 `migrations/0003_l3_auth_and_visibility.sql`
- 测试库会 seed L3 用户，logic 测试通过 actor context 注入当前用户
- 默认连接参数与 `etc/agentops-api.yaml` 一致，可通过 `AGENTOPS_TEST_PG_*` 环境变量覆盖

## How To Run

### 1. 启动 PostgreSQL

如果你本地是用 Docker：

```bash
docker start agentops-postgres
