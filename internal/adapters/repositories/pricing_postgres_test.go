package repositories

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewPostgresPricing_SetsDB(t *testing.T) {
	// pgxpool.Pool cannot be instantiated without a real DB connection,
	// but we can verify the constructor wires the field correctly using a nil pool.
	// In production, a non-nil pool would be provided.
	var pool *pgxpool.Pool // nil pool
	pp := NewPostgresPricing(pool)

	if pp == nil {
		t.Fatal("NewPostgresPricing returned nil")
	}
	if pp.DB != pool {
		t.Error("DB field not set correctly by constructor")
	}
}

func TestNewPostgresPricing_ReturnsNonNilStruct(t *testing.T) {
	pp := NewPostgresPricing(nil)
	if pp == nil {
		t.Fatal("NewPostgresPricing should never return nil")
	}
}

// TestPostgresPricing_GetPrice_RequiresIntegration documents that GetPrice
// requires a live PostgreSQL connection and cannot be unit-tested without
// a database mock or real connection. Run with integration build tags.
func TestPostgresPricing_GetPrice_RequiresIntegration(t *testing.T) {
	t.Skip("GetPrice requires a live PostgreSQL connection; use integration tests with a real or dockerised DB")
}