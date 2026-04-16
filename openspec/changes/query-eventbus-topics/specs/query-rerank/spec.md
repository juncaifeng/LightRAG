## ADDED Requirements

### Requirement: Rerank via EventBus

The system SHALL publish the rerank step as an EventBus topic `rag.query.rerank`, allowing external subscribers to replace the default reranking logic.

#### Scenario: Default subscriber reranks chunks

- **WHEN** a query is published to `rag.query.rerank` with `{"query": "What is LightRAG?", "chunks": [...]}`
- **THEN** the default subscriber applies the configured reranker model and returns `{"ranked_chunks": [...]}` in relevance-sorted order

#### Scenario: External reranker subscriber

- **WHEN** an external subscriber registers for `rag.query.rerank`
- **THEN** the subscriber can use a different reranker model (e.g., BAAI/bge-reranker-v2-m3, Jina, Cohere) and return ranked chunks

#### Scenario: Rerank skipped when disabled

- **WHEN** the request includes `enable_rerank: false`
- **THEN** the default subscriber returns chunks in original order without reranking

### Requirement: Rerank topic schema

The system SHALL register a topic schema for `rag.query.rerank` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.rerank` is listed with inputs (query, chunks, enable_rerank), outputs (ranked_chunks), and Go/Python/curl code examples
