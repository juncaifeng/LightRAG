# Proposal: eventbus-phase3-integration

## 背景 (Context)

在 Phase 2 中，我们用 Go 成功构建了一个高性能的独立 Event Bus，并使用 Python Client 跑通了 `PublishAndWait` 及其智能合并策略的验证测试。

现在我们迎来了架构演进的深水区：**如何将这套强大的订阅式总线平滑地嵌入到真实的 LightRAG 核心引擎（如 `operate.py` 和 `lightrag.py`）中，且不破坏现有的单机运行体验。**

## 目标 (Goals)

- 设计并在 LightRAG 中实现一套非侵入式（Non-invasive）的 `EventBusDispatcher`。
- 将 LightRAG 原生硬编码的处理逻辑（例如文本分块 `chunking_by_token_size` 和 实体抽取 `extract_entities`）包装成 `LocalSubscriberAdapter`，注册到总线上。
- 在核心流水线代码中完成**埋点打桩**，将直接函数调用替换为 `event_bus.publish_and_wait()`。
- 确保系统在**没有外部微服务（甚至没起 Go Event Bus 进程）时，能通过内存本地短路机制正常运行**。

## 非目标 (Non-Goals)

- 本阶段**不包含**全面替换掉现有的 LLM/KG 底层存储接口，仅聚焦于 RAG 流水线的“阶段拦截”（Hook）。
- 本阶段不实现专门的业务服务（如真实的 Rust OCR 服务），只关注主引擎架构的改造与留出扩展口。

## 关键原则 (Key Principles)

- **无感降级（Absolute Fallback）**：用户如果不配 Event Bus URL，系统应退化为内存级别的 `LocalDispatcher`，使用户体验零损失。
- **纯粹的无知（Absolute Ignorance）**：核心的 `operate.py` 在发布事件时，绝对不能关心“有谁在处理”以及“如何合并”，必须依赖 Phase 2 中定义的机械合并机制。