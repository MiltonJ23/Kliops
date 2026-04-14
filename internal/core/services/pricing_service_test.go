package services

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type mockPricingStrategy struct {
	price float64
	err   error
}

func (m *mockPricingStrategy) GetPrice(_ context.Context, _ string) (float64, error) {
	return m.price, m.err
}

func TestNewPricingService(t *testing.T) {
	svc := NewPricingService()
	if svc == nil {
		t.Fatal("NewPricingService returned nil")
	}
	// Verify an empty service has no strategies: any lookup should fail.
	_, err := svc.GetPrice(context.Background(), "any", "code")
	if err == nil {
		t.Fatal("expected error for unregistered strategy on new service, got nil")
	}
}

func TestRegisterStrategy(t *testing.T) {
	svc := NewPricingService()
	mock := &mockPricingStrategy{price: 10.0}

	svc.RegisterStrategy("excel", mock)

	// Verify the strategy is registered by calling GetPrice successfully.
	price, err := svc.GetPrice(context.Background(), "excel", "ART01")
	if err != nil {
		t.Fatalf("expected 'excel' strategy to be registered, got error: %v", err)
	}
	if price != 10.0 {
		t.Errorf("expected price 10.0 from registered strategy, got %f", price)
	}
}

func TestRegisterStrategy_OverwritesExisting(t *testing.T) {
	svc := NewPricingService()
	first := &mockPricingStrategy{price: 10.0}
	second := &mockPricingStrategy{price: 99.0}

	svc.RegisterStrategy("excel", first)
	svc.RegisterStrategy("excel", second)

	// After overwrite, GetPrice should use the second (latest) strategy.
	price, err := svc.GetPrice(context.Background(), "excel", "ART01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 99.0 {
		t.Errorf("expected price 99.0 after overwrite, got %f", price)
	}
}

func TestGetPrice_KnownStrategy_ReturnsPrice(t *testing.T) {
	svc := NewPricingService()
	svc.RegisterStrategy("excel", &mockPricingStrategy{price: 150.50})

	price, err := svc.GetPrice(context.Background(), "excel", "ART01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 150.50 {
		t.Errorf("expected 150.50, got %f", price)
	}
}

func TestGetPrice_UnknownStrategy_ReturnsError(t *testing.T) {
	svc := NewPricingService()

	_, err := svc.GetPrice(context.Background(), "nonexistent", "ART01")
	if err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
	expectedMsg := "pricing strategy nonexistent is not configured"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestGetPrice_StrategyReturnsError_PropagatesError(t *testing.T) {
	svc := NewPricingService()
	expectedErr := errors.New("article not found")
	svc.RegisterStrategy("erp", &mockPricingStrategy{err: expectedErr})

	_, err := svc.GetPrice(context.Background(), "erp", "UNKNOWN")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error %v, got %v", expectedErr, err)
	}
}

func TestGetPrice_MultipleStrategies_RoutesCorrectly(t *testing.T) {
	svc := NewPricingService()
	svc.RegisterStrategy("excel", &mockPricingStrategy{price: 100.0})
	svc.RegisterStrategy("erp", &mockPricingStrategy{price: 200.0})
	svc.RegisterStrategy("postgres", &mockPricingStrategy{price: 300.0})

	cases := []struct {
		source        string
		expectedPrice float64
	}{
		{"excel", 100.0},
		{"erp", 200.0},
		{"postgres", 300.0},
	}

	for _, tc := range cases {
		price, err := svc.GetPrice(context.Background(), tc.source, "ART01")
		if err != nil {
			t.Errorf("[%s] unexpected error: %v", tc.source, err)
			continue
		}
		if price != tc.expectedPrice {
			t.Errorf("[%s] expected %.2f, got %.2f", tc.source, tc.expectedPrice, price)
		}
	}
}

func TestGetPrice_ZeroPriceIsValid(t *testing.T) {
	svc := NewPricingService()
	svc.RegisterStrategy("excel", &mockPricingStrategy{price: 0.0})

	price, err := svc.GetPrice(context.Background(), "excel", "FREE01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0.0 {
		t.Errorf("expected 0.0 for zero-priced article, got %f", price)
	}
}

func TestGetPrice_EmptySourceName_ReturnsError(t *testing.T) {
	svc := NewPricingService()

	_, err := svc.GetPrice(context.Background(), "", "ART01")
	if err == nil {
		t.Fatal("expected error for empty source, got nil")
	}
}
func TestPricingService_RegisterAndQuery(t *testing.T) {
	svc := NewPricingService()
	mock := &mockPricingStrategy{price: 42.5}
	svc.RegisterStrategy("excel", mock)

	price, err := svc.GetPrice(context.Background(), "excel", "ART-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 42.5 {
		t.Errorf("expected 42.5, got %f", price)
	}
}

func TestPricingService_UnknownSource(t *testing.T) {
	svc := NewPricingService()

	_, err := svc.GetPrice(context.Background(), "nonexistent", "ART-001")
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestPricingService_StrategyErrorPropagated(t *testing.T) {
	svc := NewPricingService()
	expectedErr := errors.New("db unavailable")
	mock := &mockPricingStrategy{err: expectedErr}
	svc.RegisterStrategy("postgres", mock)

	_, err := svc.GetPrice(context.Background(), "postgres", "ART-002")
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestPricingService_NilStrategyIgnored(t *testing.T) {
	svc := NewPricingService()

	// Must not panic and must not register the nil strategy.
	svc.RegisterStrategy("x", nil)

	_, err := svc.GetPrice(context.Background(), "x", "ART-003")
	if err == nil {
		t.Fatal("expected error because nil strategy should not be registered")
	}
}

func TestPricingService_ConcurrentAccess(t *testing.T) {
	svc := NewPricingService()
	mock := &mockPricingStrategy{price: 1.0}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				svc.RegisterStrategy("src", mock)
			} else {
				svc.GetPrice(context.Background(), "src", "ART-X") //nolint:errcheck
			}
		}(i)
	}
	wg.Wait()
}
