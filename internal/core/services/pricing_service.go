package services

import (
	"context"
	"fmt"
	"sync"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
)


// PricingService is the orchestrator of the mercuriale 
type PricingService struct {
	strategies map[string]ports.PricingStrategy
	mu         sync.RWMutex
}

func NewPricingService() *PricingService {
	return &PricingService{
		strategies: make(map[string]ports.PricingStrategy),
	}
}


// RegisterStrategy allows to register the pricing query strategy(postgres-database,excel,erp) under a precise name 
func (s *PricingService) RegisterStrategy(name string, strategy ports.PricingStrategy) {
	if strategy == nil {
		fmt.Printf("Warning: nil strategy provided for %s\n", name)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategies[name] = strategy
}


func (s *PricingService) GetPrice(ctx context.Context, source string, codeArticle string) (float64,error) {
	s.mu.RLock()
	strategy, exists := s.strategies[source]
	s.mu.RUnlock()
	
	if !exists {
		return 0, fmt.Errorf("pricing strategy %s is not configured",source)
	}
	return strategy.GetPrice(ctx,codeArticle)
}