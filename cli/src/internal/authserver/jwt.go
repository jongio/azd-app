package authserver

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims represents the JWT claims for an Azure access token.
type TokenClaims struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token creation and validation.
type JWTManager struct {
	secret []byte
}

// NewJWTManager creates a new JWT manager with the given secret.
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

// CreateToken creates a JWT token wrapping an Azure access token.
func (j *JWTManager) CreateToken(azureToken, scope string, expiresIn time.Duration) (string, error) {
	if azureToken == "" {
		return "", fmt.Errorf("azure token cannot be empty")
	}

	now := time.Now()
	claims := TokenClaims{
		AccessToken: azureToken,
		Scope:       scope,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "azd-auth-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	signedToken, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (j *JWTManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
