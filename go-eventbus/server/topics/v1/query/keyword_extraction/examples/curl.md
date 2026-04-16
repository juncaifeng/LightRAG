```bash
# Publish a keyword extraction event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.keyword_extraction",
    "inputs": {
      "query": "What is LightRAG?"
    }
  }'
```
