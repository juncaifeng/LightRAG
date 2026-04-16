## ADDED Requirements

### Requirement: Vector search via EventBus

The system SHALL publish the vector search step as an EventBus topic `rag.query.vector_search`, allowing external subscribers to replace the default chunk vector retrieval.

#### Scenario: Default subscriber retrieves chunks

- **WHEN** a query is published to `rag.query.vector_search` with `{"query": "What is LightRAG?", "top_k": 20}`
- **THEN** the default subscriber calls `_get_vector_context()` and returns `{"chunks": [{"chunk_id": "...", "content": "...", "file_path": "..."}]}`

#### Scenario: External vector database subscriber

- **WHEN** an external subscriber registers for `rag.query.vector_search`
- **THEN** the subscriber can query an external vector database (e.g., Pinecone, Weaviate) and return chunks in the same format

#### Scenario: Rerank included in vector search

- **WHEN** the subscriber processes the request with `enable_rerank: true`
- **THEN** the subscriber applies reranking to the retrieved chunks before returning

### Requirement: Vector search topic schema

The system SHALL register a topic schema for `rag.query.vector_search` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.vector_search` is listed with inputs (query, top_k, enable_rerank), outputs (chunks), and Go/Python/curl code examples
