L3 计划（真实身份与权限版）

目标

把 L2 里“占位的业务角色语义”升级成：

- 真实用户身份
- 真实权限判断
- 真实任务归属
- 基于业务角色的可见性控制

L3 的本质是：

把 L2 里的 creator / reviewer / operator，从“逻辑约定”变成“真实可验证的协作系统”。

做到这一步之后，系统至少要能可信回答：

- 谁创建了任务
- 谁审批了任务
- 谁启动并结束了执行
- 谁有权限查看这个任务
- 谁因为什么做了这次动作

L3 核心业务流

- 用户登录
- creator 创建任务
- reviewer 审批任务
- operator 开始执行
- operator 回写执行结果
- creator / reviewer / admin 在权限内取消任务
- creator / reviewer / operator / admin 只能看到自己有权访问的任务
- 所有动作都带真实 actor

L3 先拍板的事

在进入实现前，L3 需要先固定几条默认规则，避免后面边写边改：

1. 用户主键与 actor 字段兼容策略

- 当前 `tasks / task_executions / approval_records / task_status_histories` 里的 actor 字段都是 `VARCHAR(64)`
- L3 第一版建议让 `users.id` 也使用字符串主键
- 先把“真实身份链”跑通，不在 L3 同时引入 numeric ID 改造

2. 系统角色与任务归属分层

- 系统角色：`admin / reviewer / operator / viewer`
- 任务归属：`creator_id / reviewer_id / operator_id`
- 系统角色回答“总体上能做什么”
- 任务归属回答“在某个 task 上你是谁”

3. 最小可见性规则

- creator 能看自己创建的任务
- reviewer 能看分配给自己审批的任务
- operator 能看分配给自己执行的任务
- admin 能看全部

4. 查询失败语义

- 未登录：`401`
- 已登录但关键动作越权：`403`
- 查询一个自己不可见的任务：`404`

`404` 的目的不是“严格 REST 教条”，而是避免暴露任务存在性。

5. migration 策略

- 不回写修改 `migrations/0001_init.sql`
- L3 新增独立 migration，例如 `0003_l3_auth_and_visibility.sql`
- 从 L3 开始建立“增量迁移”的工程习惯

L3 要做的事

1. 建立真实用户体系

至少补：

- `users` 表
- 用户基础信息
- password hash
- 基本系统角色字段
- `UserModel`

L2 里的占位 `creator_id / reviewer_id / operator_id / approved_by / cancelled_by`，在 L3 要正式关联到 `users.id`。

2. 接入真实登录和身份注入

替换当前占位登录逻辑，补齐：

- JWT 签发
- JWT 解析
- middleware 身份校验
- 请求上下文 actor 注入

要求不是“接口能登录”就算完，而是任务链上的关键动作都能拿到可信 actor。

3. 区分系统角色和任务归属

L3 要明确两层概念：

- 系统角色：你总体上能做什么
- 任务归属：在某个具体任务上你是谁

例如，一个用户可以有系统角色 `reviewer`，但只有当他是某个任务的 `reviewer_id` 时，才是那个任务上的具体 reviewer。

4. 把权限控制接入任务操作链

至少明确：

- 谁能创建任务
- 谁能审批任务
- 谁能开始执行
- 谁能提交 succeed / fail
- 谁能取消任务

这一步不能只停留在 middleware，要真正进入 task logic。

5. 做任务可见性控制

至少覆盖这些查询：

- list
- detail
- logs
- executions
- status histories

L3 的查询控制不是“看不到按钮”，而是服务端根本不给越权结果。

6. 让审计链和状态历史写入真实 actor

L2 里只是埋了 `actor_id / actor_role`。

L3 要做到：

- `actor_id` 来自当前登录用户
- `actor_role` 来自真实权限与归属判断
- approve / start / succeed / fail / cancel 的 actor 都可信

7. 把理由字段和业务角色绑定

至少补齐：

- 审批理由
- 取消理由
- 执行失败理由

这些内容需要进入：

- `approval_records`
- `task_status_histories`
- `audit_logs`

要求不是“有 comment 字段”，而是能回答“谁因为什么做了这次动作”。

8. 补 L3 测试

至少覆盖：

- 登录成功 / 失败
- 未登录访问受保护接口失败
- 不同系统角色的权限差异
- 非法角色执行关键动作失败
- 非归属用户执行关键动作失败
- 可见性规则生效
- 审计与状态历史写入真实 actor

L3 环境与前置条件

在进入 L3 实现前，本地至少要满足：

- L2 当前测试可跑通
- `go version` 可用
- `goctl --version` 可用，且能从 `.api` 重新生成 `types / handler`
- PostgreSQL 可访问
- `migrations/0001_init.sql` 已可稳定建库
- 开发者可以稳定运行：

```bash
go test ./internal/logic/tasks -run TestL2 -count=1
```

L3 具体执行计划（落地顺序版）

Phase 1. 规则冻结与边界确认

要完成的事：

- 写死系统角色：`admin / reviewer / operator / viewer`
- 写死任务归属：`creator_id / reviewer_id / operator_id`
- 写死最小可见性规则
- 写死查询失败语义：`401 / 403 / 404`
- 写死 L3 不做的范围，避免实现中途膨胀
- 明确哪些 actor 字段从请求体移除，哪些字段保留为任务分配输入

阶段产出：

- 一份固定的权限与可见性规则表
- 一份固定的 L3 默认边界

完成判断：

- 后续实现者不需要再边写边定义权限规则
- “谁能做、谁能看、失败返回什么”都已经写死

Phase 2. 数据模型与 migration

要完成的事：

- 新增 `users` 表
- 字段至少包括：`id / username / password_hash / system_role / created_at / updated_at`
- 给 `username` 加唯一约束
- 新增 `UserModel`
- 在 `ServiceContext` 注册 `UserModel`
- 给 `tasks.creator_id / reviewer_id / operator_id / approved_by / cancelled_by` 补外键到 `users.id`
- 给 `task_executions.operator_id` 补外键到 `users.id`
- 给 `approval_records.reviewer_id` 补外键到 `users.id`
- 给 `task_status_histories.actor_id` 补外键到 `users.id`

固定约定：

- L3 第一版沿用字符串用户主键
- L3 使用新增 migration，不回改 `0001_init.sql`

阶段产出：

- 一份新的 migration 文件
- 新的 `User` model 与 SQL 映射
- 一份明确的外键落地策略

完成判断：

- 数据库层已经能表达真实用户
- 任务链上所有核心 actor 都能被数据库约束

Phase 3. 登录、JWT 与身份注入

要完成的事：

- 在配置中增加 JWT 所需字段，例如 `secret / expire_seconds`
- 把当前占位登录改成真实登录
- 使用 password hash 校验密码
- 登录成功后签发 JWT
- 为受保护路由接入认证 middleware
- 在请求上下文注入当前用户信息
- 预留一个稳定的 actor context 读取入口，避免后面 logic 重复解析 token

阶段产出：

- 真实可用的登录链路
- 统一的当前用户上下文获取方式

完成判断：

- `/auth/login` 已经真实查询数据库用户
- `/tasks` 相关接口不再依赖匿名请求
- logic 层可以从 context 稳定拿到当前 actor

Phase 4. 接口契约调整

要完成的事：

- 在 `.api` 里移除由客户端自报当前 actor 的字段
- `CreateTaskReq` 去掉 `creatorId`
- `ApproveTaskReq` 去掉 `reviewerId`
- `StartTaskReq` 去掉 `operatorId`
- `SucceedTaskReq` 去掉 `operatorId`
- `FailTaskReq` 去掉 `operatorId`
- `CancelTaskReq` 去掉 `actorId / actorRole`
- `CreateTaskReq` 可以继续保留 `reviewerId / operatorId`，因为它们是任务分配目标，不是当前 actor
- 重新生成 `types / handler`

固定约定：

- 从 L3 开始，当前操作者一律来自 auth context
- 请求体只允许表达“任务目标信息”，不允许表达“我是谁”

阶段产出：

- 更新后的 `.api`
- 重新生成后的 `types` 与 `handler`
- 一套新的 actor 输入约定

完成判断：

- 客户端已经无法冒充 `reviewer / operator / admin`
- “当前用户”和“任务分配对象”在接口层彻底分开

Phase 5. 写操作权限接入任务主链

要完成的事：

- `CreateTask` 从当前登录用户写入 `creator_id`
- `ApproveTask` 从当前登录用户解析 reviewer 身份
- `StartTask / SucceedTask / FailTask` 从当前登录用户解析 operator 身份
- `CancelTask` 从当前登录用户计算 `actor_id / actor_role`
- 把权限判断真正落到 task logic，不只靠 middleware
- 给 `approve / start / succeed / fail / cancel` 接入统一权限判断入口
- 把 `actor_role` 改为由系统根据任务归属计算，不再接受请求体自报
- 保持已有状态机和事务边界不被破坏

建议同时补出的共享规则文件：

- `authz.go`
- `visibility.go`
- `actor.go`
- `state_machine.go`

阶段产出：

- 一套真实权限驱动的写操作主链
- 统一的 actor / 权限 / 归属判断逻辑

完成判断：

- 非法系统角色不能执行关键动作
- 非归属用户不能执行属于别人的动作
- actor 已经从“客户端输入”变成“系统推导”

Phase 6. 查询可见性控制

要完成的事：

- 改造 `ListTasks`，从“全量返回”变为“按当前 actor 过滤”
- 给 `GetTask` 增加可见性校验
- 给 `GetTaskLogs` 增加可见性校验
- 给 `GetTaskExecutions` 增加可见性校验
- 给 `GetTaskStatusHistories` 增加可见性校验
- 对不可见任务统一返回 `404`

阶段产出：

- 一套基于业务角色的服务端可见性控制

完成判断：

- 查询接口不再只是“知道 taskId 就能查”
- creator / reviewer / operator / admin 的可见范围已在服务端收口

Phase 7. 测试与回归校验

要完成的事：

- 为测试库补用户 seed
- 补登录成功 / 失败测试
- 补未登录访问受保护接口失败测试
- 补不同系统角色的动作权限测试
- 补非归属用户越权测试
- 补查询可见性测试
- 补审计与状态历史 actor 正确性测试
- 保留并复用 L2 已有生命周期与并发测试，避免 L3 回归打断 L2 主链

阶段产出：

- 第一批 L3 集成测试
- 一份“身份可信、权限正确、查询受控”的回归结果

完成判断：

- L3 不只是“能登录”
- 身份、权限、查询、审计都已经有自动化保障

Phase 8. README 与阶段文档同步

要完成的事：

- 在 README 中明确当前 actor 已来自真实登录用户
- 在 README 中明确请求体不再传 `creatorId / reviewerId / operatorId / actorId`
- 在 README 中明确查询接口已按可见性过滤
- 在本文件保留“目标 / 范围 / 完成标志”，同时补足可执行顺序与默认规则

阶段产出：

- 与实现边界一致的 README
- 可直接指导实施的 L3 文档

完成判断：

- 一个新接手的开发者可以直接按阶段推进
- 文档、接口、数据库、测试口径一致

接口与公开契约变更清单

L3 需要明确未来会变更的公开接口与类型：

- `POST /auth/login` 从假 token 改为真实登录
- 任务写接口不再接受客户端自报当前 actor
- `CreateTaskReq` 去掉 `creatorId`
- `ApproveTaskReq` 去掉 `reviewerId`
- `Start / Succeed / Fail` 请求去掉 `operatorId`
- `CancelTaskReq` 去掉 `actorId / actorRole`
- 查询接口开始按业务角色做可见性控制

这些契约必须先在 `.api` 中定义，再进入实现。

文档验收标准

文档补充完成后，至少要满足：

- 已明确 L3 的默认边界和不做项
- 已写死实施顺序，不需要实现者再自行决定阶段顺序
- 数据库、接口、认证、权限、可见性、测试、文档同步都被覆盖
- actor 来源已经明确：来自 auth context，不来自请求体
- 查询失败语义已经明确：`401 / 403 / 404`

L3 明确不做

- OAuth2 / OIDC / SSO
- 组织树
- 多租户
- 细粒度复杂权限平台
- Casbin 式重型策略体系
- 复杂审批流
- 多 reviewer / 多 operator 分派

L3 完成标志

做到下面这些，L3 才算完整：

- 系统有真实用户，不再依赖占位 actor
- JWT 登录和身份注入能工作
- 系统角色和任务归属都成立
- 关键动作权限判断进入任务主链
- list / detail / logs / executions / status histories 已按业务角色做可见性控制
- 审计日志和状态历史能写入真实 actor
- 客户端不能再通过请求体冒充 reviewer / operator / admin
- 项目开始具备真实团队协作属性
