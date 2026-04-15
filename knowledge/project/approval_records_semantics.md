# 【主题】
approval_records 当前语义与 L2 目标

一句话概述：

当前 `approval_records` 还是“谁批准了 + 留了什么评论”的最小记录，还不是 L2 最终目标里的完整审批决策模型。

## 【所属分类】
项目实战 / 当前项目中的表结构 / 设计取舍

## 【核心结论】

- 当前表更像审批痕迹，不是完整审批模型。
- `approved_by` 和 `reviewer_id` 不是完全等价的字段。
- L2 最终更理想的字段是 `reviewer_id / decision / reason`。
- 但从推进节奏看，先最小改造继续往前走是合理选择。

## 【展开解释】

当前表结构：

```sql
approval_records (
    id,
    task_id,
    approved_by,
    comment,
    created_at
)
```

当前最能表达的是：

- 谁执行了 approve 动作
- 留了什么自由文本评论

它还不能稳定表达：

- 这个任务分配给谁审核
- 这次审批结果是什么（approved / rejected）
- 这次审批的业务原因是什么

其中要特别区分：

- `approved_by`
  谁执行了这次批准动作
- `reviewer_id`
  任务上被分配的 reviewer

很多时候它们可能一样，但语义不是一回事。

## 【代码/场景对应】

当前项目中直接对应：

- [migrations/0001_init.sql](../../migrations/0001_init.sql)
- [internal/model/approvalrecordmodel.go](../../internal/model/approvalrecordmodel.go)
- [internal/logic/tasks/approvetasklogic.go](../../internal/logic/tasks/approvetasklogic.go)

## 【易错点】

- 直接把 `approved_by` 当成 `reviewer_id` 理解。
- 认为只靠 `comment` 就足够表达审批结果。
- 用 `bool` 直接代替 `decision`，导致后续扩展空间很差。

## 【关联知识】

- 数据建模
- actor 与 role assignment
- 结构化决策字段
- 项目分阶段重构策略
- [AgentOps L2 approve 链路笔记](./agentops_l2_approve_flow.md)
