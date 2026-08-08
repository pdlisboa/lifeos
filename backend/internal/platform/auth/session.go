package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/platform/idgen"
)

var ErrInvalidSession = errors.New("sessão inválida ou expirada")

// SessionStore guarda só o hash do token (02-modelo-de-dados.md §3):
// um dump do banco nunca expõe uma sessão válida.
type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

func generateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *SessionStore) Create(ctx context.Context, userID, kind, userAgent string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	token, err = generateToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gerar token: %w", err)
	}
	id, err := idgen.NewUUIDv7()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gerar id: %w", err)
	}
	expiresAt = time.Now().Add(ttl)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO user_session (id, user_id, token_hash, kind, user_agent, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, hashToken(token), kind, userAgent, expiresAt,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gravar sessão: %w", err)
	}
	return token, expiresAt, nil
}

func (s *SessionStore) Validate(ctx context.Context, token string) (userID string, err error) {
	row := s.pool.QueryRow(ctx,
		`SELECT user_id FROM user_session
		 WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		hashToken(token),
	)
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInvalidSession
		}
		return "", fmt.Errorf("validar sessão: %w", err)
	}
	return userID, nil
}

func (s *SessionStore) Revoke(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE user_session SET revoked_at = now()
		 WHERE token_hash = $1 AND revoked_at IS NULL`,
		hashToken(token),
	)
	if err != nil {
		return fmt.Errorf("revogar sessão: %w", err)
	}
	return nil
}
