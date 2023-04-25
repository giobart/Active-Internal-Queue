package gRPCspec

import (
	"context"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

type QueueServer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
	UnimplementedQueueServiceServer
}

func (s *QueueServer) NextFrame(ctx context.Context, in *Frame) (*Frame, error) {

	//TODO: your logic goes here! leaving this empty because this library is meant to be the gRPC client only.

	return in, nil
}
