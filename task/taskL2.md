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
