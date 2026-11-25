package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

func seedAdmin(d testDeps) {
	d.admin.auditoriums = map[int64]domain.Auditorium{
		1: {ID: 1, Name: "Sala 1", Rows: 8, Cols: 10},
		2: {ID: 2, Name: "Sala 2", Rows: 6, Cols: 8},
	}
}

func issueRoleToken(t *testing.T, d testDeps, userID int64, role domain.Role) string {
	t.Helper()
	tok, err := d.tokens.Issue(domain.User{ID: userID, Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestAdmin_RequiresAuthAndAdminRole(t *testing.T) {
	h, d := newTestRouter(t)
	seedAdmin(d)
	userTok := issueRoleToken(t, d, 2, domain.RoleUser)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	routes := []struct {
		method, path, body string
	}{
		{"GET", "/api/admin/auditoriums", ""},
		{"POST", "/api/admin/movies", `{"title":"Dune","durationMin":155}`},
		{"PUT", "/api/admin/movies/1", `{"title":"Dune","durationMin":155}`},
		{"DELETE", "/api/admin/movies/1", ""},
		{"POST", "/api/admin/showtimes", fmt.Sprintf(`{"movieId":1,"auditoriumId":1,"startsAt":%q,"priceCents":8000}`, future)},
		{"PUT", "/api/admin/showtimes/1", fmt.Sprintf(`{"movieId":1,"auditoriumId":1,"startsAt":%q,"priceCents":8000}`, future)},
		{"DELETE", "/api/admin/showtimes/1", ""},
	}
	for _, r := range routes {
		t.Run("unauthenticated "+r.method+" "+r.path, func(t *testing.T) {
			rec := request(t, h, r.method, r.path, r.body, nil)
			if rec.Code != http.StatusUnauthorized || decodeError(t, rec).Error.Code != "UNAUTHORIZED" {
				t.Fatalf("got %d %s, want 401 UNAUTHORIZED", rec.Code, rec.Body)
			}
		})
		t.Run("non-admin user "+r.method+" "+r.path, func(t *testing.T) {
			rec := request(t, h, r.method, r.path, r.body, bearer(userTok))
			if rec.Code != http.StatusForbidden || decodeError(t, rec).Error.Code != "FORBIDDEN" {
				t.Fatalf("got %d %s, want 403 FORBIDDEN", rec.Code, rec.Body)
			}
		})
	}
}

func TestAdmin_MoviesAndShowtimesLifecycle(t *testing.T) {
	h, d := newTestRouter(t)
	seedAdmin(d)
	adminTok := issueRoleToken(t, d, 1, domain.RoleAdmin)

	// List auditoriums.
	audRec := request(t, h, "GET", "/api/admin/auditoriums", "", bearer(adminTok))
	if audRec.Code != http.StatusOK || !strings.Contains(audRec.Body.String(), `"Sala 1"`) {
		t.Fatalf("GET auditoriums: %d %s", audRec.Code, audRec.Body)
	}

	// Create movie.
	createMov := request(t, h, "POST", "/api/admin/movies",
		`{"title":"Arrival","synopsis":"Time.","durationMin":116,"rating":"B","posterUrl":"https://cdn.example/a.jpg"}`,
		bearer(adminTok))
	if createMov.Code != http.StatusCreated {
		t.Fatalf("POST movie: %d %s", createMov.Code, createMov.Body)
	}
	var mv struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal(createMov.Body.Bytes(), &mv)
	if mv.ID == 0 || mv.Title != "Arrival" {
		t.Fatalf("unexpected movie: %s", createMov.Body)
	}

	// Update movie.
	updMov := request(t, h, "PUT", fmt.Sprintf("/api/admin/movies/%d", mv.ID),
		`{"title":"Arrival (Director Cut)","synopsis":"More time.","durationMin":120,"rating":"B","posterUrl":""}`,
		bearer(adminTok))
	if updMov.Code != http.StatusOK || !strings.Contains(updMov.Body.String(), `"durationMin":120`) {
		t.Fatalf("PUT movie: %d %s", updMov.Code, updMov.Body)
	}

	// Create showtime.
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	createSt := request(t, h, "POST", "/api/admin/showtimes",
		fmt.Sprintf(`{"movieId":%d,"auditoriumId":1,"startsAt":%q,"priceCents":8500}`, mv.ID, future),
		bearer(adminTok))
	if createSt.Code != http.StatusCreated {
		t.Fatalf("POST showtime: %d %s", createSt.Code, createSt.Body)
	}
	var st struct {
		ID           int64  `json:"id"`
		AuditoriumID int64  `json:"auditoriumId"`
		Auditorium   string `json:"auditorium"`
	}
	_ = json.Unmarshal(createSt.Body.Bytes(), &st)
	if st.ID == 0 || st.AuditoriumID != 1 || st.Auditorium != "Sala 1" {
		t.Fatalf("unexpected showtime: %s", createSt.Body)
	}

	// Active reservation blocks delete of both showtime and movie -> 409 HAS_ACTIVE_RESERVATIONS.
	d.admin.activeShowtime[st.ID] = true
	delBusySt := request(t, h, "DELETE", fmt.Sprintf("/api/admin/showtimes/%d", st.ID), "", bearer(adminTok))
	if delBusySt.Code != http.StatusConflict || decodeError(t, delBusySt).Error.Code != "HAS_ACTIVE_RESERVATIONS" {
		t.Fatalf("DELETE busy showtime: got %d %s, want 409 HAS_ACTIVE_RESERVATIONS", delBusySt.Code, delBusySt.Body)
	}
	delBusyMov := request(t, h, "DELETE", fmt.Sprintf("/api/admin/movies/%d", mv.ID), "", bearer(adminTok))
	if delBusyMov.Code != http.StatusConflict || decodeError(t, delBusyMov).Error.Code != "HAS_ACTIVE_RESERVATIONS" {
		t.Fatalf("DELETE busy movie: got %d %s, want 409 HAS_ACTIVE_RESERVATIONS", delBusyMov.Code, delBusyMov.Body)
	}

	// Changing auditorium while active reservations exist -> 409 HAS_ACTIVE_RESERVATIONS.
	moveBusy := request(t, h, "PUT", fmt.Sprintf("/api/admin/showtimes/%d", st.ID),
		fmt.Sprintf(`{"movieId":%d,"auditoriumId":2,"startsAt":%q,"priceCents":9000}`, mv.ID, future),
		bearer(adminTok))
	if moveBusy.Code != http.StatusConflict || decodeError(t, moveBusy).Error.Code != "HAS_ACTIVE_RESERVATIONS" {
		t.Fatalf("PUT busy showtime auditorium: got %d %s, want 409 HAS_ACTIVE_RESERVATIONS", moveBusy.Code, moveBusy.Body)
	}

	// Once reservations are no longer active, update & deletes succeed.
	d.admin.activeShowtime[st.ID] = false
	updSt := request(t, h, "PUT", fmt.Sprintf("/api/admin/showtimes/%d", st.ID),
		fmt.Sprintf(`{"movieId":%d,"auditoriumId":2,"startsAt":%q,"priceCents":9000}`, mv.ID, future),
		bearer(adminTok))
	if updSt.Code != http.StatusOK || !strings.Contains(updSt.Body.String(), `"Sala 2"`) {
		t.Fatalf("PUT showtime: %d %s", updSt.Code, updSt.Body)
	}

	delSt := request(t, h, "DELETE", fmt.Sprintf("/api/admin/showtimes/%d", st.ID), "", bearer(adminTok))
	if delSt.Code != http.StatusNoContent || delSt.Body.Len() != 0 {
		t.Fatalf("DELETE showtime: %d %q", delSt.Code, delSt.Body.String())
	}
	delMov := request(t, h, "DELETE", fmt.Sprintf("/api/admin/movies/%d", mv.ID), "", bearer(adminTok))
	if delMov.Code != http.StatusNoContent || delMov.Body.Len() != 0 {
		t.Fatalf("DELETE movie: %d %q", delMov.Code, delMov.Body.String())
	}
}

func TestAdmin_ValidationAndErrorHygiene(t *testing.T) {
	h, d := newTestRouter(t)
	seedAdmin(d)
	adminTok := issueRoleToken(t, d, 1, domain.RoleAdmin)

	// Bad path IDs -> 400 VALIDATION_ERROR.
	for _, path := range []string{"/api/admin/movies/0", "/api/admin/movies/abc", "/api/admin/showtimes/-1"} {
		rec := request(t, h, "DELETE", path, "", bearer(adminTok))
		if rec.Code != http.StatusBadRequest || decodeError(t, rec).Error.Code != "VALIDATION_ERROR" {
			t.Errorf("DELETE %s: got %d %s", path, rec.Code, rec.Body)
		}
	}

	// Unknown JSON field -> 400 VALIDATION_ERROR.
	unknown := request(t, h, "POST", "/api/admin/movies", `{"title":"X","durationMin":90,"extra":true}`, bearer(adminTok))
	if unknown.Code != http.StatusBadRequest || decodeError(t, unknown).Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("unknown field: %d %s", unknown.Code, unknown.Body)
	}

	// Non-JSON Content-Type -> 415 UNSUPPORTED_MEDIA_TYPE.
	badCT := request(t, h, "POST", "/api/admin/movies", `{"title":"X","durationMin":90}`, map[string]string{
		"Authorization": "Bearer " + adminTok,
		"Content-Type":  "text/plain",
	})
	if badCT.Code != http.StatusUnsupportedMediaType || decodeError(t, badCT).Error.Code != "UNSUPPORTED_MEDIA_TYPE" {
		t.Fatalf("bad content-type: %d %s", badCT.Code, badCT.Body)
	}

	// Missing resource -> 404 NOT_FOUND.
	missing := request(t, h, "DELETE", "/api/admin/movies/999", "", bearer(adminTok))
	if missing.Code != http.StatusNotFound || decodeError(t, missing).Error.Code != "NOT_FOUND" {
		t.Fatalf("missing movie: %d %s", missing.Code, missing.Body)
	}
}
