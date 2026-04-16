# cURL Example — rag.insert.ocr

```bash
# Publish an OCR event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.insert.ocr",
    "inputs": {
      "language": "auto"
    }
  }'
```
