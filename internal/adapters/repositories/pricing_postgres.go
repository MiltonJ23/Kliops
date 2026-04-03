package repositories 

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPricing struct {
	DB *pgxpool.Pool 
}

func NewPostgresPricing(db *pgxpool.Pool) *PostgresPricing {
	return &PostgresPricing{
		DB:db,
	}
}

func (p *PostgresPricing) GetPrice(ctx context.Context, codeArticle string) (float64,error) {
	var price float64 
	// let's admit we have a table named mercuriale with two columns "code_article" and "prix_unitaire"
	query := `SELECT prix_unitaire FROM mercuriale WHERE code_article= $1 LIMIT 1`
	queryingDatabaseError := p.DB.QueryRow(ctx,query,codeArticle).Scan(&price)
	if queryingDatabaseError != nil {
		return 0, fmt.Errorf("unable to find article %s in Postgres database : %v",codeArticle,queryingDatabaseError)
	}
	return price,nil
}