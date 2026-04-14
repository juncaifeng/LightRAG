package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"google.golang.org/grpc"
	pb "github.com/HKUDS/LightRAG/go-eventbus/proto/eventbus/v1"
	"github.com/HKUDS/LightRAG/go-eventbus/server"
)

func main() {
	// Start pprof server for observability (memory, CPU, goroutines)
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	busServer := server.NewEventBusServer()
	
	pb.RegisterEventBusServer(s, busServer)
	
	log.Printf("Event Bus gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
