package user

import (
	"context"
	"errors"
	"strings"

	"gin-template-v2/internal/platform/dbgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Create(ctx context.Context, email, name string, password string) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	List(ctx context.Context, page, perPage int) ([]User, int64, error)
	Update(ctx context.Context, id uuid.UUID, email, name *string) (User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// service talks to sqlc's generated Querier directly. Querier is already an
// interface, so it can be mocked in tests; a separate Repository layer would
// only forward calls. Use q.WithTx(tx) when you need a transaction.
type service struct{ q dbgen.Querier }

func NewService(q dbgen.Querier) Service { return &service{q: q} }

func (s *service) Create(ctx context.Context, email, name string, password string) (User, error) {
	email, name = normalizeEmail(email), strings.TrimSpace(name)
	if email == "" || name == "" {
		return User{}, ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	hashStr := string(hash)

	user, err := s.q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: &hashStr,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return user, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	user, err := s.q.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (s *service) List(ctx context.Context, page, perPage int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	items, err := s.q.ListUsers(ctx, dbgen.ListUsersParams{
		PageLimit:  int32(perPage),
		PageOffset: int32((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, email, name *string) (User, error) {
	if email == nil && name == nil {
		return User{}, ErrInvalidInput
	}
	if email != nil {
		e := normalizeEmail(*email)
		if e == "" {
			return User{}, ErrInvalidInput
		}
		email = &e
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return User{}, ErrInvalidInput
		}
		name = &n
	}

	user, err := s.q.UpdateUser(ctx, dbgen.UpdateUserParams{
		ID:    id,
		Email: email,
		Name:  name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return user, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Candidate to move into a shared platform/dbx package once a second feature needs it.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }
