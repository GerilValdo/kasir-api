package models

type BestSellerProduct struct {
	Name         string `json:"name"`
	QuantitySold int    `json:"quantity_sold"`
}

type SalesSummary struct {
	TotalRevenue      int               `json:"total_revenue"`
	TotalTransactions int               `json:"total_transactions"`
	BestSellerProduct BestSellerProduct `json:"best_seller_product"`
}
