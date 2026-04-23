# 【主题】
AgentOps 中 JWT claims 与 CurrentUser 上下文的绑定路径

一句话概述：

生产链路里 CurrentUser 主要来自 JWT 鉴权中间件写入 context 的 claims，而不是业务代码手动调用 `WithCurrentUser`。

## 【所属分类】
项目实战 / 关键链路

## 【核心结论】

- 任务相关接口开启了 JWT 鉴权，认证发生在路由层。
- 登录接口签发 token 时写入了 `userId`、`username`、`systemRole` 三个业务 claims。
- 运行时由 go-zero 的 JWT 中间件解析 token 并把非标准 claims 注入请求 context。
- handler 将 `r.Context()` 透传到 logic，logic 再通过 `CurrentUserFromContext` 读取用户信息。
- `WithCurrentUser` 在当前仓库主要用于测试或手工注入场景，不是 HTTP 正常主链路必经步骤。

## 【展开解释】

当前仓库的实际链路可以概括为：

1. 任务路由启用 `rest.WithJwt(...)`，所有受保护接口都会先走 JWT 中间件。
2. 登录逻辑签发 access token 时，把用户标识和系统角色写入 claims。
3. 框架中间件验证 token 后，将 claims 写入 `request.Context()`。
4. 各 handler 用 `r.Context()` 创建 logic。
5. logic 通过 `CurrentUserFromContext` 优先读取 typed user，读不到则回退读取 claims（`userId`/`username`/`systemRole`）。

这解释了为什么在业务代码里几乎看不到 `WithCurrentUser` 的调用，但 `CurrentUserFromContext` 仍然能在运行时拿到当前用户。

## 【代码/场景对应】

当前项目中的直接对应点：

- [internal/handler/routes.go](../../internal/handler/routes.go)（任务路由开启 JWT）
- [internal/logic/auth/loginlogic.go](../../internal/logic/auth/loginlogic.go)（签发 token 的 claims）
- [internal/handler/tasks/createtaskhandler.go](../../internal/handler/tasks/createtaskhandler.go)（透传 `r.Context()`）
- [internal/logic/tasks/createtasklogic.go](../../internal/logic/tasks/createtasklogic.go)（读取 CurrentUser）
- [internal/auth/actor.go](../../internal/auth/actor.go)（claims 与 typed user 的读取逻辑）
- [internal/auth/actor_test.go](../../internal/auth/actor_test.go)（`WithCurrentUser` 的测试注入示例）

## 【易错点】

- 误以为 `WithCurrentUser` 没有生产调用就代表鉴权上下文没有绑定。
- 登录写入的 claim 名和 `CurrentUserFromContext` 读取的 key 不一致，导致运行时 `unauthenticated`。
- 在 handler 或 logic 中新建了 `context.Background()`，导致 JWT 注入过的请求上下文丢失。

## 【关联知识】

- [AgentOps 里 `.api -> goctl -> types/handler/logic` 为什么必须同步](./api_goctl_sync_boundary.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)
- [Go 请求绑定、导出字段与 tag](../go/request_binding_and_tags.md)