# LightRAG EventBus Java SDK

## 生成方法

需要先安装：
1. `protoc` — https://github.com/protocolbuffers/protobuf/releases
2. `protoc-gen-grpc-java` 插件 — https://github.com/grpc/grpc-java/blob/master/README.md

### 生成命令

```bash
# 从 go-eventbus/ 目录运行
protoc \
  -I proto \
  --java_out=sdk/v1/java/src/main/java \
  --plugin=protoc-gen-grpc-java=/path/to/protoc-gen-grpc-java \
  --grpc-java_out=sdk/v1/java/src/main/java \
  proto/lightrag_eventbus.proto
```

### 使用 Maven 构建

```bash
cd sdk/v1/java
mvn clean compile
```
