package model

// User represents an authenticated Notion user stored in SQLite.
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

// JWTClaims holds the JWT payload.
// SessionID (jti) ties the token to a row in the sessions table.
type JWTClaims struct {
	SessionID string `json:"jti"`
	UserID    string `json:"sub"`
	UserName  string `json:"user_name"`
}
