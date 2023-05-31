package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/gRPCspec"
	"google.golang.org/grpc"
	"log"
	"net"
	"strconv"
)

type StreamServer struct {
	gRPCspec.UnimplementedQueueServiceServer
}

var ExternalPort = flag.String("p", "50505", "port that will be exposed to receive the frames")

func main() {
	port, err := strconv.Atoi(*ExternalPort)
	if err != nil {
		log.Fatalf("%v", err)
	}
	serverListener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	gRPCspec.RegisterQueueServiceServer(s, &StreamServer{})
	fmt.Println("Press ctrl+c to kill the app")
	if err := s.Serve(serverListener); err != nil {
		log.Println("gRPC server error")
		log.Println(err)
		return
	}
}

func (*StreamServer) NextFrame(ctx context.Context, in *gRPCspec.Frame) (*gRPCspec.Frame, error) {
	return in, nil
}
