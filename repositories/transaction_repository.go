package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"kasir-api/models"
	"strings"

	"github.com/lib/pq"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func bulkInsertTransactionDetails(
	tx *sql.Tx,
	transactionID int,
	details []models.TransactionDetail,
) error {

	if len(details) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(details))
	valueArgs := make([]interface{}, 0, len(details)*4)

	for i, d := range details {
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d,$%d,$%d,$%d)",
				i*4+1, i*4+2, i*4+3, i*4+4,
			),
		)

		valueArgs = append(valueArgs,
			transactionID,
			d.ProductID,
			d.Quantity,
			d.Subtotal,
		)
	}

	query := `
		INSERT INTO transaction_details
		(transaction_id, product_id, quantity, subtotal)
		VALUES ` + strings.Join(valueStrings, ",")

	_, err := tx.Exec(query, valueArgs...)
	return err
}

func (repo *TransactionRepository) CreateTransaction(
	items []models.CheckoutItem,
) (*models.Transaction, error) {

	tx, err := repo.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if len(items) == 0 {
		return nil, errors.New("checkout item kosong")
	}

	/* =========================
	   1. Ambil semua product (1 query)
	========================= */
	productIDs := make([]int, 0, len(items))
	for _, item := range items {
		productIDs = append(productIDs, item.ProductID)
	}

	rows, err := tx.Query(`
		SELECT id, name, price, stock
		FROM products
		WHERE id = ANY($1::bigint[])
		FOR UPDATE
	`, pq.Array(productIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type product struct {
		Name  string
		Price int
		Stock int
	}

	products := make(map[int]product)

	for rows.Next() {
		var id int
		var p product
		if err := rows.Scan(&id, &p.Name, &p.Price, &p.Stock); err != nil {
			return nil, err
		}
		products[id] = p
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	/* =========================
	   2. Validasi + hitung total + detail
	========================= */
	totalAmount := 0
	details := make([]models.TransactionDetail, 0, len(items))

	for _, item := range items {
		p, ok := products[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("product id %d not found", item.ProductID)
		}

		if item.Quantity > p.Stock {
			return nil, fmt.Errorf("stock product %d tidak cukup", item.ProductID)
		}

		subtotal := p.Price * item.Quantity
		totalAmount += subtotal

		details = append(details, models.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: p.Name,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	/* =========================
	   3. Update stock (bulk, 1 query)
	========================= */
	updateValues := make([]string, 0, len(items))
	updateArgs := make([]interface{}, 0, len(items)*2)

	for i, item := range items {
		updateValues = append(updateValues,
			fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2),
		)
		updateArgs = append(updateArgs, item.ProductID, item.Quantity)
	}

	updateQuery := `
		UPDATE products p
		SET stock = p.stock - v.qty::integer
		FROM (VALUES ` + strings.Join(updateValues, ",") + `) AS v(id, qty)
		WHERE p.id = v.id::bigint
	`

	if _, err := tx.Exec(updateQuery, updateArgs...); err != nil {
		return nil, err
	}

	/* =========================
	   4. Insert transaction (1 query)
	========================= */
	var transactionID int
	err = tx.QueryRow(
		"INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id",
		totalAmount,
	).Scan(&transactionID)
	if err != nil {
		return nil, err
	}

	/* =========================
	   5. Insert transaction details (bulk insert, 1 query)
	========================= */
	if err := bulkInsertTransactionDetails(tx, transactionID, details); err != nil {
		return nil, err
	}

	/* =========================
	   6. Commit
	========================= */
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Transaction{
		ID:          transactionID,
		TotalAmount: totalAmount,
		Details:     details,
	}, nil
}
