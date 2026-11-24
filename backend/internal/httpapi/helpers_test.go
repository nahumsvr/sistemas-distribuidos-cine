package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"proyecto-cine/backend/internal/auth"
	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

const testOrigin = "http://localhost:3000"

var testSecret = []byte("0123456789abcdef0123456789abcdef")

type testDeps struct {
	tokens *auth.Tokens
	users  *memUsers
}

// newTestRouterFull wires the real router, service and token code with an
// in-memory user store, so tests exercise the public HTTP interface.
func newTestRouterFull(t *testing.T, db Pinger, rl RateLimit) (http.Handler, testDeps) {
	t.Helper()
	users := &memUsers{byID: map[int64]domain.User{}}
	tokens := auth.NewTokens(testSecret, time.Hour)
	h := NewRouter(Deps{
		DB:            db,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth:          service.NewAuthService(users, tokens, bcrypt.MinCost),
		Tokens:        tokens,
		AllowedOrigin: testOrigin,
		AuthRateLimit: rl,
	})
	return h, testDeps{tokens: tokens, users: users}
}

// generous keeps the limiter out of the way in tests that are not about it.
var generous = RateLimit{Every: time.Millisecond, Burst: 1000}

func newTestRouter(t *testing.T) (http.Handler, testDeps) {
	return newTestRouterFull(t, fakePinger{}, generous)
}

func newTestRouterWith(t *testing.T, rl RateLimit) (http.Handler, testDeps) {
	return newTestRouterFull(t, fakePinger{}, rl)
}

func newTestRouterDB(t *testing.T, p Pinger) http.Handler {
	h, _ := newTestRouterFull(t, p, generous)
	return h
}

type memUsers struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]domain.User
}

func (m *memUsers) Create(_ context.Context, u domain.User) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.byID {
		if strings.EqualFold(e.Email, u.Email) {
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
