## ADDED Requirements

### Requirement: Response generation via EventBus

The system SHALL publish the response generation step as an EventBus topic `rag.query.response`, allowing external subscribers to replace the default LLM response generation.

#### Scenario: Default subscriber generates response

- **WHEN** a query is published to `rag.query.response` with `{"query": "What is LightRAG?", "context": "...", "stream": false}`
- **THEN** the default subscriber calls the LLM with the assembled prompt and returns `{"response": "LightRAG is a..."}`

#### Scenario: Streaming response

- **WHEN** the request includes `stream: true`
- **THEN** the default subscriber returns a streaming response iterator

#### Scenario: External LLM subscriber

- **WHEN** an external subscriber registers for `rag.query.response`
- **THEN** the subscriber can use a different LLM or response generation strategy

#### Scenario: Custom prompt template

- **WHEN** the request includes `user_prompt: "Answer in bullet points"`
- **THEN** the default subscriber injects the user prompt into the system prompt template

### Requirement: Response topic schema

The system SHALL register a topic schema for `rag.query.response` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.response` is listed with inputs (query, context, stream, response_type, conversation_history, user_prompt), outputs (response), and Go/Python/curl code examples
