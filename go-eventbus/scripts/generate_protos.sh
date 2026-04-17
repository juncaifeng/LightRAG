#!/bin/bash
set -e

# LightRAG EventBus SDK — Multi-language Protobuf Code Generator
# Usage: ./scripts/generate_protos.sh
# Prerequisites: protoc, protoc-gen-go, protoc-gen-go-grpc, python grpcio-tools

PROTO_DIR="proto"
TOPIC_DIR="proto/topics"
GO_OUT="sdk/v1/go"
PYTHON_OUT="sdk/v1/python"

echo "=== Generating SDKs ==="

# --- Go: EventBus protocol ---
echo "[Go] Generating EventBus protocol ..."
protoc \
  -I "${PROTO_DIR}" \
  --go_out="${GO_OUT}" --go_opt=paths=source_relative \
  --go-grpc_out="${GO_OUT}" --go-grpc_opt=paths=source_relative \
  lightrag_eventbus.proto

# --- Go: Topic data models ---
echo "[Go] Generating Topic models ..."
mkdir -p "${GO_OUT}/topics"
protoc \
  -I "${TOPIC_DIR}" \
  --go_out="${GO_OUT}/topics" --go_opt=paths=source_relative \
  insert.proto query.proto
echo "[Go] Done."

# --- Python: EventBus protocol ---
echo "[Python] Generating EventBus protocol ..."
python -m grpc_tools.protoc \
  -I "${PROTO_DIR}" \
  --python_out="${PYTHON_OUT}" \
  --grpc_python_out="${PYTHON_OUT}" \
  lightrag_eventbus.proto

# --- Python: Topic data models ---
echo "[Python] Generating Topic models ..."
mkdir -p "${PYTHON_OUT}/topics"
python -m grpc_tools.protoc \
  -I "${TOPIC_DIR}" \
  --python_out="${PYTHON_OUT}/topics" \
  --grpc_python_out="${PYTHON_OUT}/topics" \
  insert.proto query.proto
echo "[Python] Done."

# --- Rust (uses tonic-build via cargo) ---
if command -v cargo &> /dev/null; then
  echo "[Rust] Building via cargo (tonic-build) ..."
  (cd sdk/v1/rust && cargo build 2>&1)
  echo "[Rust] Done."
else
  echo "[Rust] Skipped: cargo not found. Install Rust toolchain to generate Rust SDK."
fi

# --- TypeScript/Node.js: Topic data models ---
TS_OUT="sdk/v1/node/src/topics"
if command -v npx &> /dev/null; then
  echo "[TypeScript] Generating Topic models to ${TS_OUT}/ ..."
  mkdir -p "${TS_OUT}"
  npx protoc \
    --ts_proto_out="${TS_OUT}" \
    --ts_proto_opt=outputJsonMethods=false,esModuleInterop=true \
    --proto_path="${TOPIC_DIR}" \
    "${TOPIC_DIR}/insert.proto" \
    "${TOPIC_DIR}/query.proto"
  echo "[TypeScript] Done."
else
  echo "[TypeScript] Skipped: npx not found."
fi

# --- Java (requires protoc-gen-grpc-java plugin) ---
if command -v protoc-gen-grpc-java &> /dev/null || [ -f "${GRPC_JAVA_PLUGIN}" ]; then
  JAVA_OUT="sdk/v1/java/src/main/java"
  PLUGIN="${GRPC_JAVA_PLUGIN:-protoc-gen-grpc-java}"
  echo "[Java] Generating to ${JAVA_OUT}/ ..."
  protoc \
    -I "${PROTO_DIR}" \
    --java_out="${JAVA_OUT}" \
    --plugin=protoc-gen-grpc-java="${PLUGIN}" \
    --grpc-java_out="${JAVA_OUT}" \
    lightrag_eventbus.proto
  protoc \
    -I "${TOPIC_DIR}" \
    --java_out="${JAVA_OUT}" \
    insert.proto query.proto
  echo "[Java] Done."
else
  echo "[Java] Skipped: protoc-gen-grpc-java not found. See sdk/v1/java/README.md for setup."
fi

echo "=== All done ==="
