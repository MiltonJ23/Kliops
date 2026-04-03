package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler is a simple handler that returns 200 OK when reached.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestAPIKeyMiddleware_NoEnvKey_Returns500(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "anything")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when API_KEY_SECRET is empty, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_ValidKey_Passes(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "test-secret-key")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "test-secret-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid API key, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_WrongKey_Returns401(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "correct-key")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "wrong-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong API key, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_MissingHeader_Returns401(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "correct-key")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	// no X-API-KEY header
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when API key header is absent, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_EmptyHeader_Returns401(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "correct-key")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty API key header, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_NextNotCalledOnFailure(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "secret")

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "bad-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Error("next handler should NOT be called when API key is invalid")
	}
}

func TestAPIKeyMiddleware_CaseSensitiveKey(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "MySecretKey")

	handler := APIKeyMiddleware(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/price", nil)
	req.Header.Set("X-API-KEY", "mysecretkey") // lowercase variant
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("API key comparison must be case-sensitive; expected 401, got %d", rr.Code)
	}
}