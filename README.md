# AgentOps

## What Is This

AgentOps 是一个基于 go-zero 的学习型后端项目。
当前目标不是一次性做完整平台，而是通过“任务创建、查询、状态流转、日志与执行记录”这几类接口，逐步熟悉 go-zero 的接口契约、代码生成链路和后端项目骨架。

## Current Scope

当前第一版已经完成这些接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/approve`
- `POST /api/v1/tasks/:id/cancel`
- `GET /api/v1/tasks/:id/logs`
- `GET /api/v1/tasks/:id/executions`

当前这些接口主要用于学习和验证 go-zero 请求链，任务数据仍然以假数据为主，还没有接数据库持久化。

## Not Done Yet

当前第一版还没有完成：

- 数据库存储
- 真实状态持久化
- 完整错误码体系
- 权限控制
- 审批规则 / 策略模块拆分
- 审计模块拆分
- 执行器模块拆分

目前的重点仍然是先把项目结构、职责边界和请求链路理解清楚。

## Tech Stack

- Go
- go-zero
- goctl

## Project Structure

- `agentops.api`
  接口契约源头，定义路由、请求结构、响应结构。

- `internal/types`
  由 `.api` 生成的请求/响应结构体，不作为主维护入口。

- `internal/handler`
  HTTP 请求入口，负责接收请求、解析参数、调用 logic、返回响应。

- `internal/logic`
  业务流程层，负责参数校验、状态判断、假数据返回和后续真实业务逻辑。

- `internal/svc`
  ServiceContext 所在位置，负责管理共享依赖。当前项目依赖较少，后续数据库、客户端、配置等都会从这里接入。

- `etc`
  配置文件目录。

## How To Run

在项目根目录执行：

```bash
go run . -f etc/agentops-api.yaml
默认启动地址：

bash

http://localhost:8888
API Examples
查询任务列表：

bash

curl -i 'http://localhost:8888/api/v1/tasks'
查询任务详情：

bash

curl -i 'http://localhost:8888/api/v1/tasks/task-001'
审批任务：

bash

curl -i -X POST 'http://localhost:8888/api/v1/tasks/task-001/approve' \
  -H 'Content-Type: application/json' \
  -d '{}'
取消任务：

bash

curl -i -X POST 'http://localhost:8888/api/v1/tasks/task-001/cancel' \
  -H 'Content-Type: application/json' \
  -d '{}'
查询任务日志：

bash

curl -i 'http://localhost:8888/api/v1/tasks/task-001/logs'
查询任务执行记录：

bash

curl -i 'http://localhost:8888/api/v1/tasks/task-001/executions'
go-zero Skeleton Notes
这个项目当前最重要的认知不是“接口数量”，而是 go-zero 的项目骨架：

.api -> goctl -> types / handler / logic
.api 是接口契约源头
修改 .api 后，应该重新生成代码
types.go 是生成物，不应该被当成主维护对象
handler 负责请求入口，不负责写业务规则
logic 负责业务流程，不负责处理 HTTP 细节
ServiceContext 用来承载共享依赖
NewXxxLogic(ctx, svcCtx) 是 go-zero 里常见的业务逻辑模板
Current Learning Summary
当前已经摸清的 3 类接口视角：

查询类接口

ListTasks
GetTask
动作类接口

ApproveTask
CancelTask
治理 / 审计类接口

GetTaskLogs
GetTaskExecutions
对应的核心理解：

查询接口重点是取参和组织响应
动作接口重点是表达状态变化
日志和执行记录不是任务本体，而是任务过程的两个不同视角
Next Step
后续计划逐步增加：

真实任务存储
状态持久化
更清晰的错误定义
policy / audit / executor 模块拆分
更像真实平台后端的任务生命周期管理