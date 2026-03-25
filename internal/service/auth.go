package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/yourname/tabikake/internal/model"
)

const notionTokenURL = "https://api.notion.com/v1/oauth/token"
const notionUserURL = "https://api.notion.com/v1/users/me"

// AuthService handles Notion OAuth and JWT issuance.
type AuthService struct {
	clientID     string
	clientSecret string
	redirectURI  string
	jwtSecret    string
}

// NewAuthService creates a new AuthService.
func NewAuthService(clientID, clientSecret, redirectURI, jwtSecret string) *AuthService {
	return &AuthService{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		jwtSecret:    jwtSecret,
	}
}

// ExchangeCode exchanges a Notion OAuth code for user info and returns a JWT.
func (s *AuthService) ExchangeCode(ctx context.Context, code string) (*model.AuthResponse, error) {
	accessToken, err := s.exchangeToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	user, err := s.fetchNotionUser(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch user failed: %w", err)
	}

	tokenStr, err := s.issueJWT(user)
	if err != nil {
		return nil, fmt.Errorf("issue jwt failed: %w", err)
	}

	return &model.AuthResponse{
		Token: tokenStr,
		User:  *user,
	}, nil
}

// ValidateToken parses and validates a JWT, returning the claims.
func (s *AuthService) ValidateToken(tokenStr string) (*model.JWTClaims, error) {
	claims := &jwtMapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return &model.JWTClaims{
		UserID:   claims.UserID,
		UserName: claims.UserName,
	}, nil
}

func (s *AuthService) exchangeToken(ctx context.Context, code string) (string, error) {
	credentials := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))

	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("redirect_uri", s.redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notionTokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("notion token endpoint returned %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}
	return result.AccessToken, nil
}

func (s *AuthService) fetchNotionUser(ctx context.Context, accessToken string) (*model.NotionUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, notionUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion users/me returned %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Person    struct {
			Email string `json:"email"`
		} `json:"person"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return &model.NotionUser{
		ID:        result.ID,
		Name:      result.Name,
		AvatarURL: result.AvatarURL,
		Email:     result.Person.Email,
	}, nil
}

func (s *AuthService) issueJWT(user *model.NotionUser) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"user_name": user.Name,
		"exp":       time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

type jwtMapClaims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	jwt.RegisteredClaims
}
