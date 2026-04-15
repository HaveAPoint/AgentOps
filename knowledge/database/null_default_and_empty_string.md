# 【主题】
NULL、默认值与空字符串

一句话概述：

SQL 里的 `NULL`、空字符串 `''`、默认值 `DEFAULT` 是三件完全不同的事。

## 【所属分类】
数据库 / SQL / PostgreSQL

## 【核心结论】

- `NULL` 不等于空字符串。
- 显式传空字符串，不会触发 SQL 默认值。
- 显式传 `NULL`，也不会自动触发默认值。
- `string` 的 Go 零值是 `""`，不是 `NULL`。
- 想表达“缺席”和“空字符串”两种不同语义时，普通 `string` 不够。

## 【展开解释】

当前表定义里有：

```sql
comment TEXT NOT NULL DEFAULT ''
```

而当前 Go 插入代码里是：

```go
record.Comment
```

如果 `record.Comment == ""`，数据库收到的是显式的空字符串：

- 存进去的是 `''`
- 不是 `NULL`
- 也不是“默认值自动帮你补出来的 `''`”

默认值什么时候触发？

- `INSERT` 根本不写这个列
- 或者显式写 `DEFAULT`

如果以后要区分：

- 没传 comment
- 传了空字符串
- 传了非空字符串

更适合的 Go 表达通常是：

- `*string`
- 或数据库层 `sql.NullString`

## 【代码/场景对应】

当前项目中直接对应：

- [migrations/0001_init.sql](../../migrations/0001_init.sql)
- [internal/model/approvalrecordmodel.go](../../internal/model/approvalrecordmodel.go)

## 【易错点】

- 以为“字段空了数据库就会自动套默认值”。不会。
- 以为 `""` 和 `NULL` 差不多。不是。
- 以为可以用某个特殊字符串当“没传值”的哨兵。工程上不推荐，会污染真实数据。

## 【关联知识】

- Go 零值
- `*string`
- `sql.NullString`
- PostgreSQL `DEFAULT`
- [事务、Rollback/Commit 与 FOR UPDATE](./transactions_and_row_locks.md)
