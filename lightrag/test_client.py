import grpc
import time
import uuid
import sys
import os

# Add generated proto files to path
sys.path.append('/workspace/lightrag')

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc

def run_test_client():
    # Connect to Go Event Bus
    channel = grpc.insecure_channel('localhost:50051')
    stub = pb2_grpc.EventBusStub(channel)
    
    correlation_id = str(uuid.uuid4())
    print(f"[*] Publishing event with Correlation ID: {correlation_id}")
    
    inputs = {
        "query": b"AI"
    }
    
    envelope = pb2.EventEnvelope(
        topic="rag.query.query_expansion",
        correlation_id=correlation_id,
        trace_id="test-trace-001",
        deadline_timestamp=int((time.time() + 5.0) * 1000), # 5 seconds deadline
        priority=1,
        source_service="lightrag-python-client",
        inputs=inputs
    )
    
    # Measure P50/P99 locally in client for MVP
    start_time = time.time()
    
    try:
        reply = stub.PublishAndWait(envelope)
        elapsed_ms = int((time.time() - start_time) * 1000)
        
        print(f"[+] Received Gathered Reply in {elapsed_ms}ms:")
        print(f"    - Strategy: {pb2.SubscriberReply.MergeStrategy.Name(reply.strategy)}")
        print(f"    - Weight: {reply.weight}")
        print(f"    - Error Code: {reply.error_code}")
        
        for k, v in reply.outputs.items():
            print(f"    - Output[{k}]: {v.decode('utf-8')}")
            
    except grpc.RpcError as e:
        print(f"[-] RPC failed: {e.code()} - {e.details()}")

if __name__ == '__main__':
    run_test_client()