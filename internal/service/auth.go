package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/security"
	"marketplace/internal/store"
)

type AuthService struct {
	queries *store.Queries
	tokens  *security.Manager
}

func NewAuthService(queries *store.Queries, tokens *security.Manager) *AuthService {
	return &AuthService{queries: queries, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password string, role domain.Role) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if role != domain.RoleUser && role != domain.RoleSeller {
		return domain.User{}, apperror.Validation(map[string]any{"role": "only USER or SELLER can be registered"})
	}
	if len([]byte(password)) > 72 {
		return domain.User{}, apperror.Validation(map[string]any{"password": "must be at most 72 bytes for bcrypt"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, apperror.Internal(err)
	}

	user, err := s.queries.CreateUser(ctx, email, string(hash), role)
	if store.IsUniqueViolation(err) {
		return domain.User{}, apperror.New(http.StatusConflict, "USER_ALREADY_EXISTS", "User with this email already exists", nil)
	}
	if err != nil {
		return domain.User{}, apperror.Internal(err)
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (security.Pair, error) {
	user, err := s.queries.UserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return security.Pair{}, invalidCredentials()
	}
	if err != nil {
		return security.Pair{}, apperror.Internal(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return security.Pair{}, invalidCredentials()
	}

	pair, err := s.tokens.NewPair(user.ID, user.Role)
	if err != nil {
		return security.Pair{}, apperror.Internal(err)
	}
	return pair, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (security.Pair, error) {
	userID, _, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return security.Pair{}, apperror.RefreshTokenInvalid()
	}

	user, err := s.queries.UserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return security.Pair{}, apperror.RefreshTokenInvalid()
	}
	if err != nil {
		return security.Pair{}, apperror.Internal(err)
	}

	pair, err := s.tokens.NewPair(user.ID, user.Role)
	if err != nil {
		return security.Pair{}, apperror.Internal(err)
	}
	return pair, nil
}

func invalidCredentials() *apperror.Error {
	return apperror.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect", nil)
}
