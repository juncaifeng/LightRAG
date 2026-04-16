# Publish a vector search event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.vector_search",
    "inputs": {
      "query": "What is LightRAG?",
      "top_k": "20",
      "enable_rerank": "true"
    }
  }'
