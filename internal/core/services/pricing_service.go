package services

import (
	"context"
	"fmt"
	"github.com/MiltonJ23/Kliops/internal/core/ports"
)


// PricingService is the orchestrator of the mercuriale 
type PricingService struct {
	Strategies map[string]ports.PricingStrategy
}

func NewPricingService() *PricingService {
	return &PricingService{
		Strategies: make(map[string]ports.PricingStrategy),
	}
}


// RegisterStrategy allows to register the pricing query strategy(postgres-database,excel,erp) under a precise name 
func (s *PricingService) RegisterStrategy(name string, strategy ports.PricingStrategy) {
	s.Strategies[name] = strategy
}


func (s *PricingService) GetPrice(ctx context.Context, source string, codeArticle string) (float64,error) {
	strategy, exists := s.Strategies[source]
	if !exists {
		return 0, fmt.Errorf("pricing strategy %s is not configured",source)
	}
	return strategy.GetPrice(ctx,codeArticle)
}