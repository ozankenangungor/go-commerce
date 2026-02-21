package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	usersv1 "github.com/ozankenangungor/go-commerce/api/gen/go/users/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService implements users.v1.UserServiceServer.
type UserService struct {
	usersv1.UnimplementedUserServiceServer

	logger zerolog.Logger
	db     *pgxpool.Pool
}

// NewUserService creates a new user service handler.
func NewUserService(logger zerolog.Logger, db *pgxpool.Pool) *UserService {
	return &UserService{
		logger: logger,
		db:     db,
	}
}

func (s *UserService) Register(ctx context.Context, req *usersv1.RegisterRequest) (*usersv1.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *UserService) Login(ctx context.Context, req *usersv1.LoginRequest) (*usersv1.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *UserService) RefreshToken(ctx context.Context, req *usersv1.RefreshTokenRequest) (*usersv1.RefreshTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *UserService) GetProfile(ctx context.Context, req *usersv1.GetProfileRequest) (*usersv1.GetProfileResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *UserService) ValidateAccessToken(ctx context.Context, req *usersv1.ValidateAccessTokenRequest) (*usersv1.ValidateAccessTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
