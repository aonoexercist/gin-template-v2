package organization

import (
	"context"
	"errors"
	"strings"

	"gin-template-v2/internal/platform/dbgen"

	"github.com/jackc/pgx/v5/pgconn"
)

type Service interface {
	CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (Organization, error)
}

// service talks to sqlc's generated Querier directly. Querier is already an
// interface, so it can be mocked in tests; a separate Repository layer would
// only forward calls. Use q.WithTx(tx) when you need a transaction.
type service struct{ q dbgen.Querier }

func NewService(q dbgen.Querier) Service { return &service{q: q} }

func (s *service) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (Organization, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Organization{}, ErrInvalidInput
	}

	org, err := s.q.CreateOrganization(ctx, name)
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, ErrOrganizationTaken
		}
		return Organization{}, err
	}
	return org, nil
}

// Candidate to move into a shared platform/dbx package once a second feature needs it.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
