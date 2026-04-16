## ADDED Requirements

### Requirement: Keyword extraction via EventBus

The system SHALL publish the keyword extraction step as an EventBus topic `rag.query.keyword_extraction`, allowing external subscribers to replace or enhance the default LLM-based keyword extraction.

#### Scenario: Default subscriber extracts keywords

- **WHEN** a query is published to `rag.query.keyword_extraction` with `{"query": "What is LightRAG?"}`
- **THEN** the default subscriber calls `extract_keywords_only()` and returns `{"hl_keywords": [...], "ll_keywords": [...]}`

#### Scenario: External subscriber replaces extraction

- **WHEN** an external subscriber registers for `rag.query.keyword_extraction`
- **THEN** the external subscriber's result is merged with the default subscriber's result according to the merge strategy

#### Scenario: Simple query fallback

- **WHEN** the LLM returns empty keyword lists and `len(query) < 50`
- **THEN** the default subscriber falls back to `ll_keywords = [query]`

### Requirement: Keyword extraction topic schema

The system SHALL register a topic schema for `rag.query.keyword_extraction` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.keyword_extraction` is listed with inputs (query), outputs (hl_keywords, ll_keywords), and Go/Python/curl code examples
