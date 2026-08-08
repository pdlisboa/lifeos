package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("usuário não encontrado")

type User struct {
	ID             string
	Email          string
	PasswordHash   string
	Timezone       string
	QuietHoursFrom *int32
	QuietHoursTo   *int32
}

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) ByEmail(ctx context.Context, email string) (User, error) {
	return s.scanOne(ctx, `SELECT id, email, password_hash, timezone, quiet_hours FROM app_user WHERE email = $1`, email)
}

func (s *UserStore) ByID(ctx context.Context, id string) (User, error) {
	return s.scanOne(ctx, `SELECT id, email, password_hash, timezone, quiet_hours FROM app_user WHERE id = $1`, id)
}

func (s *UserStore) scanOne(ctx context.Context, query, arg string) (User, error) {
	var u User
	var quietHours pgtype.Range[pgtype.Int4]

	row := s.pool.QueryRow(ctx, query, arg)
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Timezone, &quietHours); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("buscar usuário: %w", err)
	}

	// quiet_hours é null = sem janela de silêncio configurada (comum na Fatia 0).
	if quietHours.Valid {
		if quietHours.LowerType == pgtype.Inclusive {
			from := quietHours.Lower.Int32
			u.QuietHoursFrom = &from
		}
		if quietHours.UpperType == pgtype.Inclusive || quietHours.UpperType == pgtype.Exclusive {
			to := quietHours.Upper.Int32
			u.QuietHoursTo = &to
		}
	}

	return u, nil
}

func (s *UserStore) Create(ctx context.Context, id, email, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_user (id, email, password_hash) VALUES ($1, $2, $3)`,
		id, email, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("criar usuário: %w", err)
	}
	return nil
}
