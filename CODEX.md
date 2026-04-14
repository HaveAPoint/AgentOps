# CODEX.md

## 项目名称

AgentOps Platform

## 一句话定义

一个基于 go-zero 的、任务主链优先的受控任务平台后端。

它的核心不是“再造一个 coding agent”，而是把代码任务、分析任务纳入一个可控、可审计、可审批、可追踪的后端平台骨架。

---

## 当前项目定位

### 本项目是什么

这是一个控制平面（control plane）项目。

当前平台负责：

- 用户认证
- 任务生命周期管理
- 策略快照持久化
- 审批流
- 审计日志
- 执行记录查询
- Git 上下文感知

### 本项目不是什么

本项目当前不是：

- 通用聊天产品
- 大而全的 AI 平台
- 完整 agent framework
- 微服务体系演示项目
- 前端优先产品

---

## 当前技术选型

- 语言：Go
- 框架：go-zero
- 接口契约：`.api` + `goctl`
- 数据库：PostgreSQL
- 版本管理上下文：Git（当前先做只读感知）

后续方向：

- CLI
- 外部执行器 / Codex 接入

---

## 开发总原则

1. 先闭环，后扩展
2. 先真实主链，后抽象优化
3. 先平台治理，后执行增强
4. 接口结构变更先改 `.api`
5. 不直接把生成文件当成主设计文件
6. 平台逻辑、策略逻辑、执行逻辑必须分开
7. 当前阶段优先保持结构清晰，不追求过早泛化

---

## go-zero 相关规则

### 1. `.api` 是源头

请求体和响应体的结构，以 `.api` 为准。

如果接口字段发生变化：

1. 修改 `.api`
2. 重新执行 `goctl api go -api agentops.api -dir .`
3. 再调整 handler / logic / model

不要把 [internal/types/types.go](/Users/huahuaclaw/agentops/internal/types/types.go) 当作长期手工维护文件。

### 2. handler 必须薄

handler 只负责：

- 接收请求
- 参数绑定
- 调用 logic
- 返回响应

handler 不负责：

- 业务规则
- 状态流转
- SQL
- 审批判定
- 策略判断

### 3. logic 负责业务流程

logic 负责：

- 业务语义校验
- 状态流转
- 协调 model / audit / approval / gitctx

logic 不负责：

- 直接堆大量 SQL
- 处理 HTTP 细节
- 演化成杂物堆

### 4. model 负责数据访问

model 负责：

- SQL
- 单表或单类数据访问方法
- 数据库读写细节

model 不负责：

- 业务状态机
- 审批语义
- 跨多表的业务编排

### 5. ServiceContext 是共享依赖入口

当前通过 `svc.ServiceContext` 注入：

- PostgreSQL 连接
- 配置
- Task / Policy / Audit / Approval / Execution model

后续可继续注入：

- Executor adapter
- 缓存
- 其他共享资源

---

## 当前主链

当前已经真实接通的主链是：

1. 用户登录
2. 创建任务
3. 任务与策略落库
4. 写入最小审计日志
5. 查询任务列表 / 详情
6. 审批任务
7. 取消任务
8. 查询日志
9. 查询执行记录
10. 采集 Git 上下文并回传

---

## 当前真实能力

当前已经完成：

- `POST /api/v1/auth/login`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/approve`
- `POST /api/v1/tasks/:id/cancel`
- `GET /api/v1/tasks/:id/logs`
- `GET /api/v1/tasks/:id/executions`

其中任务主链已经接入 PostgreSQL，主要表包括：

- `tasks`
- `task_policies`
- `task_executions`
- `approval_records`
- `audit_logs`

---

## 当前未完成能力

当前还没有完全做完：

- 统一错误码体系
- 更细的 HTTP 状态映射
- 真实执行器写入链
- CLI 主链接入
- 更完整的审批人身份
- 更复杂的策略模型拆分
- 更结构化的状态变更历史

---

## 核心业务对象

当前系统围绕这些对象建设：

1. User
2. Task
3. TaskPolicy
4. TaskExecution
5. ApprovalRecord
6. AuditLog

### 说明

- `Task`
  任务主体
- `TaskPolicy`
  策略快照（allow/deny、审批要求等）
- `TaskExecution`
  一次执行记录
- `ApprovalRecord`
  审批动作记录
- `AuditLog`
  审计日志 / 过程追踪

---

## 状态设计

当前允许的任务状态只有：

- `pending`
- `waiting_approval`
- `running`
- `succeeded`
- `failed`
- `cancelled`

禁止随意扩状态。

---

## Mode 设计

当前只允许两种 mode：

- `analyze`
- `patch`

含义：

- `analyze`
  只读分析类任务
- `patch`
  修改类任务

---

## Git 集成规则

当前只做 Git 只读感知，不做 Git 自动化。

当前采集：

- 当前 HEAD 的可读引用
- 当前 HEAD commit
- working tree 是否 dirty

说明：

- API 字段当前仍使用 `gitBranch`
- 但在 detached HEAD 场景下，它不一定是真正分支名，更接近“HEAD 的可读引用”
- 非 Git 仓库和“无首个 commit”的仓库，当前都应返回业务错误，而不是直接暴露底层 git 报错

---

## 目录职责

- [agentops.api](/Users/huahuaclaw/agentops/agentops.api)
  接口契约源头

- [internal/types](/Users/huahuaclaw/agentops/internal/types)
  从 `.api` 生成的请求/响应结构

- [internal/handler](/Users/huahuaclaw/agentops/internal/handler)
  请求入口

- [internal/logic](/Users/huahuaclaw/agentops/internal/logic)
  业务流程控制

- [internal/svc](/Users/huahuaclaw/agentops/internal/svc)
  共享依赖聚合

- [internal/model](/Users/huahuaclaw/agentops/internal/model)
  数据访问

- [internal/gitctx](/Users/huahuaclaw/agentops/internal/gitctx)
  Git 上下文读取辅助包

- [migrations](/Users/huahuaclaw/agentops/migrations)
  建表 SQL / migration

---

## PostgreSQL 路线

数据库统一使用 PostgreSQL。

当前第一批核心表：

- `tasks`
- `task_policies`
- `task_executions`
- `approval_records`
- `audit_logs`

当前 PostgreSQL 使用原则：

1. 第一版优先简单、可查、可更新
2. 主键先用 `bigserial`
3. 插入后获取主键用 `RETURNING id`
4. 不急着引入 ORM
5. 先写清晰 SQL，后续再抽象

---

## AI / Codex 协作规则

后续继续开发时，优先遵循这个闭环：

1. 明确当前任务目标
2. 确认影响范围
3. 判断是否涉及外部契约变更
4. 若涉及契约，先改 `.api`
5. 再改 logic / model / 其他实现
6. 跑验证命令（build / curl / SQL）
7. 根据结果修正
8. 更新 README / CODEX / 状态说明

---

## 当前阶段判断标准

如果一个改动不能增强下面这些主线之一，默认先不做：

- 任务主链
- 状态流转
- 审批与审计
- 执行记录
- Git 上下文
- PostgreSQL 真实化

当前阶段优先的是“真实主链闭环”，不是功能数量。
