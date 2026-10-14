package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo is an in-memory Repository used by service and handler tests.
type fakeRepo struct {
	byID    map[uuid.UUID]User
	lastLim int32
	lastOff int32
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[uuid.UUID]User{}} }

func (f *fakeRepo) Create(_ context.Context, email, name string) (User, error) {
	for _, u := range f.byID {
		if u.Email == email {
			return User{}, ErrEmailTaken
		}
	}
	now := time.Now()
	u := User{ID: uuid.New(), Email: email, Name: name, CreatedAt: now, UpdatedAt: now}
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (User, error) {
	u, ok := f.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepo) List(_ context.Context, limit, offset int32) ([]User, error) {
	f.lastLim, f.lastOff = limit, offset
	out := make([]User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeRepo) Count(context.Context) (int64, error) { return int64(len(f.byID)), nil }

func (f *fakeRepo) Update(_ context.Context, id uuid.UUID, email, name *string) (User, error) {
	u, ok := f.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	if email != nil {
		u.Email = *email
	}
	if name != nil {
		u.Name = *name
	}
	f.byID[id] = u
	return u, nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func TestService_CreateNormalizesAndRejectsDuplicates(t *testing.T) {
	svc := NewService(newFakeRepo())
	ctx := context.Background()

	u, err := svc.Create(ctx, "  Jane@Example.COM ", "  Jane  ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if u.Email != "jane@example.com" || u.Name != "Jane" {
		t.Errorf("not normalized: %+v", u)
	}

	if _, err := svc.Create(ctx, "jane@example.com", "Other"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate err = %v, want ErrEmailTaken", err)
	}
	if _, err := svc.Create(ctx, "a@b.co", "   "); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("blank name err = %v, want ErrInvalidInput", err)
	}
}

func TestService_ListPaginationBounds(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	if _, _, err := svc.List(context.Background(), 3, 500); err != nil {
		t.Fatal(err)
	}
	if repo.lastLim != 100 || repo.lastOff != 200 {
		t.Errorf("limit/offset = %d/%d, want 100/200", repo.lastLim, repo.lastOff)
	}
	if _, _, err := svc.List(context.Background(), 0, 0); err != nil {
		t.Fatal(err)
	}
	if repo.lastLim != 20 || repo.lastOff != 0 {
		t.Errorf("defaults limit/offset = %d/%d, want 20/0", repo.lastLim, repo.lastOff)
	}
}

func TestService_Update(t *testing.T) {
	svc := NewService(newFakeRepo())
	ctx := context.Background()
	u, _ := svc.Create(ctx, "a@b.co", "A")

	if _, err := svc.Update(ctx, u.ID, nil, nil); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("empty update err = %v, want ErrInvalidInput", err)
	}
	name := " New "
	got, err := svc.Update(ctx, u.ID, nil, &name)
	if err != nil || got.Name != "New" || got.Email != "a@b.co" {
		t.Errorf("Update() = %+v, %v", got, err)
	}
	if _, err := svc.Update(ctx, uuid.New(), nil, &name); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing err = %v, want ErrNotFound", err)
	}
}
