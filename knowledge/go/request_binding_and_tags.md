# 【主题】
Go 请求绑定、导出字段与 tag

一句话概述：

HTTP 请求绑定到结构体时，Go 字段是否可导出决定能不能被稳定处理，tag 决定 JSON/path 里的键名映射。

## 【所属分类】
Go语言基础 / 标准库  
后端开发 / HTTP / REST

## 【核心结论】

- Go 结构体字段首字母大写，表示导出，便于包外访问和运行时处理。
- ``json:"reviewerId"`` 控制 JSON 键名，不决定字段是否导出。
- ``path:"id"`` 这类 tag 用于把 URL 路径参数映射到结构体字段。
- 请求绑定这类能力通常依赖运行时读取结构体字段和 tag，背后常和反射有关。

## 【展开解释】

在项目里，请求结构体长这样：

```go
type ApproveTaskReq struct {
	Id         string `path:"id"`
	ReviewerId string `json:"reviewerId"`
	Comment    string `json:"comment,optional"`
}
```

这里需要分清两层：

- `ReviewerId`
  Go 字段名，首字母大写，表示导出。
- `json:"reviewerId"`
  表示 HTTP JSON body 里使用的小驼峰键名。

这两者不是一回事。

如果前端传：

```json
{
  "reviewerId": "u1001",
  "comment": "ok"
}
```

绑定到 Go 结构体后：

- `ReviewerId == "u1001"`
- `Comment == "ok"`

## 【代码/场景对应】

当前项目中直接对应：

- [agentops.api](../../agentops.api)
- [internal/types/types.go](../../internal/types/types.go)
- [internal/handler/tasks/approvetaskhandler.go](../../internal/handler/tasks/approvetaskhandler.go)

`httpx.Parse(r, &req)` 会把 path/body 中的数据绑定进结构体。

## 【易错点】

- 以为 tag 可以代替字段导出。不能。
- 以为 `json:"reviewerId"` 会让 Go 字段名也变成小写。不会。
- 以为 path 参数和 JSON 参数都是同一种绑定。不是，它们只是最后都落在同一个结构体里。

## 【关联知识】

- 反射
- handler 层请求解析
- Go 结构体 tag
- JSON 编码/解码
- [approval_records 当前语义与 L2 目标](../project/approval_records_semantics.md)
