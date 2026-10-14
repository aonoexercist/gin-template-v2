package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"gin-template-v2/internal/features/users/usersdb"
)

// Repository is the persistence boundary of the feature (mockable in tests).
type Repository interface {
	Create(ctx context.Context, email, name string) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	List(ctx context.Context, limit, offset int32) ([]User, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, id uuid.UUID, email, name *string) (User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgRepository struct{ q usersdb.Querier }

// NewRepository returns a Postgres-backed Repository. db is typically a *pgxpool.Pool.
func NewRepository(db usersdb.DBTX) Repository {
	return &pgRepository{q: usersdb.New(db)}
}

func (r *pgRepository) Create(ctx context.Context, email, name string) (User, error) {
	row, err := r.q.CreateUser(ctx, usersdb.CreateUserParams{Email: email, Name: name})
	if err != nil {
		return User{}, mapError(err)
	}
	return toUser(row), nil
}

func (r *pgRepository) Get(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.GetUser(ctx, id)
	if err != nil {
		return User{}, mapError(err)
	}
	return toUser(row), nil
}

func (r *pgRepository) List(ctx context.Context, limit, offset int32) ([]User, error) {
	rows, err := r.q.ListUsers(ctx, usersdb.ListUsersParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]User, len(rows))
	for i, row := range rows {
		out[i] = toUser(row)
	}
	return out, nil
}

func (r *pgRepository) Count(ctx context.Context) (int64, error) {
	n, err := r.q.CountUsers(ctx)
	return n, mapError(err)
}

func (r *pgRepository) Update(ctx context.Context, id uuid.UUID, email, name *string) (User, error) {
	row, err := r.q.UpdateUser(ctx, usersdb.UpdateUserParams{ID: id, Email: email, Name: name})
	if err != nil {
		return User{}, mapError(err)
	}
	return toUser(row), nil
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteUser(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func toUser(u usersdb.User) User {
	return User{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}

// mapError translates driver errors into domain errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrEmailTaken
	}
	return err
}
