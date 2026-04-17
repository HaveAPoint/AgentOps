# 【主题】
AgentOps 里 `.api -> goctl -> types/handler/logic` 为什么必须同步

一句话概述：

在当前项目里，`.api` 是接口契约源头；只改 logic 不改 `.api`，或者改了 `.api` 不重新生成，最终都会以字段缺失、构造函数缺失或路由不一致的方式爆出来。

## 【所属分类】
项目实战 / 当前项目中的错误案例

## 【核心结论】

- `.api` 才是请求响应契约的源头，不是 `internal/types/types.go`。
- 改字段名、增删接口后，必须重新执行 `goctl api go -api agentops.api -dir .`。
- `types`、`handler`、`routes` 的很多编译错误，本质上是生成链路没同步。
- 生成代码只能提供骨架，真正的状态机和数据库语义仍要手工实现。

## 【展开解释】

当前项目采用的链路是：

1. 在 `agentops.api` 里定义接口和请求响应
2. 用 `goctl` 生成：
   - `internal/types/types.go`
   - `internal/handler`
   - `internal/logic` 骨架
   - `internal/handler/routes.go`
3. 在 logic / model 里补实际业务

这意味着，下面这些动作如果只做一半，就会出问题：

- 把 `ApproveTaskReq.Comment` 改成 `Reason`
- 新增 `SucceedTaskReq / FailTaskReq`
- 新增 `/tasks/:id/succeed`、`/tasks/:id/fail`

如果 logic 已经写成：

```go
reason := strings.TrimSpace(req.Reason)
```

但 `.api` 还没改，或者改了却没重新跑 `goctl`，就会出现：

- `req.Reason undefined`

同理，如果 handler 里会调用：

```go
tasks.NewApproveTaskLogic(...)
```

但 logic 文件里把构造函数删掉了，或者生成代码没有同步，就会出现：

- `undefined: tasks.NewApproveTaskLogic`

因此这条链路里要始终记住：

- 契约先行
- 生成同步
- 业务落地最后补

## 【代码/场景对应】

当前项目中直接对应：

- [agentops.api](../../agentops.api)
- [internal/types/types.go](../../internal/types/types.go)
- [internal/handler/routes.go](../../internal/handler/routes.go)
- [internal/handler/tasks](../../internal/handler/tasks)
- [internal/logic/tasks](../../internal/logic/tasks)

本次会话中已经实际出现过的典型现象：

- `ApproveTaskReq` 从 `Comment` 改成 `Reason` 后，logic 先用了 `req.Reason`，但生成代码未同步，导致字段未定义。
- 新增 `succeed/fail` 路由后，需要同时生成 `types`、`handler`、`routes`，否则只能手写半套接口。
- `NewApproveTaskLogic` 构造函数缺失，会直接导致 handler 调用失败。

## 【易错点】

- 把 `internal/types/types.go` 当成长期手工维护文件。
- 只改 logic，不改 `.api`。
- 改了 `.api` 但忘记重新跑 `goctl`。
- 以为“已经生成了 handler”就代表业务已经完成。
- 混淆“接口骨架存在”和“生命周期语义已经落地”。

## 【关联知识】

- [Go test、GOCACHE 与 goctl 生成边界](../go/go_test_gocache_and_goctl_generation.md)
- [Go 请求绑定、导出字段与 tag](../go/request_binding_and_tags.md)
- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
- [AgentOps 中 logic 层与 model 层的边界](./logic_model_boundary_in_agentops.md)
