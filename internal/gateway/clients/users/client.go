package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/ozankenangungor/go-commerce/api/gen/go/common/v1"
	usersv1 "github.com/ozankenangungor/go-commerce/api/gen/go/users/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps users.v1 gRPC calls used by the API gateway.
type Client struct {
	conn   *grpc.ClientConn
	client usersv1.UserServiceClient
}

// ValidateAccessTokenError represents a contract-level auth error returned by user service.
type ValidateAccessTokenError struct {
	ErrCode    string
	ErrMessage string
}

func (e *ValidateAccessTokenError) Error() string {
	if e == nil {
		return "user service validation failed"
	}
	if e.ErrMessage == "" {
		return fmt.Sprintf("user service validation failed: %s", e.ErrCode)
	}
	return fmt.Sprintf("user service validation failed: %s (%s)", e.ErrCode, e.ErrMessage)
}

// Code returns the stable contract error code.
func (e *ValidateAccessTokenError) Code() string {
	if e == nil {
		return ""
	}
	return e.ErrCode
}

// NewClient creates a users service gRPC client for local development.
func NewClient(ctx context.Context, addr string, dialTimeout time.Duration) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("dial context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("dial context: %w", err)
	}
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("users grpc address is required")
	}
	if dialTimeout <= 0 {
		return nil, fmt.Errorf("grpc dial timeout must be > 0")
	}

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: dialTimeout,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial user service grpc: %w", err)
	}

	return &Client{
		conn:   conn,
		client: usersv1.NewUserServiceClient(conn),
	}, nil
}

// Close closes the underlying grpc connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ValidateAccessToken validates a bearer token via users.v1.UserService.
func (c *Client) ValidateAccessToken(ctx context.Context, accessToken string, requestID string) (string, []string, error) {
	if c == nil || c.client == nil {
		return "", nil, errors.New("users grpc client is not initialized")
	}
	if strings.TrimSpace(accessToken) == "" {
		return "", nil, errors.New("access token is required")
	}

	resp, err := c.client.ValidateAccessToken(ctx, &usersv1.ValidateAccessTokenRequest{
		Ctx: &commonv1.RequestContext{
			RequestId: requestID,
		},
		AccessToken: accessToken,
	})
	if err != nil {
		return "", nil, fmt.Errorf("validate access token rpc: %w", err)
	}
	if resp == nil {
		return "", nil, errors.New("validate access token rpc returned nil response")
	}

	if resp.GetError() != nil && resp.GetError().GetCode() != "" {
		return "", nil, &ValidateAccessTokenError{
			ErrCode:    resp.GetError().GetCode(),
			ErrMessage: resp.GetError().GetMessage(),
		}
	}

	roles := append([]string(nil), resp.GetRoles()...)
	return resp.GetUserId(), roles, nil
}
