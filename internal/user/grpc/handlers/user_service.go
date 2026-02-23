package handlers

import (
	"context"
	"errors"
	"strings"

	commonv1 "github.com/ozankenangungor/go-commerce/api/gen/go/common/v1"
	usersv1 "github.com/ozankenangungor/go-commerce/api/gen/go/users/v1"
	userservice "github.com/ozankenangungor/go-commerce/internal/user/service"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthService defines service methods used by the gRPC handler layer.
type AuthService interface {
	Register(ctx context.Context, email, password, name string) (userservice.User, userservice.AuthTokens, *userservice.DomainError, error)
	Login(ctx context.Context, email, password string) (userservice.User, userservice.AuthTokens, *userservice.DomainError, error)
	RefreshToken(ctx context.Context, refreshToken string) (userservice.AuthTokens, *userservice.DomainError, error)
	GetProfile(ctx context.Context, userID string) (userservice.User, *userservice.DomainError, error)
	ValidateAccessToken(ctx context.Context, accessToken string) (string, []string, *userservice.DomainError, error)
}

// UserService implements users.v1.UserServiceServer.
type UserService struct {
	usersv1.UnimplementedUserServiceServer

	logger      zerolog.Logger
	authService AuthService
}

// NewUserService creates a new user service handler.
func NewUserService(logger zerolog.Logger, authService AuthService) *UserService {
	return &UserService{
		logger:      logger,
		authService: authService,
	}
}

func (s *UserService) Register(ctx context.Context, req *usersv1.RegisterRequest) (*usersv1.RegisterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	user, tokens, domainErr, err := s.authService.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetName())
	if domainErr != nil {
		return &usersv1.RegisterResponse{Error: toProtoError(domainErr)}, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("register failed")
		return nil, mapInfraError(err)
	}

	return &usersv1.RegisterResponse{
		User:   toProtoUser(user),
		Tokens: toProtoTokens(tokens),
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *usersv1.LoginRequest) (*usersv1.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	user, tokens, domainErr, err := s.authService.Login(ctx, req.GetEmail(), req.GetPassword())
	if domainErr != nil {
		return &usersv1.LoginResponse{Error: toProtoError(domainErr)}, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("login failed")
		return nil, mapInfraError(err)
	}

	return &usersv1.LoginResponse{
		User:   toProtoUser(user),
		Tokens: toProtoTokens(tokens),
	}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *usersv1.RefreshTokenRequest) (*usersv1.RefreshTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tokens, domainErr, err := s.authService.RefreshToken(ctx, req.GetRefreshToken())
	if domainErr != nil {
		return &usersv1.RefreshTokenResponse{Error: toProtoError(domainErr)}, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("refresh token failed")
		return nil, mapInfraError(err)
	}

	return &usersv1.RefreshTokenResponse{Tokens: toProtoTokens(tokens)}, nil
}

func (s *UserService) GetProfile(ctx context.Context, req *usersv1.GetProfileRequest) (*usersv1.GetProfileResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		userID = req.GetCtx().GetUserId()
	}

	user, domainErr, err := s.authService.GetProfile(ctx, userID)
	if domainErr != nil {
		return &usersv1.GetProfileResponse{Error: toProtoError(domainErr)}, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("get profile failed")
		return nil, mapInfraError(err)
	}

	return &usersv1.GetProfileResponse{User: toProtoUser(user)}, nil
}

func (s *UserService) ValidateAccessToken(ctx context.Context, req *usersv1.ValidateAccessTokenRequest) (*usersv1.ValidateAccessTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	userID, roles, domainErr, err := s.authService.ValidateAccessToken(ctx, req.GetAccessToken())
	if domainErr != nil {
		return &usersv1.ValidateAccessTokenResponse{Error: toProtoError(domainErr)}, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("validate access token failed")
		return nil, mapInfraError(err)
	}

	return &usersv1.ValidateAccessTokenResponse{
		UserId: userID,
		Roles:  append([]string(nil), roles...),
	}, nil
}

func mapInfraError(err error) error {
	if errors.Is(err, userservice.ErrUnavailable) {
		return status.Error(codes.Unavailable, "dependency unavailable")
	}
	return status.Error(codes.Internal, "internal error")
}

func toProtoError(domainErr *userservice.DomainError) *commonv1.Error {
	if domainErr == nil {
		return nil
	}

	return &commonv1.Error{
		Code:    domainErr.Code,
		Message: domainErr.Message,
	}
}

func toProtoUser(user userservice.User) *usersv1.User {
	return &usersv1.User{
		UserId:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: timestamppb.New(user.CreatedAt.UTC()),
	}
}

func toProtoTokens(tokens userservice.AuthTokens) *usersv1.AuthTokens {
	return &usersv1.AuthTokens{
		AccessToken:             tokens.AccessToken,
		RefreshToken:            tokens.RefreshToken,
		AccessExpiresInSeconds:  tokens.AccessExpiresInSeconds,
		RefreshExpiresInSeconds: tokens.RefreshExpiresInSeconds,
	}
}
