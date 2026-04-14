# codex.md

## 项目名称

AgentOps Platform

## 一句话定义

一个基于 go-zero 的、CLI 优先的受控任务平台后端。

它的核心不是“再造一个 coding agent”，而是“把代码任务、分析任务纳入一个可控、可审计、可审批的后端平台”。

---

## 项目定位

### 本项目是什么

这是一个 **control plane（控制平面）** 项目。

平台负责：

- 用户认证
- 任务生命周期管理
- 策略约束
- 审批流
- 审计日志
- 执行记录查询
- Git 上下文感知

执行器负责：

- 真正执行代码分析/修改任务
- 返回执行结果
- 返回日志和摘要

### 本项目不是什么

本项目不是：

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
- 客户端：CLI
- 执行器：先 mock，后接外部执行器（Codex / Claude Code）
- 版本管理上下文：Git（先只读感知，后续再逐步增强）

---

## 开发总原则

1. 先闭环，后扩展
2. 先平台治理，后执行增强
3. 先 CLI，后 Web
4. 先真实主链，后优化抽象
5. 接口结构变更先改 `.api`，再重新生成
6. 不直接把生成文件当成主设计文件
7. 平台逻辑、策略逻辑、执行器逻辑必须分开

---

## go-zero 相关规则

### 1. `.api` 是源头

请求体和响应体的结构，以 `.api` 为准。

如果接口字段发生变化：

1. 修改 `.api`
2. 重新执行 `goctl api go -api xxx.api -dir .`
3. 再调整 logic / handler / 其他业务代码

不要把 `internal/types/types.go` 当作长期手工维护文件。go-zero 的参数规则支持 `optional`、`default`、`options` 等 DSL 扩展。:contentReference[oaicite:1]{index=1}

### 2. handler 必须薄

handler 只负责：

- 接收请求
- 参数绑定
- 调用 logic
- 返回响应

handler 不能负责：

- 业务规则
- 状态流转
- SQL
- 审批判定
- 策略判断

### 3. logic 负责业务流程

logic 负责：

- 业务语义校验
- 状态流转
- 协调 model / policy / audit / executor / git

logic 不负责：

- 直接写大量 SQL
- 处理 HTTP 细节
- 演化成杂物堆

### 4. ServiceContext 是共享依赖入口

后续这些都应通过 `svc.ServiceContext` 注入：

- PostgreSQL 连接
- 配置
- Git reader
- Executor adapter
- 可能的缓存或其他共享资源

---

## 外部执行器接入约定

当平台开始接真实执行能力时，`Codex` 和 `Claude Code` 都应被视为：

- 外部执行器
- 统一 executor 接口下的不同 provider
- 平台控制面之外的受控执行单元

接入原则：

1. 业务层不直接调用具体 CLI
2. `logic` 只面向统一 `executor` 接口编排流程
3. `executor` 适配层负责屏蔽 `codex` / `claudecode` 的命令差异
4. 状态流转、审批校验、权限校验、审计写入仍然属于平台，不下沉到执行器
5. 执行器只负责执行、日志回传、结果归一化，不负责业务判定

建议 provider 形态：

- `mock`：本地开发与联调
- `codex`：调用 Codex CLI 或对应适配入口
- `claudecode`：调用 Claude Code CLI 或对应适配入口

建议调用链：

1. 平台完成任务状态与策略校验
2. 平台创建 execution 记录
3. `logic` 调用统一 `executor adapter`
4. adapter 选择 `mock / codex / claudecode`
5. adapter 启动外部执行器并捕获 stdout / stderr / 退出状态
6. 平台根据归一化结果更新 execution、task、audit、history

边界要求：

- 不允许在 task logic 中直接拼接 `codex` 或 `claudecode` 命令
- 不允许把数据库写入责任塞进 executor provider
- 不允许让 provider 自己决定审批是否通过或路径是否越权
- repo / path / mode / actor 的约束必须先由平台决定，再交给执行器执行

目录建议：

- `internal/executor/interface.go`
  定义统一执行接口与通用结果结构
- `internal/executor/mock/`
  mock provider
- `internal/executor/codex/`
  Codex provider
- `internal/executor/claudecode/`
  Claude Code provider

如果后续要支持更多执行器，也必须继续挂在统一 adapter 之下，而不是把平台写成某一个具体工具的硬编码壳。

---

## Codex / AI 修改工作流

所有 AI / Codex 参与本项目开发时，都应按这个闭环执行：

1. 先明确当前任务目标
2. 确认影响范围
3. 先计划
4. 再修改代码
5. 跑验证命令（build / test / curl / lint）
6. 根据结果修正
7. 更新文档 / 状态说明

这是当前 Codex 官方强调的高质量任务循环：计划、编辑代码、跑工具、观察结果、修复失败、更新文档/状态。:contentReference[oaicite:2]{index=2}

---

## 项目边界

## v1 必做能力

- 登录
- 创建任务
- 查询任务列表
- 查询任务详情
- 审批任务
- 取消任务
- 查询任务日志
- 查询执行记录
- CLI 命令入口
- PostgreSQL 落库
- Git 基础上下文感知（只读）

## v1 明确不做

- Web 页面
- 微服务拆分
- 多执行器编排
- 自动合并 patch
- 自动 push / PR
- 复杂 RBAC
- RAG / 知识库
- 多 agent 协作
- 泛化工作流引擎
- 完整自动回滚引擎

---

## 核心业务对象

当前系统围绕以下对象建设：

1. User
2. Task
3. TaskPolicy
4. TaskExecution
5. ApprovalRecord
6. AuditLog

### 说明

- `Task`：任务主体
- `TaskPolicy`：策略快照（allow/deny、mode、审批要求、步数等）
- `TaskExecution`：一次任务执行记录
- `ApprovalRecord`：审批动作记录
- `AuditLog`：过程追踪与审计日志

---

## 核心主链

当前系统一切功能都必须围绕这一条主链：

1. 用户登录
2. 创建任务
3. 平台校验策略
4. 若高风险则进入 `waiting_approval`
5. 审批通过后进入 `running`
6. 执行器返回结果
7. 平台记录 execution 与 audit
8. 任务进入 `success / failed / cancelled`

如果某个功能不能增强这条链，默认不做。

---

## 状态设计

当前允许的任务状态只有：

- `pending`
- `waiting_approval`
- `running`
- `success`
- `failed`
- `cancelled`

禁止随意扩状态。

---

## mode 设计

当前只允许两种 mode：

- `analyze`
- `patch`

含义：

- `analyze`：只读分析类任务
- `patch`：修改类任务

---

## 目录职责

- `api/`  
  接口契约定义

- `internal/handler/`  
  请求入口

- `internal/logic/`  
  业务流程控制

- `internal/svc/`  
  共享依赖聚合

- `internal/model/`  
  数据访问

- `internal/types/`  
  从 `.api` 生成的请求/响应结构

计划新增：

- `internal/policy/`  
  策略判断：allow/deny、风险等级、审批要求

- `internal/executor/`  
  执行器适配层

- `internal/audit/`  
  审计记录封装

- `internal/git/`  
  Git 只读上下文采集

---

## 数据库路线

数据库统一使用 PostgreSQL。

### 第一批表

- `tasks`
- `task_policies`
- `task_executions`
- `approval_records`
- `audit_logs`

### PostgreSQL 设计原则

1. 第一版优先简单、可查、可更新
2. 主键可先用 `bigserial`
3. 插入后获取主键用 `RETURNING id`
4. 不急着引入 ORM
5. 先用清晰 SQL，后续再抽象

---

## Git 集成路线

### v1

只做 Git 感知，不做 Git 自动化。

需要感知：

- 当前 branch
- 当前 HEAD commit
- 工作区是否 dirty

### v1.2+

再考虑：

- diff 摘要
- 变更文件列表
- 更完整上下文记录

### 暂不做

- 自动 commit
- 自动 push
- 自动开 PR
- 自动回滚 Git 历史

---

## 第一周目标（已完成 / 已进入尾声）

目标：打通假数据闭环，理解 go-zero 骨架。

### Day 1
- 起 go-zero 骨架
- 替换 `.api`
- 跑通服务

### Day 2
- 完成登录链
- 理解 handler / logic / svc / types

### Day 3
- 完成创建任务链第一版
- 先假数据返回，不急着落库

### Day 4
- 完成任务列表、任务详情
- 继续用假数据打通查询链

### Day 5
- 完成任务审批、任务取消
- 理顺状态流转

### Day 6
- 完成日志查询、执行记录查询
- 明确 Task / Execution / Audit 的区别

### Day 7
- 整理错误返回
- 整理目录职责
- 整理命名
- 补 README 初版
- 写清 go-zero 骨架认知

---

## 第二周目标

目标：把第一周的假数据平台，推进成 PostgreSQL 驱动的真实任务平台雏形。

### Day 8：表设计 + PostgreSQL 接入

完成：

- 设计五张核心表
- 配置 PostgreSQL 连接
- 注入 ServiceContext
- 确定 `tasks`、`task_policies` 为第一批落库对象

### Day 9：CreateTask 真实落库

完成：

- `CreateTask` 插入 `tasks`
- 同时插入 `task_policies`
- 根据输入生成初始状态
- 补一条最小审计日志

### Day 10：ListTasks / GetTask 查库

完成：

- `ListTasks` 改成查 `tasks`
- `GetTask` 改成查 `tasks + task_policies`

### Day 11：ApproveTask / CancelTask 状态更新

完成：

- 合法状态检查
- 更新状态
- 写审批记录
- 写审计日志

### Day 12：Logs / Executions 真实雏形

完成：

- `GetTaskLogs` 读 `audit_logs`
- `GetTaskExecutions` 读最简 `task_executions`

### Day 13：Git 上下文感知

完成：

- 创建任务时采集 branch / HEAD / dirty
- 存入任务或执行上下文

### Day 14：第二周收口

完成：

- 整理 SQL / migration
- 整理 README 第二版
- 整理 model / logic 边界
- 总结当前能力边界

---

## 第三周目标

目标：开始把平台做得更像一个“受控任务平台”，而不是简单 CRUD。

### 重点

- 把策略判断抽到 `internal/policy`
- 把审计写入抽到 `internal/audit`
- 开始补最简错误码体系
- 开始形成执行器适配层骨架
- CLI 初版开始接真实接口

### 交付物

- `policy` 包骨架
- `audit` 包骨架
- CLI `login / create / list / get`
- README 第三版

---

## 第四周目标

目标：建立平台自举能力，让这个系统开始管理自身迭代。

### 重点

- 任务模板
- 任务参数约束更清晰
- 平台记录“开发本平台”的任务
- 执行器接入从 mock 走向真实适配

### 交付物

- 任务模板雏形
- 平台内创建“开发本平台”的任务
- 执行器 adapter 雏形
- 更完整的审计与日志视图

---

## 后续版本路线

### v1
CLI 优先的受控任务平台后端，具备真实落库、状态流转、审批、审计、Git 上下文。

### v1.1
任务模板化：
- 修 bug
- 加接口
- 重构目录
- 补文档

### v1.2
平台开始管理自身后续开发任务。

### v1.3
加入 diff 摘要、风险分级、关键路径审批。

### v1.4
加入测试验证门：
- go test
- build
- lint

### v1.5
形成“平台迭代平台”的正式闭环：
- 创建开发任务
- 限定可修改目录
- 执行器尝试完成
- 跑验证
- 生成人工可审阅结果
- 决定是否接受

---

## 当前编码规则

1. 一次改动只解决一个明确问题
2. 不做顺手的大重构
3. 不引入和当前目标无关的新库
4. 不提前做复杂抽象
5. 所有新增代码都要能解释其属于哪一层
6. 改接口先改 `.api`
7. 写完就验证

---

## 当前验证规则

每完成一个接口或一组行为，至少做一项验证：

- `go run`
- `curl`
- `go test`
- `go build`

Codex/AI 修改必须把“验证步骤”写清楚。Codex 的推荐方式本身就强调在代理循环中运行测试、构建和 lint，再根据反馈修复。:contentReference[oaicite:3]{index=3}

---

## 当前最短目标

在不引入前端、不引入复杂执行器、不引入过度抽象的前提下，先做出一个：

- 结构清楚
- 主链完整
- PostgreSQL 驱动
- 可审计
- 可审批
- 可继续迭代
- 可交给 Codex 持续协作开发

的 Go 后端平台。

---

## 一句话总纲

这个项目不是为了证明“AI 能自动做很多事”，  
而是为了证明“系统如何把 AI/执行器纳入清晰边界、策略约束、审批机制和审计记录中”。
