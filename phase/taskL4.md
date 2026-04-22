L4 计划（受控执行平台版）

目标

把 L3 的“协作型任务系统”推进成：

- 围绕 creator / reviewer / operator 分工的最小可控执行平台

L4 的本质是：

让“谁创建、谁审批、谁执行”这三条责任链，不只是存在于数据库和接口里，而是真正影响执行安全。

L4 核心业务流

- creator 创建代码任务
- reviewer 判断任务是否允许执行
- operator 在受控条件下触发执行
- 平台记录执行过程与结果
- 审计能够回答：谁发起、谁批准、谁执行、改了什么、结果如何

当前实现进度（2026-04-22）

当前 L4 约完成 55%-60%。

已经真实生效：

- 已抽出 `internal/executor` 执行器边界
- `ServiceContext` 已注入 `TaskRunner`
- `StartTask` 已能同步调用 mock executor
- executor 成功后自动收口到 `succeeded`
- executor 失败后自动收口到 `failed`
- 执行 timeout 已接入 `context.WithTimeout`
- repo 白名单已在执行前校验
- `allowedPaths / deniedPaths` 已从快照数据推进到执行路径约束
- `patch` 模式要求 `allowedPaths` 非空
- executor 的 `stdout / stderr / summary` 已汇总进 execution result
- Git changed files 已记录 before / after / new
- 新增越权变更文件会导致执行收口到 `failed`
- `running -> cancelled` 已能做数据库状态收口
- cancel 权限已按状态拆分：`waiting_approval / pending` 由 creator / assigned reviewer / admin 取消，`running` 由 assigned operator / admin 取消

当前只是够用版：

- `newChangedFiles = after - before` 只能识别执行后新增进入 dirty 集合的文件
- 如果执行前某个文件已经 dirty，executor 又继续修改它，当前还不能精确识别这次二次修改
- 要达到更严格的工业级路径控制，后续需要 diff 摘要、diff hash 或执行前后 patch 对比
- `running -> cancelled` 当前只完成数据库收口，还没有真实中断正在执行的 runner / CLI 进程

仍未完成：

- reviewer 审批语义还没有升级成“批准某 operator / mode / repo / path 范围”
- 尚未接入真实 `codex / claudecode` provider
- 还没有完整工作目录沙箱和真实 CLI 进程 kill
- 执行审计仍是文本摘要，尚未结构化记录 diff / changed file 明细

L4 要做的事

1. 抽执行器模块，并围绕 operator 建立输入语义

执行器不应该是“系统自动乱跑”，而应该是：

- operator 在权限允许下，触发一次受控执行

初始 provider 建议至少预留：

- `mock`
- `codex`
- `claudecode`

executor 输入至少要有：

- `task`
- `operator identity`
- `repo context`
- `policy context`
- `execution mode`

2. 让 repo 白名单和路径策略真正生效

L4 之前，`allowed_paths / denied_paths` 主要还是快照数据。

L4 要把它们落到真实执行前校验里：

- 未授权 repo 不允许执行
- 超出允许路径直接拒绝
- 命中拒绝路径直接拒绝

这一步约束的是执行，不只是创建。

3. 把执行模式和审批语义挂钩

至少区分：

- `analyze`
- `patch`

并让 reviewer 审批的内容不再只是“同意执行”，而是：

- 批准某个 operator
- 以某种 mode
- 在某个 repo / path 范围内执行

这样审批才和真实风险绑定。

4. 增加执行保护

至少补：

- timeout
- context cancel
- 工作目录限制
- stdout / stderr 捕获
- 执行失败时 execution 正确收口

如果支持 `running -> cancelled`，这里还需要定义取消如何传递到执行器并安全中止。

5. 强化 Git 执行上下文

在当前 `branch / head / dirty` 的基础上，至少增加一种：

- 变更文件列表
- diff 摘要
- 执行前后仓库状态对比

目的是让平台不仅知道“在哪个 repo 上执行了”，还知道“执行对仓库造成了什么影响”。

6. 补全执行审计

L4 的执行审计至少要能回答：

- 谁创建的任务
- 谁批的
- 谁执行的
- 在哪个 repo
- 用什么 mode
- 是否命中路径限制
- 最终 succeeded 还是 failed
- 失败原因是什么

7. 让执行失败收口到完整责任链

如果 operator 执行失败，平台要同时正确写入：

- execution = failed
- task = failed
- 状态历史里的 actor 与 reason
- 审计日志里的失败信息

这样“失败”不再是一个裸技术错误，而是责任链中的一次失败执行。

8. 补 L4 测试

至少覆盖：

- repo 未授权拒绝
- 路径未授权拒绝
- analyze / patch 模式限制正确
- 超时可终止
- cancel 能中断执行并正确收口
- 执行失败时 task / execution / history 一致
- 执行审计完整

L4 明确不做

- 多 worker 编排
- MQ
- 自动 commit / push / PR 全套
- 重型沙箱平台
- 插件系统
- 多租户
- 复杂策略中心

L4 完成标志

做到下面这些，L4 才算有价值：

- creator / reviewer / operator 三条责任链已经贯通到执行层
- repo / path / mode 三类约束已经真实影响 execution
- reviewer 的审批已经能表达具体执行风险
- operator 的执行行为可以被完整追踪
- 平台能够回答“谁发起、谁批准、谁执行、执行了什么、结果如何”
