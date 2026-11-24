package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

func TestUserRepo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := Connect(ctx, testDatabaseURL(t))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	repo := NewUserRepo(pool)

	email := "repo-test-" + time.Now().Format("150405.000000") + "@mail.com"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	created, err := repo.Create(ctx, domain.User{Email: email, Name: "Repo", PasswordHash: "hash", Role: domain.RoleUser})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Errorf("Create must return id and created_at: %+v", created)
	}

	t.Run("duplicate email, any case", func(t *testing.T) {
		_, err := repo.Create(ctx, domain.User{Email: "REPO-" + email[5:], Name: "Dup", PasswordHash: "h", Role: domain.RoleUser})
		if !errors.Is(err, domain.ErrEmailTaken) {
			t.Fatalf("err = %v, want ErrEmailTaken", err)
		}
	})
	t.Run("find by email is case-insensitive", func(t *testing.T) {
		u, err := repo.FindByEmail(ctx, "REPO-"+email[5:])
		if err != nil || u.ID != created.ID || u.PasswordHash != "hash" {
			t.Fatalf("FindByEmail = %+v, %v", u, err)
		}
	})
	t.Run("find by id", func(t *testing.T) {
		u, err := repo.FindByID(ctx, created.ID)
		if err != nil || u.Email != email {
			t.Fatalf("FindByID = %+v, %v", u, err)
		}
	})
	t.Run("not found", func(t *testing.T) {
		if _, err := repo.FindByID(ctx, -1); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("FindByID err = %v, want ErrNotFound", err)
		}
		if _, err := repo.FindByEmail(ctx, "missing@nowhere.test"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("FindByEmail err = %v, want ErrNotFound", err)
		}
	})
}
