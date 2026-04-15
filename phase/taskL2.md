L2 计划（业务语义版）

目标

把项目从“有接口、有状态”的原型，推进到：

- 状态流转正确
- execution 闭环成立
- 任务主链语义清楚
- 虽然还没有真实用户系统，但系统内部已经能表达 creator / reviewer / operator

当前仓库基线

当前仓库已经有：

- create / list / detail / approve / cancel / logs / executions 接口
- PostgreSQL 落库
- Git 基础上下文采集

当前仓库还缺：

- `approve -> pending` 的中间态语义
- `start / succeed / fail` 接口和逻辑
- execution 真正写入闭环
- 结构化状态历史
- creator / reviewer / operator 的明确业务归属

L2 核心业务流

- creator 创建任务
- 如果需要审批，任务进入 `waiting_approval`
- reviewer approve 后，任务进入 `pending`
- operator start 后，任务进入 `running`
- operator succeed / fail 后，任务结束
- 合法角色可 cancel

L2 必须先改正的一点

当前仓库里 `approve` 还是直接把任务从 `waiting_approval` 推到 `running`。

L2 需要统一改成：

- `waiting_approval -> pending`

原因：

- approve = 允许执行
- start = 真正开始执行

这一步在 L2 改好，L3 的权限语义和 L4 的执行器约束都会更顺。

L2 要做的事

1. 调整状态机，显式区分审批通过与开始执行

统一状态：

- `waiting_approval`
- `pending`
- `running`
- `succeeded`
- `failed`
- `cancelled`

统一动作：

- `create`
- `approve`
- `start`
- `succeed`
- `fail`
- `cancel`

建议固定迁移：

- `create`: `none -> waiting_approval` 或 `none -> pending`
- `approve`: `waiting_approval -> pending`
- `start`: `pending -> running`
- `succeed`: `running -> succeeded`
- `fail`: `running -> failed`
- `cancel`: `waiting_approval -> cancelled`、`pending -> cancelled`

`running -> cancelled` 是否允许，要在 L2 明确拍板；如果允许，必须定义成一次受控中断，而不是模糊取消。

2. 补齐业务角色字段，但先不做真实用户体系

`tasks` 至少补这些字段：

- `creator_id`
- `reviewer_id`，可空
- `operator_id`，可空
- `approved_by`，可空
- `approved_at`，可空
- `cancelled_by`，可空
- `cancelled_at`，可空

这些字段在 L2 可以先用业务占位值或简单字符串，不要求立刻关联 `users` 表。

建议不要继续沿用泛化的 `created_by` 语义，优先直接改成 `creator_id`。

3. 补 execution 真闭环，并把 execution 和 operator 绑定

`task_executions` 至少要表达：

- `task_id`
- `operator_id`
- `status`
- `started_at`
- `finished_at`
- `result_summary`
- `error_message`

语义要求：

- `start` 时创建 execution
- `succeed / fail / cancel` 时收口 execution
- execution 要能回答“谁执行了这次任务”

4. 把任务状态更新和 execution 更新放进同一事务

至少保证这些动作在单事务里完成：

- `approve`
- `start`
- `succeed`
- `fail`
- `cancel`

核心要求：

- 锁 task
- 校验当前状态是否合法
- 更新 task / execution / approval / history
- 保证 task 状态和 execution 状态不分裂

5. 新增结构化状态历史，替代纯 message 审计

新增 `task_status_histories` 表，至少记录：

- `task_id`
- `from_status`
- `to_status`
- `action`
- `actor_id`
- `actor_role`
- `reason`
- `created_at`

这里的 `actor_role` 很关键。

因为 L2 虽然还没有真实身份体系，但已经要能表达：

- 这是 creator 触发的
- 这是 reviewer 触发的
- 这是 operator 触发的

6. 补全 approval 记录语义，但不要让设计跑到主链前面

`approval_records` 建议升级成：

- `task_id`
- `reviewer_id`
- `decision`
- `reason`
- `created_at`

其中：

- `decision` 可以为后续预留 `approved / rejected`
- 但 L2 主链默认只要求把 `approved` 做完整

如果要引入 `reject`，必须同步定义它对应的任务状态结果；这不属于当前 L2 的默认必做项。

7. 按业务角色收口动作语义

L2 先从业务上约定清楚这些动作是谁做的：

- `create`: creator
- `approve`: reviewer
- `start`: operator
- `succeed / fail`: operator
- `cancel`: creator / reviewer / admin

即使没有 JWT，logic 层也不要再继续写成“匿名 system 在执行所有事”，而是要允许动作显式带 actor。

8. 补接口与契约

`.api` 需要补充至少这些接口：

- `POST /tasks/:id/start`
- `POST /tasks/:id/succeed`
- `POST /tasks/:id/fail`

如果 L2 要承接占位 actor，相关请求或上下文也要能把 actor 信息传入 logic 层并写入数据库。

9. 补第一批行为级测试

至少覆盖：

- create task 成功 / 失败
- 非 git 仓库失败
- approve / start / succeed / fail / cancel 的合法与非法状态
- 重复 approve、重复 start、重复 finish 的预期行为
- 至少两组并发测试

当前仓库还没有测试文件，这一轮要把“只编译通过”推进到“最小行为正确”。

10. 同步 README 和阶段说明

README 需要明确：

- `approve` 不再直接进入 `running`
- `start` 才是真正开始执行
- `task_executions` 已经进入真实闭环
- 当前只是占位 actor，不是完整认证体系

L2 明确不做

- JWT
- 真实 `users` 表
- RBAC
- 可见性控制
- OAuth / SSO
- 多 worker
- MQ
- 复杂沙箱
- repo/path 策略的真实执行

L2 完成标志

做到下面这些，L2 才算完整：

- 任务主链已经是 `create -> approve -> start -> succeed/fail`
- `approve` 不再直接变成 `running`
- `task_executions` 真正可用
- 状态流转和 execution 更新有事务保护
- 状态历史中能看到 `action + actor_id + actor_role`
- 任务已经能表达 creator / reviewer / operator 三类业务角色
- 非法状态迁移、重复动作、并发场景都有明确业务语义
- README 与当前实现边界一致

当前环境前置条件

为了让 L2 可以从“文档设计”进入“可开发、可验证”的状态，本地环境至少要满足：

- Go `1.26.1`
- `goctl 1.10.1`
- PostgreSQL 客户端 `psql`
- Docker
- Git

固定约定：

- PostgreSQL 容器名固定为 `agentops-postgres`
- 端口映射固定为 `5432:5432`
- 数据库连接参数固定为 `postgres/postgres/agentops`
- 配置文件以 `etc/agentops-api.yaml` 为准
- migration 基线从 `migrations/0001_init.sql` 开始

环境完成标准：

- `go version` 可用，且版本为 `1.26.1`
- `goctl --version` 可用，且版本为 `1.10.1`
- `psql --version` 可用
- 本地可访问 `127.0.0.1:5432`
- PostgreSQL 中存在数据库 `agentops`
- `tasks / task_policies / task_executions / approval_records / audit_logs` 已创建

L2 具体执行计划（落地顺序版）

Phase 1. 环境与基线确认

要完成的事：

- 安装 Go `1.26.1`
- 安装 `goctl 1.10.1`
- 安装 PostgreSQL 客户端 `psql`
- 新建并启动 `agentops-postgres`
- 创建数据库 `agentops`
- 执行 `migrations/0001_init.sql`
- 跑一次 `goctl` 生成链路，确认 `.api -> types / handler` 工作正常
- 跑一次基础编译或测试，建立 L2 开发基线

阶段产出：

- 可运行的本地后端环境
- 可访问的 PostgreSQL
- 一份“当前基线可编译/可启动”的确认结果

完成判断：

- 开发者可以在本机直接连接数据库并启动服务
- `.api` 修改后具备重新生成代码的能力
- 后续 L2 开发不再被环境问题阻塞

Phase 2. 数据模型与迁移

要完成的事：

- 在 `tasks` 增加 `creator_id / reviewer_id / operator_id / approved_by / approved_at / cancelled_by / cancelled_at`
- 停止把 `created_by` 作为主业务字段继续扩散
- migration 中把历史 `created_by` 回填到 `creator_id`
- 在 `task_executions` 增加 `operator_id / error_message`
- 把 `approval_records` 调整为 `reviewer_id / decision / reason`
- 新增 `task_status_histories`

阶段产出：

- 一份新的 migration 文件
- 更新后的 model struct
- 更新后的 SQL 字段映射
- 一份明确的回填与兼容策略

完成判断：

- 数据库层已经能表达 creator / reviewer / operator
- execution 已经能记录执行人与失败信息
- 状态历史有独立结构化落点，不再只依赖 message 审计

Phase 3. 接口契约与占位 actor 方案

要完成的事：

- 在 `.api` 新增 `POST /tasks/:id/start`
- 在 `.api` 新增 `POST /tasks/:id/succeed`
- 在 `.api` 新增 `POST /tasks/:id/fail`
- 给 `CreateTaskReq` 增加 `creatorId`，并允许带 `reviewerId / operatorId`
- 给 `ApproveTaskReq` 增加 `reviewerId`
- 给 `CancelTaskReq` 增加 `actorId / actorRole`
- 给 `Start / Succeed / Fail` 请求显式增加 `operatorId`
- 把任务响应里的 `CreatedBy` 统一切换为 `CreatorId`
- 在任务详情响应补充 `ReviewerId / OperatorId / ApprovedBy / ApprovedAt / CancelledBy / CancelledAt`
- 在 execution 响应补充 `OperatorId / ErrorMessage`

固定约定：

- L2 的 actor 一律通过请求体字段透传
- L2 不引入 JWT
- L2 不引入真实用户表

阶段产出：

- 更新后的 `.api`
- 重新生成后的 `types` 与 `handler`
- 一套固定的占位 actor 传递方案

完成判断：

- 实现层不再依赖匿名 `system` 处理所有动作
- 公开接口已经能承接 creator / reviewer / operator 三类业务语义

Phase 4. 状态机、事务与 execution 闭环

要完成的事：

- 先修正 `approve: waiting_approval -> pending`
- 再实现 `start: pending -> running`
- 再实现 `succeed / fail: running -> succeeded / failed`
- `cancel` 只允许 `waiting_approval / pending -> cancelled`
- L2 明确不允许 `running -> cancelled`

事务要求：

- `approve / start / succeed / fail / cancel` 全部在单事务中执行
- 每次动作都必须先锁 task
- `succeed / fail` 必须锁当前 execution
- 同一事务内同时更新 `task / execution / approval / history / audit`
- 非法状态迁移必须显式失败
- 重复动作必须有清晰错误语义，不允许静默成功

阶段产出：

- 统一状态机规则
- execution 写入闭环
- 结构化状态历史落库
- actor 与 `actor_role` 可追踪

完成判断：

- 任务状态和 execution 状态不会分裂
- `approve` 不再直接推进到 `running`
- 执行记录能够回答“谁开始了执行、如何结束”

Phase 5. 测试与并发校验

要完成的事：

- 覆盖 create 成功 / 失败
- 覆盖非 git 仓库失败
- 覆盖 approve / start / succeed / fail / cancel 的合法与非法状态
- 覆盖重复 approve、重复 start、重复 finish
- 覆盖并发 approve
- 覆盖并发 start
- 覆盖详情、logs、executions 是否反映新增字段与新语义

阶段产出：

- 第一批 `_test.go`
- 最小行为正确性保障
- 一份并发状态机不分裂的验证结果

完成判断：

- 主链不再只是“能编译”
- 非法迁移、重复动作、并发竞争都有明确预期

Phase 6. README 与阶段文档同步

要完成的事：

- 在 README 中明确 `approve` 不再直接进入 `running`
- 在 README 中明确 `start` 才是真正开始执行
- 在 README 中明确 `task_executions` 已经进入真实闭环
- 在 README 中明确 actor 只是占位业务字段，不是认证体系
- 在本文件保留“目标 / 范围 / 完成标志”，同时补足可执行顺序

阶段产出：

- 与实现边界一致的 README
- 可直接指导实施的 L2 文档

完成判断：

- 文档不再只有业务语义，没有执行顺序
- 一个新接手的开发者可以直接按阶段推进

接口与公开契约变更清单

L2 需要明确未来会变更的公开接口与类型：

- 新增 `start / succeed / fail` 三个任务动作接口
- 任务响应中的 `CreatedBy` 语义切换到 `CreatorId`
- execution 查询结果增加执行人与失败信息
- actor 信息统一通过请求体显式传入

这些契约必须先在 `.api` 中定义，再进入实现。

文档验收标准

文档补充完成后，至少要满足：

- 环境部分可以指导一个新开发者把本地依赖装齐
- 实施顺序已经写死，不需要实现者再自行决定阶段顺序
- 接口、数据库、事务、测试、文档同步都被覆盖
- 默认边界已经写死：无 JWT、无用户表、无 RBAC、无 `running -> cancelled`
