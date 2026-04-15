# 【主题】
AgentOps L2 approve 链路笔记

一句话概述：

L2 的主链不是把 `approve` 和 `start` 混在一起，而是先把审批语义和执行语义拆开：`approve` 进入 `pending`，`start` 由 operator 进入 `running`。

## 【所属分类】
项目实战 / 当前项目中的关键链路

## 【核心结论】

- `approve` 的语义是批准执行，不是开始执行。
- `start` 才应该承担真正进入 `running` 的动作，且它是 operator 动作，不是审批动作。
- `start` 接口的最小输入应是 `operatorId`，它表达“谁真正开始执行了这个任务”。
- 正确的状态迁移是 `pending -> running`，不是 `approved -> start`。
- 如果不先把 `approve -> pending` 修正好，后面的 `start / succeed / fail / execution` 语义都会混乱。

## 【展开解释】

L2 文档要求的主链是：

- `create`
- `approve`
- `start`
- `succeed/fail`

对应状态迁移：

- `waiting_approval -> pending`
- `pending -> running`
- `running -> succeeded/failed`

这里要特别纠正一个容易写偏的点：

- `approved` 只是审批结果，不是执行起点
- `start` 才是执行起点
- 因此状态机里应该是先把任务送入 `pending`，再由 operator 调 `start` 进入 `running`

当前项目在这条链路上的关键点：

- `ApproveTaskLogic` 先检查 `task.Status`
- 再更新状态
- 再写审批记录和 audit log

这一步的最佳实践是：

- 先把 `approve` 改对
- 再继续做 `start` 和 execution 闭环

## 【代码/场景对应】

当前项目中直接对应：

- [task/taskL2.md](../../task/taskL2.md)
- [internal/logic/tasks/taskconsts.go](../../internal/logic/tasks/taskconsts.go)
- [internal/logic/tasks/approvetasklogic.go](../../internal/logic/tasks/approvetasklogic.go)
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
- [internal/handler/tasks/starttaskhandler.go](../../internal/handler/tasks/starttaskhandler.go)

## 【易错点】

- 把 `approve` 和 `start` 混成一个动作。
- 误以为 `start` 是审批的后半段，导致接口设计里缺少 `operatorId`。
- 把状态迁移写成 `approved -> start`，而不是 `pending -> running`。
- 只改数据库状态，不改返回响应状态，导致接口和数据库状态分裂。
- 只改响应，不改数据库，导致表面正确、持久化错误。

## 【关联知识】

- 状态机设计
- 事务
- 审批语义
- execution 闭环
- [approval_records 当前语义与 L2 目标](./approval_records_semantics.md)
