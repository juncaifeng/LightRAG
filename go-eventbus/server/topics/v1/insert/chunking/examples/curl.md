# Publish a chunking event via the EventBus HTTP API

## Basic usage

```bash
# Publish a chunking event with required and optional inputs
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.insert.chunking",
    "inputs": {
      "content": "Your document content here. This is a long text that needs to be split into smaller chunks for processing.",
      "chunk_token_size": 1200,
      "chunk_overlap_token_size": 100
    }
  }'
```

## With custom splitting delimiter

```bash
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.insert.chunking",
    "inputs": {
      "content": "Chapter 1: Introduction. Chapter 2: Methods. Chapter 3: Results.",
      "chunk_token_size": 500,
      "chunk_overlap_token_size": 50,
      "split_by_character": ".",
      "split_by_character_only": true
    }
  }'
```

## Input fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `content` | string | yes | — | Raw document content |
| `tokenizer` | bytes | yes | — | Serialized Tiktoken tokenizer object |
| `chunk_token_size` | int32 | no | 1200 | Chunk token size |
| `chunk_overlap_token_size` | int32 | no | 100 | Overlap token count |
| `split_by_character` | string | no | — | Character delimiter for splitting |
| `split_by_character_only` | bool | no | false | Split by character only |

## Output fields

| Field | Type | Description |
|-------|------|-------------|
| `chunks` | JSON | Chunk result list: `[{content: string, ...}]` |
