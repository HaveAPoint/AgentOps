L3 计划（真实身份与权限版）

目标

把 L2 里“占位的业务角色语义”升级成：

- 真实用户身份
- 真实权限判断
- 真实任务归属
- 基于业务角色的可见性控制

L3 的本质是：

把 L2 里的 creator / reviewer / operator，从“逻辑约定”变成“真实可验证的协作系统”。

L3 核心业务流

- 用户登录
- creator 创建任务
- reviewer 审批任务
- operator 开始执行
- operator 回写执行结果
- creator / reviewer / admin 在权限内取消任务
- 所有动作都带真实 actor

L3 要做的事

1. 建立真实用户体系

至少补：

- `users` 表
- 用户基础信息
- password hash
- 基本系统角色字段

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

最小系统角色建议：

- `admin`
- `reviewer`
- `operator`
- `viewer`

任务归属建议围绕：

- `creator_id`
- `reviewer_id`
- `operator_id`

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

建议最小规则：

- creator 能看自己创建的任务
- reviewer 能看需要自己审批的任务
- operator 能看分配给自己执行的任务
- admin 能看全部

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
- 可见性规则生效
- 审计与状态历史写入真实 actor

L3 明确不做

- OAuth2 / OIDC / SSO
- 组织树
- 多租户
- 细粒度复杂权限平台
- Casbin 式重型策略体系

L3 完成标志

做到下面这些，L3 才算完整：

- 系统有真实用户，不再依赖占位 actor
- JWT 登录和身份注入能工作
- 系统角色和任务归属都成立
- 关键动作权限判断进入任务主链
- list / detail / logs / executions 已按业务角色做可见性控制
- 审计日志和状态历史能写入真实 actor
- 项目开始具备真实团队协作属性
