# 【主题】
create / approve / start 的角色语义

一句话概述：

L2 虽然还没有真实用户系统，但已经要把 creator、reviewer、operator 三类动作责任分开。

## 【所属分类】
项目实战 / 角色语义 / 设计取舍

## 【核心结论】

- `create` 对应 creator
- `approve` 对应 reviewer
- `start` 对应 operator
- 当前项目中有些字段还是“借位使用”，要分清“当前落地”和“最终语义”

## 【展开解释】

当前 L2 想表达的是：

- creator 创建任务
- reviewer 决定任务能不能进入可执行状态
- operator 真正开始执行任务

这三者不是一回事。

当前代码里的对应关系：

- `CreateTaskReq`
  还没有完整的 creator 字段体系
- `ApproveTaskReq.ReviewerId`
  表达这次 approve 动作是谁触发的
- `StartTaskReq.OperatorId`
  表达这次 start 动作是谁触发的

当前数据库里 `approval_records.approved_by` 仍然存在，所以现阶段是：

- 先把 `reviewerId` 借位写进 `approved_by`
- 先把主链跑通
- 后续再把 `approval_records` 升级成更明确的 `reviewer_id / decision / reason`

要特别区分：

- `approved_by`
  谁做了 approve 这个动作
- `reviewer_id`
  任务被分配给谁审核
- `operator_id`
  谁真正开始执行或执行了任务

## 【代码/场景对应】

当前项目中直接对应：

- [agentops.api](../../agentops.api)
- [internal/types/types.go](../../internal/types/types.go)
- [internal/logic/tasks/approvetasklogic.go](../../internal/logic/tasks/approvetasklogic.go)
- [internal/logic/tasks/starttasklogic.go](../../internal/logic/tasks/starttasklogic.go)
- [internal/model/approvalrecordmodel.go](../../internal/model/approvalrecordmodel.go)

## 【易错点】

- 以为 `approved_by` 和 `reviewer_id` 完全等价
- 以为 `operatorId` 只是日志细节，不是 start 的最小必要输入
- 角色字段还没完全升级时，忘记“当前是借位使用”

## 【关联知识】

- [approval_records 当前语义与 L2 目标](./approval_records_semantics.md)
- [AgentOps L2 任务生命周期图](./task_lifecycle_map.md)
