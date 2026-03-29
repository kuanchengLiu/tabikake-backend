package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/store"
)

// AuthService handles Notion OAuth, session management, and JWT issuance.
type AuthService struct {
	db              *store.DB
	clientID        string
	clientSecret    string
	redirectURI     string
	jwtSecret       []byte
	tokenEncryptKey []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(db *store.DB, clientID, clientSecret, redirectURI, jwtSecret string, tokenEncryptKey []byte) *AuthService {
	return &AuthService{
		db:              db,
		clientID:        clientID,
		clientSecret:    clientSecret,
		redirectURI:     redirectURI,
		jwtSecret:       []byte(jwtSecret),
		tokenEncryptKey: tokenEncryptKey,
	}
}

// OAuthURL returns the Notion OAuth authorization URL.
func (s *AuthService) OAuthURL() string {
	return "https://api.notion.com/v1/oauth/authorize?client_id=" +
		url.QueryEscape(s.clientID) +
		"&response_type=code&owner=user&redirect_uri=" +
		url.QueryEscape(s.redirectURI)
}

// HandleCallback exchanges the OAuth code for a session JWT.
// Returns the signed JWT string and the user record.
func (s *AuthService) HandleCallback(ctx context.Context, code string) (string, *model.User, error) {
	notionToken, notionUser, err := s.exchangeCode(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("exchange code: %w", err)
	}

	user := model.User{
		ID:        notionUser.id,
		Name:      notionUser.name,
		AvatarURL: notionUser.avatarURL,
	}
	if err := s.db.UpsertUser(ctx, user); err != nil {
		return "", nil, fmt.Errorf("upsert user: %w", err)
	}

	encToken, err := s.encrypt(notionToken)
	if err != nil {
		return "", nil, fmt.Errorf("encrypt token: %w", err)
	}

	sess := store.Session{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		NotionToken: encToken,
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
	}
	if err := s.db.InsertSession(ctx, sess); err != nil {
		return "", nil, fmt.Errorf("insert session: %w", err)
	}

	tokenStr, err := s.signJWT(sess.ID, user.ID, user.Name)
	if err != nil {
		return "", nil, fmt.Errorf("sign jwt: %w", err)
	}

	return tokenStr, &user, nil
}

// ValidateSession parses the JWT and verifies the session still exists and is not expired.
func (s *AuthService) ValidateSession(ctx context.Context, tokenStr string) (*model.JWTClaims, error) {
	claims, err := s.parseJWT(tokenStr)
	if err != nil {
		return nil, err
	}
	sess, err := s.db.GetSession(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = s.db.DeleteSession(ctx, claims.SessionID)
		return nil, fmt.Errorf("session expired")
	}
	return claims, nil
}

// Logout deletes the session and invalidates the JWT.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.db.DeleteSession(ctx, sessionID)
}

// GetUser returns the full user record from the database.
func (s *AuthService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.db.GetUser(ctx, userID)
}

// --- internals ---

type notionUserInfo struct {
	id        string
	name      string
	avatarURL string
}

func (s *AuthService) exchangeCode(ctx context.Context, code string) (string, notionUserInfo, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": s.redirectURI,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.notion.com/v1/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", notionUserInfo{}, fmt.Errorf("notion token request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", notionUserInfo{}, fmt.Errorf("notion token %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var res struct {
		AccessToken string `json:"access_token"`
		Owner       struct {
			User struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				AvatarURL string `json:"avatar_url"`
			} `json:"user"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", notionUserInfo{}, fmt.Errorf("decode token response: %w", err)
	}

	u := notionUserInfo{
		id:        res.Owner.User.ID,
		name:      res.Owner.User.Name,
		avatarURL: res.Owner.User.AvatarURL,
	}
	return res.AccessToken, u, nil
}

func (s *AuthService) signJWT(sessionID, userID, userName string) (string, error) {
	claims := jwtAdapter{JWTClaims: &model.JWTClaims{
		SessionID: sessionID,
		UserID:    userID,
		UserName:  userName,
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) parseJWT(tokenStr string) (*model.JWTClaims, error) {
	claims := &model.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &jwtAdapter{JWTClaims: claims}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired token")
	}
	return claims, nil
}

func (s *AuthService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.tokenEncryptKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// jwtAdapter bridges model.JWTClaims with jwt.Claims.
type jwtAdapter struct{ *model.JWTClaims }

func (j jwtAdapter) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil }
func (j jwtAdapter) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil }
func (j jwtAdapter) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil }
func (j jwtAdapter) GetIssuer() (string, error)                   { return "", nil }
func (j jwtAdapter) GetSubject() (string, error)                  { return j.UserID, nil }
func (j jwtAdapter) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }
