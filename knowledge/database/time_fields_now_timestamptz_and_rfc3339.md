# 【主题】
时间字段、`NOW()`、`TIMESTAMPTZ`、`time.Time` 与 RFC3339 的区别

一句话概述：

当前项目里同一条时间数据会经过数据库时间类型、Go 内部时间对象和 HTTP 字符串三层表示，`NOW()` 不是 Unix 时间戳。

## 【所属分类】
数据库 / PostgreSQL

## 【核心结论】

- PostgreSQL 里的 `NOW()` 返回的是当前时间点，不是 Unix 秒数。
- 当前项目数据库层主要使用 `TIMESTAMPTZ` 保存时间。
- Go 内部常用 `time.Time` 或 `sql.NullTime` 表示这些时间。
- 返回给 HTTP 响应时，通常再转成 `RFC3339` 字符串。

## 【展开解释】

### 1. `NOW()` 是什么

`NOW()` 是 PostgreSQL 的时间函数。

在当前项目里，它通常配合：

- `created_at`
- `updated_at`
- `occurred_at`

这类列一起出现。

它返回的不是：

- `1713340800` 这种 Unix 时间戳

而是数据库里的时间类型值。

### 2. `TIMESTAMPTZ` 是什么

`TIMESTAMPTZ` 是 PostgreSQL 的时间类型：

- timestamp with time zone

它表示的是：

- 一个带时区语义的时间点

所以数据库层的时间列，不是 `string`，也不是 Go 结构体，而是 PostgreSQL 自己的时间类型。

### 3. Go 里如何接住它

数据库里的 `TIMESTAMPTZ` 查出来后，Go 通常用：

- `time.Time`
- `sql.NullTime`

接收。

区别：

- `time.Time` 适合非空时间
- `sql.NullTime` 适合数据库可能为 `NULL` 的时间列

### 4. 为什么还会看到 `UTC()`

logic 层常见：

- `time.Now().UTC()`

它的作用不是“把时间变成字符串”，而是：

- 先得到一个 Go 的 `time.Time`
- 再统一使用 UTC 时区，减少时区混乱

### 5. RFC3339 是什么

`RFC3339` 是接口输出格式，不是数据库存储格式。

例如：

- `2026-04-17T09:30:00Z`

这是把 `time.Time` 格式化之后得到的字符串。

所以当前项目里一条时间数据的典型流转是：

- PostgreSQL `TIMESTAMPTZ`
- Go `time.Time` / `sql.NullTime`
- HTTP 响应里的 `RFC3339 string`

### 6. `created_at` 和 `occurred_at` 有什么区别

- `created_at`
  - 记录是什么时候写进数据库的
- `occurred_at`
  - 业务事件是什么时候发生的

这两个字段看起来都像“时间”，但语义不同。

当前项目里：

- `audit_logs` 同时保留了 `occurred_at` 和 `created_at`
- `task_status_histories` 当前只有 `created_at`

这说明：

- 它已经能表达“记录创建时间”
- 但还没有单独表达“状态变化业务发生时间”

L2 阶段这是可以接受的，继续加 `occurred_at` 会更整齐，但当前收益不够大。

## 【代码/场景对应】

当前项目中直接对应：

- [migrations/0001_init.sql](../../migrations/0001_init.sql)
- [internal/model/auditlogmodel.go](../../internal/model/auditlogmodel.go)
- [internal/model/taskexecutionmodel.go](../../internal/model/taskexecutionmodel.go)
- [internal/logic/tasks/gettaskstatushistorieslogic.go](../../internal/logic/tasks/gettaskstatushistorieslogic.go)

## 【易错点】

- 误以为 `NOW()` 返回 Unix 时间戳。
- 把数据库时间类型误写成 `string`。
- 把 `RFC3339` 误认为数据库存储格式。
- 把 `updated_at` 误当成业务动作的主时间字段。

## 【关联知识】

- [NULL、默认值与空字符串](./null_default_and_empty_string.md)
- [`sql.NullString`、`*string` 和普通 `string` 的区别](./sql_nullstring_vs_ptr_string.md)
- [AgentOps 中 logic 层与 model 层的边界](../project/logic_model_boundary_in_agentops.md)

