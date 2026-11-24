package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"proyecto-cine/backend/internal/domain"
)

// memUsers is an in-memory UserRepository for tests.
type memUsers struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]domain.User
}

func newMemUsers() *memUsers { return &memUsers{byID: map[int64]domain.User{}} }

func (m *memUsers) Create(_ context.Context, u domain.User) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.byID {
		if strings.EqualFold(existing.Email, u.Email) {
			return domain.User{}, domain.ErrEmailTaken
		}
	}
	m.nextID++
	u.ID = m.nextID
	m.byID[u.ID] = u
	return u, nil
}

func (m *memUsers) FindByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *memUsers) FindByID(_ context.Context, id int64) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrNotFound
}

type fakeTokens struct{}

func (fakeTokens) Issue(u domain.User) (string, error) { return "token-for-" + u.Email, nil }

func newService() (*AuthService, *memUsers) {
	users := newMemUsers()
	return NewAuthService(users, fakeTokens{}, bcrypt.MinCost), users
}

var ctx = context.Background()

func validInput() RegisterInput {
	return RegisterInput{Name: "Ana", Email: "ana@mail.com", Password: "secret123"}
}

func TestRegister_CreatesUserWithHashedPasswordAndToken(t *testing.T) {
	svc, users := newService()

	res, err := svc.Register(ctx, RegisterInput{Name: "  Ana  ", Email: "  Ana@Mail.COM ", Password: "secret123"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if res.Token == "" {
		t.Error("expected a token")
	}
	if res.User.Role != domain.RoleUser {
		t.Errorf("role = %q, want user (registration never grants admin)", res.User.Role)
	}
	if res.User.Email != "ana@mail.com" || res.User.Name != "Ana" {
		t.Errorf("not normalized: email %q name %q", res.User.Email, res.User.Name)
	}
	stored, _ := users.FindByID(ctx, res.User.ID)
	if stored.PasswordHash == "secret123" || bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("secret123")) != nil {
		t.Error("password must be stored as a valid bcrypt hash")
	}
}

func TestRegister_RejectsInvalidInput(t *testing.T) {
	cases := map[string]func(*RegisterInput){
		"invalid email":       func(in *RegisterInput) { in.Email = "not-an-email" },
		"email with name":     func(in *RegisterInput) { in.Email = "Ana <ana@mail.com>" },
		"password too short":  func(in *RegisterInput) { in.Password = "1234567" },
		"password over 72 B":  func(in *RegisterInput) { in.Password = strings.Repeat("a", 73) },
		"empty name":          func(in *RegisterInput) { in.Name = "   " },
		"name over 100 chars": func(in *RegisterInput) { in.Name = strings.Repeat("n", 101) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			svc, _ := newService()
			in := validInput()
			mutate(&in)
			_, err := svc.Register(ctx, in)
			var ve *domain.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
		})
	}
}

func TestRegister_DuplicateEmailIsCaseInsensitive(t *testing.T) {
	svc, _ := newService()
	if _, err := svc.Register(ctx, validInput()); err != nil {
		t.Fatal(err)
	}
	in := validInput()
	in.Email = "ANA@mail.com"
	if _, err := svc.Register(ctx, in); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestLogin(t *testing.T) {
	svc, _ := newService()
	if _, err := svc.Register(ctx, validInput()); err != nil {
		t.Fatal(err)
	}

	t.Run("valid credentials", func(t *testing.T) {
		res, err := svc.Login(ctx, " ANA@mail.com ", "secret123")
		if err != nil || res.Token == "" || res.User.Email != "ana@mail.com" {
			t.Fatalf("Login = %+v, %v", res, err)
		}
	})
	t.Run("wrong password", func(t *testing.T) {
		if _, err := svc.Login(ctx, "ana@mail.com", "wrong-pass"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})
	t.Run("unknown email gives the same error", func(t *testing.T) {
		if _, err := svc.Login(ctx, "nobody@mail.com", "secret123"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestMe(t *testing.T) {
	svc, _ := newService()
	res, _ := svc.Register(ctx, validInput())

	u, err := svc.Me(ctx, res.User.ID)
	if err != nil || u.Email != "ana@mail.com" {
		t.Fatalf("Me = %+v, %v", u, err)
	}
	if _, err := svc.Me(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
