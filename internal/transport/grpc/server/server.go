package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"velocity/internal/service/userservice"
	velocityv1 "velocity/internal/transport/grpc/proto/velocity/v1"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func New(
    userService *userservice.Service,
) (*Server, error) {

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()

	velocityv1.RegisterVelocityServiceServer(
		grpcServer,
		NewUserServer(userService),
	)

	return &Server{
		grpcServer: grpcServer,
		listener: lis,
	}, nil
}

func (s *Server) Start() error {
	fmt.Println("Velocity gRPC listening on :50053")
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}