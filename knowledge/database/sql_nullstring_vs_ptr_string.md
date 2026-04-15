# 【主题】
`sql.NullString`、`*string` 和普通 `string` 的区别

一句话概述：

这三种写法的核心差别，不是“语法风格”，而是它们能表达多少种状态。

## 【所属分类】
数据库 / Go 与 SQL 交互

## 【核心结论】

- `string` 只能稳定表达“空字符串”和“非空字符串”
- `*string` 可以表达“缺席”和“有值”
- `sql.NullString` 用来精确承接数据库里的 `NULL`

## 【展开解释】

### 1. 普通 `string`

```go
var s string
```

零值是：

```go
""
```

它不能表达：

- 这个字段没传
- 这个字段在数据库里是 `NULL`

它只能表达：

- `""`
- `"ok"`

### 2. `*string`

```go
var s *string
```

它可以表达：

- `nil`
  - 没传值 / 缺席
- `ptr -> ""`
  - 明确传了空字符串
- `ptr -> "ok"`
  - 明确传了非空字符串

这很适合 HTTP 请求层或业务层的“可选字段”。

### 3. `sql.NullString`

```go
type NullString struct {
	String string
	Valid  bool
}
```

它是标准库 `database/sql` 提供的类型，用来表示：

- 这列是不是 `NULL`
- 如果不是 `NULL`，字符串值是什么

它能表达：

- `Valid=false`
  - 数据库值是 `NULL`
- `Valid=true, String=""`
  - 数据库值是空字符串
- `Valid=true, String="ok"`
  - 数据库值是普通字符串

所以 `sql.NullString` 特别适合：

- 从数据库读可空字符串列
- 向数据库写需要明确区分 `NULL` 和非 `NULL` 的列

### 当前项目里怎么选

- 请求体是否缺席：优先考虑 `*string`
- 数据库列是否可能为 `NULL`：优先考虑 `sql.NullString`
- 明确不需要区分缺席和空字符串：普通 `string` 就够

## 【代码/场景对应】

当前项目相关场景：

- [internal/model/approvalrecordmodel.go](../../internal/model/approvalrecordmodel.go)
- [knowledge/database/null_default_and_empty_string.md](./null_default_and_empty_string.md)

当前 `ApprovalRecord.Comment` 还是普通 `string`，所以它只能稳定传出空字符串，不能表达 SQL `NULL`。

## 【易错点】

- 以为 `string` 的零值等于数据库 `NULL`
- 以为 `*string` 和 `sql.NullString` 完全可以互换
- 用特殊哨兵字符串表示“字段缺席”

## 【关联知识】

- [NULL、默认值与空字符串](./null_default_and_empty_string.md)
- Go 零值
- `database/sql`
