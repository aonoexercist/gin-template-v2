package users

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Service holds the feature's business rules.
type Service interface {
	Create(ctx context.Context, email, name string, password string) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	List(ctx context.Context, page, perPage int) ([]User, int64, error)
	Update(ctx context.Context, id uuid.UUID, email, name *string) (User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, email, name string, password string) (User, error) {
	email, name = normalizeEmail(email), strings.TrimSpace(name)
	if email == "" || name == "" {
		return User{}, ErrInvalidInput
	}
	return s.repo.Create(ctx, email, name, password)
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.Get(ctx, id)
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

	items, err := s.repo.List(ctx, int32(perPage), int32((page-1)*perPage))
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx)
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
	return s.repo.Update(ctx, id, email, name)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
