## 行动清单

- 全局把握架构与模块职责
- 跑通并按需定向运行关键测试
- 端到端追踪一条数据流（Chain/Graph → Model/Tool → 回调）
- 完成 3 个渐进式小练习（流/编排/Agent）
- 用好 Go 工具链（race、cover、pprof、delve）

## 当前状态

- 已运行全仓库测试：全部包 PASS（包含 compose、schema、components、flow 等）。环境 OK，可直接动手。

## 项目地图（读什么、为什么）

### schema（底层类型与流）

- 文件：`schema/message.go`、`schema/stream.go`、`schema/tool.go`
- 亮点：泛型流、复制/合并、反压与多路复用、错误传播

### compose（编排内核）

- 文件：`compose/graph.go`、`compose/chain.go`、`compose/workflow.go`、`compose/tool_node.go`、`compose/state.go`、`compose/stream_reader.go`
- 亮点：强类型图、分支/并行、状态、回调注入、选项分发、流拼接/复制

### components（组件抽象）

- 目录：`components/{prompt,model,tool,retriever,document}`
- 亮点：各组件接口、Option 设计、流式/非流式范式

### callbacks（切面）

- 文件：`callbacks/handler_builder.go`、`callbacks/aspect_inject.go`
- 亮点：OnStart/End/Error/Stream* 回调与注入

### flow（内置最佳实践）

- 目录：`react`、`flow/retriever/*`
- 亮点：用 Graph 搭 ReAct Agent 的工程化范式

### internal（工具与安全）

- 目录：`internal/safe`、`internal/generic`、`internal/serialization`
- 亮点：panic 捕获、泛型小工具、序列化

## 推荐阅读顺序（配合对应测试）

### 流的地基

- 读：`schema/stream.go`
- 跑：`schema/stream_test.go`（复制、合并、转换、并发）

### 编排的骨架

- 读：`compose/graph.go`、`compose/chain.go`、`compose/branch.go`、`compose/stream_*`
- 跑：`compose/graph_test.go`、`compose/chain_test.go`、`compose/branch_test.go`、`compose/stream_*_test.go`

### 组件抽象与模板

- 读：`components/prompt/chat_template.go`、`components/*/interface.go`
- 跑：`components/prompt/chat_template_test.go`、各组件 `option_test.go`

### 回调/切面与选项分发

- 读：`callbacks/*`、`compose/graph_call_options.go`
- 跑：`callbacks/interface_test.go`、`compose/graph_call_options_test.go`

### 实战范式（ReAct）

- 读：`react.go`
- 跑：`react` 包测试

## 三个小练习（循序渐进）

### 练习 A：流式编程手感

- 目标：用 `Pipe` + `StreamReaderWithConvert` 过滤空值并 map 到新结构；同时用 `Copy(2)` 做双订阅消费。
- 参考：`schema/stream_test.go`：`TestStreamReaderWithConvert`、`TestNewStreamCopy`

### 练习 B：编排最小闭环

- 目标：用 compose 里的 Lambda（见 `types_lambda.go` 与其测试）串一个“模板 → 处理 → 汇总”的 Chain；再改造为 Graph，增加一个 Branch。
- 参考：`compose/types_lambda_test.go`、`compose/chain_test.go`、`compose/graph_test.go`

### 练习 C：工具调用路径

- 目标：阅读 `compose/tool_node.go` 与 `schema/tool.go`，写一个 fake Tool（返回固定 JSON），在 Graph 中让模型分支走到 ToolsNode，再将结果汇总为最终消息（可用 Lambda 模拟模型决策）。
- 参考：`compose/tool_node_test.go`
行动清单
全局把握架构与模块职责
跑通并按需定向运行关键测试
端到端追踪一条数据流（Chain/Graph → Model/Tool → 回调）
完成3个渐进式小练习（流/编排/Agent）
用好 Go 工具链（race、cover、pprof、delve）
当前状态
我已运行全仓库测试：全部包 PASS（包含 compose、schema、components、flow 等）。你的环境OK，可以直接动手。
项目地图（读什么、为什么）
schema（底层类型与流）
文件：message.go、stream.go、tool.go
亮点：泛型流、复制/合并、反压与多路复用、错误传播
compose（编排内核）
文件：graph.go、chain.go、workflow.go、tool_node.go、state.go、stream_reader.go
亮点：强类型图、分支/并行、状态、回调注入、选项分发、流拼接/复制
components（组件抽象）
目录：components/{prompt,model,tool,retriever,document}
亮点：各组件接口、Option 设计、流式/非流式范式
callbacks（切面）
文件：handler_builder.go、aspect_inject.go
亮点：OnStart/End/Error/Stream* 回调与注入
flow（内置最佳实践）
目录：react、flow/retriever/*
亮点：完全用 Graph 搭 ReAct Agent 的工程化范式
internal（工具与安全）
目录：safe、generic、serialization
亮点：panic 捕获、泛型小工具、序列化
 

### callbacks（切面）

- 文件：`callbacks/handler_builder.go`、`callbacks/aspect_inject.go`
- 亮点：OnStart/End/Error/Stream* 回调与注入

### flow（内置最佳实践）

- 目录：`react`、`flow/retriever/*`
- 亮点：完全用 Graph 搭 ReAct Agent 的工程化范式

### internal（工具与安全）

- 目录：`internal/safe`、`internal/generic`、`internal/serialization`
- 亮点：panic 捕获、泛型小工具、序列化

## 推荐阅读顺序（配合对应测试）

### 流的地基

- 读：`schema/stream.go`
- 跑：`schema/stream_test.go`（复制、合并、转换、并发）

### 编排的骨架

- 读：`compose/graph.go`、`compose/chain.go`、`compose/branch.go`、`compose/stream_*`
- 跑：`compose/graph_test.go`、`compose/chain_test.go`、`compose/branch_test.go`、`compose/stream_*_test.go`

### 组件抽象与模板

- 读：`components/prompt/chat_template.go`、`components/*/interface.go`
- 跑：`components/prompt/chat_template_test.go`、各组件 `option_test.go`

### 回调/切面与选项分发

- 读：`callbacks/*`、`compose/graph_call_options.go`
- 跑：`callbacks/interface_test.go`、`compose/graph_call_options_test.go`

### 实战范式（ReAct）

- 读：`react.go`
- 跑：`react` 包测试

## 三个小练习（循序渐进）

### 练习 A：流式编程手感

- 目标：用 `Pipe` + `StreamReaderWithConvert` 过滤空值并 map 到新结构；同时用 `Copy(2)` 做双订阅消费。
- 参考：`schema/stream_test.go`：`TestStreamReaderWithConvert`、`TestNewStreamCopy`

### 练习 B：编排最小闭环

- 目标：用 compose 里的 Lambda（见 `types_lambda.go` 与其测试）串一个“模板 → 处理 → 汇总”的 Chain；再改造为 Graph，增加一个 Branch。
- 参考：`compose/types_lambda_test.go`、`compose/chain_test.go`、`compose/graph_test.go`

### 练习 C：工具调用路径

- 目标：阅读 `compose/tool_node.go` 与 `schema/tool.go`，写一个 fake Tool（返回固定 JSON），在 Graph 中让模型分支走到 ToolsNode，再将结果汇总为最终消息（可用 Lambda 模拟模型决策）。
- 参考：`compose/tool_node_test.go`