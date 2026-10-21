package security

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"marketplace/internal/domain"
)

var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token is expired")
)

type Claims struct {
	Role      domain.Role `json:"role"`
	TokenType string      `json:"token_type"`
	jwt.RegisteredClaims
}

type Pair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewManager(secret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

func (m *Manager) NewPair(userID int64, role domain.Role) (Pair, error) {
	access, err := m.sign(userID, role, "access", m.accessTTL)
	if err != nil {
		return Pair{}, err
	}
	refresh, err := m.sign(userID, role, "refresh", m.refreshTTL)
	if err != nil {
		return Pair{}, err
	}

	return Pair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(m.accessTTL.Seconds()),
	}, nil
}

func (m *Manager) ParseAccess(raw string) (int64, domain.Role, error) {
	claims, err := m.parse(raw, "access")
	if err != nil {
		return 0, "", err
	}
	return parseSubject(claims)
}

func (m *Manager) ParseRefresh(raw string) (int64, domain.Role, error) {
	claims, err := m.parse(raw, "refresh")
	if err != nil {
		return 0, "", err
	}
	return parseSubject(claims)
}

func (m *Manager) sign(userID int64, role domain.Role, tokenType string, ttl time.Duration) (string, error) {
	now := m.now().UTC()
	claims := Claims{
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) parse(raw, expectedType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return m.secret, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid || claims.TokenType != expectedType {
		return nil, ErrTokenInvalid
	}
	if claims.Role != domain.RoleUser && claims.Role != domain.RoleSeller && claims.Role != domain.RoleAdmin {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func parseSubject(claims *Claims) (int64, domain.Role, error) {
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", ErrTokenInvalid
	}
	return id, claims.Role, nil
}
