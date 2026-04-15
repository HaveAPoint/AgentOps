# 【主题】
AgentOps L2 任务生命周期图

一句话概述：

L2 的核心不是多几个接口，而是把任务从“创建、审批、开始执行、结束”这条主链拆清楚。

## 【所属分类】
项目实战 / 当前项目中的关键链路

## 【核心结论】

- `create` 之后不会直接进入 `running`。
- 需要审批的任务：`create -> waiting_approval -> pending -> running -> succeeded/failed`
- 不需要审批的任务：`create -> pending -> running -> succeeded/failed`
- `cancel` 在当前 L2 里只允许 `waiting_approval` 或 `pending` 进入 `cancelled`

## 【展开解释】

当前 L2 里要严格区分两类东西：

- 动作：`create / approve / start / succeed / fail / cancel`
- 状态：`waiting_approval / pending / running / succeeded / failed / cancelled`

对应关系：

- `create`
  - 如果 `approvalRequired=true`，进入 `waiting_approval`
  - 如果 `approvalRequired=false`，进入 `pending`
- `approve`
  - `waiting_approval -> pending`
- `start`
  - `pending -> running`
- `succeed`
  - `running -> succeeded`
- `fail`
  - `running -> failed`
- `cancel`
  - `waiting_approval -> cancelled`
  - `pending -> cancelled`

这里最容易混的是：

- `approve` 不是“开始执行”
- `pending` 不是“没创建好”
- `pending` 表示“已经允许执行，但还没真正开始”

## 【代码/场景对应】

当前项目中直接对应：

- [task/taskL2.md](../../task/taskL2.md)
- [internal/logic/tasks/createtasklogic.go](../../internal/logic/tasks/createtasklogic.go)
- [internal/logic/tasks/approvetasklogic.go](../../internal/logic/tasks/approvetasklogic.go)
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)

## 【易错点】

- 把 migration 里状态枚举的排列顺序误认为状态流转顺序
- 把 `approve` 和 `start` 混成一个动作
- 误以为 `pending` 还不能开始执行

## 【关联知识】

- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
- [create / approve / start 的角色语义](./create_approve_start_roles.md)
