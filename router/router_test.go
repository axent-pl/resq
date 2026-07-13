package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterDispatchesByMethodAndPathParam(t *testing.T) {
	router := NewRouter()
	router.HandleFunc(http.MethodGet, "/incident/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(PathParam(r, "id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/incident/42", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "42" {
		t.Fatalf("expected path param 42, got %q", rec.Body.String())
	}
}

func TestRouterReturnsMethodNotAllowedForMatchingPath(t *testing.T) {
	router := NewRouter()
	router.HandleFunc(http.MethodGet, "/incidents", func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodPost, "/incidents", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func TestRouterReturnsNotFoundForMissingPath(t *testing.T) {
	router := NewRouter()
	router.HandleFunc(http.MethodGet, "/incidents", func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}
