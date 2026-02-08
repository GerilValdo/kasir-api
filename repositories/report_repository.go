package repositories

import (
	"database/sql"
	"kasir-api/models"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (repo *ReportRepository) GetSalesSummary(startDate, endDate *string) (*models.SalesSummary, error) {
	var summary models.SalesSummary

	dateFilterTx := "DATE(created_at) = CURRENT_DATE"
	dateFilterJoin := "DATE(t.created_at) = CURRENT_DATE"
	args := []any{}

	if startDate != nil && endDate != nil {
		dateFilterTx = "DATE(created_at) BETWEEN $1 AND $2"
		dateFilterJoin = "DATE(t.created_at) BETWEEN $1 AND $2"
		args = append(args, *startDate, *endDate)
	}

	querySummary := `
		SELECT 
			COALESCE(SUM(total_amount), 0),
			COUNT(*)
		FROM transactions
		WHERE ` + dateFilterTx

	err := repo.db.QueryRow(querySummary, args...).Scan(
		&summary.TotalRevenue,
		&summary.TotalTransactions,
	)
	if err != nil {
		return nil, err
	}

	queryBestSeller := `
	SELECT p.name, SUM(td.quantity) AS qty
	FROM transaction_details td
	JOIN transactions t ON t.id = td.transaction_id
	JOIN products p ON p.id = td.product_id
	WHERE ` + dateFilterJoin + `
	GROUP BY p.name
	ORDER BY qty DESC
	LIMIT 1
`
	err = repo.db.QueryRow(queryBestSeller, args...).Scan(
		&summary.BestSellerProduct.Name,
		&summary.BestSellerProduct.QuantitySold,
	)

	if err == sql.ErrNoRows {
		summary.BestSellerProduct = models.BestSellerProduct{}
		return &summary, nil
	}

	if err != nil {
		return nil, err
	}

	return &summary, nil

}
