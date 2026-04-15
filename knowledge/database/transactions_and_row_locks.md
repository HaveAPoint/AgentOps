# 【主题】
事务、Rollback/Commit 与 FOR UPDATE

一句话概述：

`FOR UPDATE` 负责锁住当前行，事务负责让这把锁持续到整组业务操作完成。

## 【所属分类】
数据库 / PostgreSQL / 事务 / 锁

## 【核心结论】

- `SELECT ... FOR UPDATE` 锁的是命中的行，不是整张表。
- 没有事务时，`FOR UPDATE` 的保护效果通常非常短。
- `defer tx.Rollback()` 是常见兜底写法：成功路径无害，失败路径关键。
- 并发 approve 同一条 task 的核心问题是同一行并发更新竞争，不是幻读。

## 【展开解释】

典型模式：

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{})
if err != nil {
	return err
}
defer func() {
	_ = tx.Rollback()
}()
```

然后在同一个事务里：

1. `FindByIDForUpdate`
2. 检查当前 `status`
3. `UpdateStatus`
4. 写审批记录 / 审计日志
5. `Commit`

为什么不能只写 `FOR UPDATE`，不写事务？

- 因为锁要活在事务里。
- 如果没有显式事务，这条 SQL 往往运行在一个隐式事务中。
- 语句执行完，隐式事务结束，锁也就释放了。

## 【代码/场景对应】

当前项目中直接对应：

- [internal/model/taskmodel.go](../../internal/model/taskmodel.go)
- [internal/logic/tasks/approvetasklogic.go](../../internal/logic/tasks/approvetasklogic.go)

`FindByIDForUpdate` 的 SQL：

```sql
SELECT ... FROM tasks WHERE id = $1 FOR UPDATE
```

## 【易错点】

- 把当前 approve 场景误认为幻读。
- 以为 `FOR UPDATE` 能替代事务。
- 以为 `defer tx.Rollback()` 只有失败时才会调用。其实函数退出时都会调用，只是成功提交后它通常不会再产生实际回滚效果。
- 以为 `BeginTx` 失败后 defer 也会执行。不会，因为代码还没走到 `defer` 那一行。

## 【关联知识】

- check-then-act 并发问题
- 丢失更新
- 隐式事务与显式事务
- PostgreSQL 行级锁
- [NULL、默认值与空字符串](./null_default_and_empty_string.md)
