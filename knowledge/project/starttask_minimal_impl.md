# 【主题】
StartTask 的最小实现为什么先只做 `pending -> running`

一句话概述：

L2 推进时，`start` 的第一版应该先把状态机主链打通，而不是一口气把 execution、history、migration 全堆进去。

## 【所属分类】
项目实战 / 设计取舍 / 最小闭环

## 【核心结论】

- `start` 的第一目标是把 `pending -> running` 真正实现
- 当前最小必要输入是 `id + operatorId`
- 第一版先不写 execution，是为了减少改动面，先把主链立住

## 【展开解释】

当前 `start` 这个动作最本质的事情只有一件：

- 把已经可执行但尚未开始的任务，从 `pending` 推进到 `running`

如果第一版就同时加入：

- `task_executions` 插入
- `operator_id` 落库
- 状态历史
- migration 升级
- 审计补齐

那一次改动会同时碰：

- `.api`
- types
- handler
- logic
- model
- SQL migration

进度会明显下降，也更难定位问题。

所以当前阶段的最佳实践是：

1. 先补 `start` 接口契约
2. 先补 `StartTaskLogic`
3. 先只做状态迁移和最小校验
4. 再进入 execution 闭环

## 【代码/场景对应】

当前项目中直接对应：

- [agentops.api](../../agentops.api)
- [internal/handler/tasks/starttaskhandler.go](../../internal/handler/tasks/starttaskhandler.go)
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
- [task/taskL2.md](../../task/taskL2.md)

## 【易错点】

- 以为没把 execution 一起做完，`start` 就不该先写
- 以为 `operatorId` 只是日志字段，不是 start 的最小输入
- 把“先最小闭环”误解成“偷工减料”

## 【关联知识】

- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
