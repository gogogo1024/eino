## Eino 项目学习与实战路线

目标：系统吃透当前仓库，配合测试驱动练习，顺带提升 Go 并发与工程能力。

---

## 行动清单

- 全局把握架构与模块职责
- 跑通并按需定向运行关键测试
- 端到端追踪一条数据流（Chain/Graph → Model/Tool → 回调）
- 完成 3 个渐进式小练习（流 / 编排 / Agent）
- 用好 Go 工具链（race、cover、pprof、delve）

---

## 当前状态

- 已跑通全仓库测试：compose、schema、components、flow 等包均 PASS。
- 环境健康，可以直接动手（建议测试添加 `-race` 观察并发问题）。

---

## 项目地图（读什么、为什么）

### schema（底层类型与流）
- 文件：`message.go`、`stream.go`、`tool.go`
- 亮点：泛型流、复制/合并、反压与多路复用、错误传播

### compose（编排内核）
- 文件：`graph.go`、`chain.go`、`workflow.go`、`tool_node.go`、`state.go`、`stream_reader.go`
- 亮点：强类型图、分支/并行、状态、回调注入、选项分发、流拼接/复制

### components（组件抽象）
- 目录：`components/{prompt, model, tool, retriever, document}`
- 亮点：组件接口、Option 设计、流式/非流式范式

### callbacks（切面）
- 文件：`handler_builder.go`、`aspect_inject.go`
- 亮点：`OnStart/End/Error/Stream*` 回调与注入

### flow（内置最佳实践）
- 目录：`flow/react`、`flow/retriever/*`
- 亮点：完全用 Graph 搭 ReAct Agent 的工程化范式

### internal（工具与安全）
- 目录：`internal/safe`、`internal/generic`、`internal/serialization`
- 亮点：panic 捕获、泛型小工具、序列化

---

## 推荐阅读顺序（配合对应测试）

### 1) 流的地基
- 读：`schema/stream.go`
- 跑：`schema/stream_test.go`（复制、合并、转换、并发）

### 2) 编排的骨架
- 读：`compose/graph.go`、`compose/chain.go`、`compose/branch.go`、`compose/stream_*`
- 跑：`compose/graph_test.go`、`compose/chain_test.go`、`compose/branch_test.go`、`compose/stream_*_test.go`

### 3) 组件抽象与模板
- 读：`components/prompt/chat_template.go`、各 `interface.go`
- 跑：`components/prompt/chat_template_test.go`、各组件 `option_test.go`

### 4) 回调/切面与选项分发
- 读：`callbacks/*`、`compose/graph_call_options.go`
- 跑：`callbacks/interface_test.go`、`compose/graph_call_options_test.go`

### 5) 实战范式（ReAct）
- 读：`flow/react/*`
- 跑：react 包相关测试

---

## 三个小练习（循序渐进）

### 练习 A：流式编程手感
- 目标：用 `Pipe` + `StreamReaderWithConvert` 过滤空值并 map 到新结构；同时用 `Copy(2)` 做双订阅消费。
- 参考：`schema/stream_test.go: TestStreamReaderWithConvert`、`TestNewStreamCopy`

### 练习 B：编排最小闭环
- 目标：用 compose 里的 Lambda（见 `types_lambda.go` 与其测试）串一个“模板 → 处理 → 汇总”的 Chain；再改造为 Graph，增加一个 Branch。
- 参考：`types_lambda_test.go`、`compose/chain_test.go`、`compose/graph_test.go`

### 练习 C：工具调用路径
- 目标：阅读 `tool_node.go` 与 `schema/tool.go`，写一个 fake Tool（返回固定 JSON），在 Graph 中让模型分支走到 `ToolsNode`，再将结果汇总为最终消息（可用 Lambda 模拟模型决策）。
- 参考：`compose/tool_node_test.go`

> 如需，我可以为每个练习在 `playground/` 下生成最小测试样例，直接 `go test` 驱动。

---

## 用好 Go 工具链（即时实操）

### 只跑关键用例（加速迭代）
- 示例：
	- 只跑某包：`go test ./schema -run TestStream -v`
	- 全量但过滤：`go test ./... -run Stream -v`

### 覆盖率总览（聚焦薄弱点）
- 示例：
	- 生成覆盖率：`go test ./... -coverprofile=cover.out`
	- 查看函数级：`go tool cover -func=cover.out`
	- 打开 HTML：`go tool cover -html=cover.out`

### 查看数据竞争
- 对流与并发逻辑价值很大：`go test ./... -race`

### 调试与火焰图
- 调试：`dlv test ./compose`（在 `graph_run.go` 等处下断点）
- CPU 分析（示例）：
	- 在目标测试中加开关（避免长跑污染），运行：`go test ./compose -run YourCase -cpuprofile cpu.out`
	- 分析：`go tool pprof ./compose.test cpu.out`

---

## 借此提升的关键点

- 泛型与强类型设计：compose 的类型参数在编排期保证 I/O 类型安全
- 并发与流：schema 的 Copy/Merge/Convert，`reflect.Select` 多路复用，`atomic` 计数与资源回收
- 选项与切面：函数式 Option、回调注入与跨组件关注点
- 错误与健壮性：`internal/safe` 的 panic 包装、流中错误的传递与终止语义

---

## 下一步你可以立刻做的事

- 按“推荐阅读顺序”过一遍对应源码与测试（建议 `-race`）
- 在 `playground/` 选择一个小练习（A/B/C）开干；确定后我可生成对应最小测试/样例，直接 `go test` 即见结果，并带覆盖率与 race 检查

> 需要我先为练习 A 创建示例（已部分存在于 `playground/stream_kata_test.go`），或继续补 B/C 的最小可运行用例吗？