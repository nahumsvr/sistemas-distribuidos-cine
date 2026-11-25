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

func seedReservations(d testDeps) {
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	past := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	d.reservations.showtimes = map[int64]domain.ReservationShowtime{
		7: {ID: 7, AuditoriumID: 1, MovieTitle: "Dune", StartsAt: future, PriceCents: 8000},
		8: {ID: 8, AuditoriumID: 1, MovieTitle: "Dune", StartsAt: past, PriceCents: 8000},
	}
	d.reservations.auditoriums = map[int64][]domain.ReservedSeat{
		1: {
			{ID: 1, Row: "A", Number: 1},
			{ID: 2, Row: "A", Number: 2},
			{ID: 3, Row: "A", Number: 3},
			{ID: 4, Row: "A", Number: 4},
			{ID: 5, Row: "A", Number: 5},
		},
	}
}

func issueTokenFor(t *testing.T, d testDeps, userID int64) string {
	t.Helper()
	tok, err := d.tokens.Issue(domain.User{ID: userID, Role: domain.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestReservations_RequireAuth(t *testing.T) {
	h, d := newTestRouter(t)
	seedReservations(d)

	routes := []struct {
		method, path, body string
	}{
		{"GET", "/api/reservations", ""},
		{"GET", "/api/reservations/1", ""},
		{"POST", "/api/reservations", `{"showtimeId":7,"seatIds":[1]}`},
		{"PUT", "/api/reservations/1", `{"seatIds":[2]}`},
		{"DELETE", "/api/reservations/1", ""},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			rec := request(t, h, r.method, r.path, r.body, nil)
			if rec.Code != http.StatusUnauthorized || decodeError(t, rec).Error.Code != "UNAUTHORIZED" {
				t.Fatalf("got %d %s, want 401 UNAUTHORIZED", rec.Code, rec.Body)
			}
			if rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestReservations_LifecycleAndJSONShape(t *testing.T) {
	h, d := newTestRouter(t)
	seedReservations(d)
	tok1 := issueTokenFor(t, d, 1)
	tok2 := issueTokenFor(t, d, 2)

	// Empty list is [] never null.
	empty := request(t, h, "GET", "/api/reservations", "", bearer(tok1))
	if empty.Code != http.StatusOK || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", empty.Code, empty.Body)
	}

	// Create reservation.
	created := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[3,1]}`, bearer(tok1))
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body %s", created.Code, created.Body)
	}
	var got struct {
		ID         int64            `json:"id"`
		Status     string           `json:"status"`
		Showtime   map[string]any   `json:"showtime"`
		Seats      []map[string]any `json:"seats"`
		TotalCents int              `json:"totalCents"`
		CreatedAt  string           `json:"createdAt"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 || got.Status != "active" || got.TotalCents != 16000 {
		t.Fatalf("unexpected body: %s", created.Body)
	}
	if got.Showtime["id"] != float64(7) || got.Showtime["movieTitle"] != "Dune" {
		t.Errorf("unexpected showtime: %v", got.Showtime)
	}
	if _, err := time.Parse(time.RFC3339, got.Showtime["startsAt"].(string)); err != nil {
		t.Errorf("showtime.startsAt not RFC3339: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, got.CreatedAt); err != nil {
		t.Errorf("createdAt not RFC3339: %v", err)
	}
	if len(got.Seats) != 2 || got.Seats[0]["row"] != "A" || got.Seats[0]["number"] != float64(1) || got.Seats[1]["number"] != float64(3) {
		t.Errorf("unexpected seats: %v", got.Seats)
	}

	resPath := fmt.Sprintf("/api/reservations/%d", got.ID)

	// List & Get for owner.
	list := request(t, h, "GET", "/api/reservations", "", bearer(tok1))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"movieTitle":"Dune"`) {
		t.Fatalf("GET list: %d %s", list.Code, list.Body)
	}
	detail := request(t, h, "GET", resPath, "", bearer(tok1))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"totalCents":16000`) {
		t.Fatalf("GET detail: %d %s", detail.Code, detail.Body)
	}

	// Ownership isolation: user 2 cannot view, modify or cancel user 1's reservation.
	for _, tc := range []struct{ method, body string }{
		{"GET", ""},
		{"PUT", `{"seatIds":[4]}`},
		{"DELETE", ""},
	} {
		rec := request(t, h, tc.method, resPath, tc.body, bearer(tok2))
		if rec.Code != http.StatusNotFound || decodeError(t, rec).Error.Code != "NOT_FOUND" {
			t.Errorf("other user %s: got %d %s, want 404 NOT_FOUND", tc.method, rec.Code, rec.Body)
		}
	}
	if rec := request(t, h, "GET", "/api/reservations", "", bearer(tok2)); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("other user list must be empty, got %s", rec.Body)
	}

	// Modify seats.
	upd := request(t, h, "PUT", resPath, `{"seatIds":[4,5]}`, bearer(tok1))
	if upd.Code != http.StatusOK || !strings.Contains(upd.Body.String(), `"number":5`) {
		t.Fatalf("PUT: %d %s", upd.Code, upd.Body)
	}

	// Cancel returns 204 with no body, preserves cancelled record with seats: [], and frees seats.
	del := request(t, h, "DELETE", resPath, "", bearer(tok1))
	if del.Code != http.StatusNoContent || del.Body.Len() != 0 {
		t.Fatalf("DELETE: status %d body %q, want 204 empty", del.Code, del.Body.String())
	}
	afterCancel := request(t, h, "GET", resPath, "", bearer(tok1))
	if afterCancel.Code != http.StatusOK ||
		!strings.Contains(afterCancel.Body.String(), `"status":"cancelled"`) ||
		!strings.Contains(afterCancel.Body.String(), `"seats":[]`) ||
		!strings.Contains(afterCancel.Body.String(), `"totalCents":0`) {
		t.Fatalf("GET after cancel: %d %s", afterCancel.Code, afterCancel.Body)
	}
	// User 2 can now book seats 4 and 5.
	rebook := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[4,5]}`, bearer(tok2))
	if rebook.Code != http.StatusCreated {
		t.Fatalf("rebook after cancel: %d %s", rebook.Code, rebook.Body)
	}
}

func TestReservations_ErrorMappingAndValidation(t *testing.T) {
	h, d := newTestRouter(t)
	seedReservations(d)
	tok1 := issueTokenFor(t, d, 1)
	tok2 := issueTokenFor(t, d, 2)

	// Seat already taken -> 409 SEAT_TAKEN.
	first := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[1,2]}`, bearer(tok1))
	if first.Code != http.StatusCreated {
		t.Fatalf("setup create: %d %s", first.Code, first.Body)
	}
	dup := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[2,3]}`, bearer(tok2))
	if dup.Code != http.StatusConflict || decodeError(t, dup).Error.Code != "SEAT_TAKEN" {
		t.Fatalf("double booking: got %d %s, want 409 SEAT_TAKEN", dup.Code, dup.Body)
	}

	// Past showtime -> 422 SHOWTIME_STARTED.
	started := request(t, h, "POST", "/api/reservations", `{"showtimeId":8,"seatIds":[1]}`, bearer(tok1))
	if started.Code != http.StatusUnprocessableEntity || decodeError(t, started).Error.Code != "SHOWTIME_STARTED" {
		t.Fatalf("started showtime: got %d %s, want 422 SHOWTIME_STARTED", started.Code, started.Body)
	}

	// Bad path IDs -> 400 VALIDATION_ERROR.
	for _, id := range []string{"abc", "0", "-1"} {
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			body := ""
			if method == "PUT" {
				body = `{"seatIds":[1]}`
			}
			rec := request(t, h, method, "/api/reservations/"+id, body, bearer(tok1))
			if rec.Code != http.StatusBadRequest || decodeError(t, rec).Error.Code != "VALIDATION_ERROR" {
				t.Errorf("%s /%s: got %d %s, want 400 VALIDATION_ERROR", method, id, rec.Code, rec.Body)
			}
		}
	}

	// Unknown JSON field -> 400 VALIDATION_ERROR.
	unknownField := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[3],"userId":999}`, bearer(tok1))
	if unknownField.Code != http.StatusBadRequest || decodeError(t, unknownField).Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("unknown field: got %d %s, want 400 VALIDATION_ERROR", unknownField.Code, unknownField.Body)
	}

	// Missing/non-JSON Content-Type -> 415 UNSUPPORTED_MEDIA_TYPE.
	badCT := request(t, h, "POST", "/api/reservations", `{"showtimeId":7,"seatIds":[3]}`, map[string]string{
		"Authorization": "Bearer " + tok1,
		"Content-Type":  "text/plain",
	})
	if badCT.Code != http.StatusUnsupportedMediaType || decodeError(t, badCT).Error.Code != "UNSUPPORTED_MEDIA_TYPE" {
		t.Fatalf("bad content-type: got %d %s, want 415 UNSUPPORTED_MEDIA_TYPE", badCT.Code, badCT.Body)
	}
}
