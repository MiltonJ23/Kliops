package ports 

import "context"


// PricingStrategy is the contract for querying the mercurial prices , it's gonna be implemented for database(postgres) , excel and ERP
type PricingStrategy interface {

	GetPrice(ctx context.Context, codeArticle string) (float64,error)
}