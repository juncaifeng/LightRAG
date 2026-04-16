```bash
# Publish a KG search event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.kg_search",
    "inputs": {
      "hl_keywords": "[\"RAG framework\"]",
      "ll_keywords": "[\"LightRAG\"]",
      "mode": "mix",
      "top_k": "60"
    }
  }'
```
