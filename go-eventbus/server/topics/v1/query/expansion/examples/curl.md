```bash
# Publish a query expansion event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.query_expansion",
    "inputs": {
      "hl_keywords": "[\"AI\", \"retrieval\"]",
      "ll_keywords": "[\"LightRAG\", \"RAG\"]",
      "expansion_config": "{\"synonym\": true, \"near_synonym\": true, \"terminology\": true}"
    }
  }'
```
