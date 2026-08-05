package identity

import (
	"context"

	identitypb "velocity/internal/transport/grpc/proto/identity/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client identitypb.AuthServiceClient
}

func New(addr string) (*Client, error) {

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: identitypb.NewAuthServiceClient(conn),
	}, nil
}

func (c *Client) ValidateToken(
	ctx context.Context,
	token string,
) (*identitypb.ValidateTokenResponse, error) {

	return c.client.ValidateToken(
		ctx,
		&identitypb.ValidateTokenRequest{
			Token: token,
		},
	)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
