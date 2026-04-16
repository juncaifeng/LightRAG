# Publish a rerank event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.rerank",
    "inputs": {
      "query": "What is LightRAG?",
      "chunks": "[{\"chunk_id\": \"chunk-001\", \"content\": \"LightRAG is...\"}]",
      "enable_rerank": "true"
    }
  }'
