package server

import (
    "context"

    velocityv1 "velocity/internal/transport/grpc/proto/velocity/v1"

    "velocity/internal/service/userservice"
)

type UserServer struct {
    velocityv1.UnimplementedVelocityServiceServer

    userService *userservice.Service
}

func NewUserServer(
    userService *userservice.Service,
) *UserServer {

    return &UserServer{
        userService: userService,
    }
}

func (s *UserServer) CreateUser(
    ctx context.Context,
    req *velocityv1.CreateUserRequest,
) (*velocityv1.CreateUserResponse, error) {

    user, err := s.userService.CreateUser(
        ctx,
        userservice.CreateUserRequest{
            ID:    req.Id,
            Email: req.Email,
        },
    )

    if err != nil {
        return &velocityv1.CreateUserResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }

    return &velocityv1.CreateUserResponse{
        Success: true,
        UserId:  user.ID,
        Email:   user.Email,
    }, nil
}