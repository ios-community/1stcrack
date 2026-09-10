package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"1stcrack/internal/domain"
)

// OrderRepository provides atomic checkout persistence with FIFO stock deduction.
type OrderRepository struct {
	// db is the shared SQLite connection pool.
	db *sql.DB
}

// NewOrderRepository creates an OrderRepository using the given connection pool.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder records an order and deducts roast stock in FIFO order atomically.
//
// The order parameter must carry a pre-generated identifier, totals, and
// payment details. The items parameter holds one line per product. The needs
// parameter maps each green bean identifier to the total roasted milligrams
// required.
//
// Batches are consumed from oldest to newest. Audit rows are written to
// batch_deductions. When stock is insufficient the transaction is rolled back.
//
// It returns [domain.ErrInsufficientStock] when batches cannot fulfil needs.
func (r *OrderRepository) CreateOrder(ctx context.Context, order *domain.Order, items []domain.OrderItem, needs map[string]domain.WeightMg) error {
	if r.db == nil {
		return fmt.Errorf("order repository database must not be nil")
	}
	if order == nil {
		return fmt.Errorf("order must not be nil")
	}
	if len(items) == 0 {
		return domain.ErrEmptyCart
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin checkout transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	createdAt := order.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	const orderQ = `INSERT INTO orders (id, order_type, customer_name, total_amount, paid_amount, payment_method, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);`
	if _, err := tx.ExecContext(ctx, orderQ, order.ID, order.OrderType, order.CustomerName, int64(order.TotalAmount), int64(order.PaidAmount), order.PaymentMethod, createdAt); err != nil {
		return fmt.Errorf("insert order %s: %w", order.ID, err)
	}
	const itemQ = `INSERT INTO order_items (order_id, product_id, quantity, subtotal) VALUES (?, ?, ?, ?);`
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, itemQ, order.ID, item.ProductID, item.Quantity, int64(item.Subtotal)); err != nil {
			return fmt.Errorf("insert order item %s: %w", item.ProductID, err)
		}
	}
	for _, beanID := range sortedBeanIDs(needs) {
		if err := deductFIFO(ctx, tx, order.ID, beanID, needs[beanID]); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checkout %s: %w", order.ID, err)
	}
	committed = true
	return nil
}

// GetOrder returns the order with the given identifier.
func (r *OrderRepository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	const q = `SELECT id, order_type, COALESCE(customer_name, ''), total_amount, paid_amount, payment_method, created_at FROM orders WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, q, id)
	var order domain.Order
	var total, paid int64
	var created sql.NullTime
	if err := row.Scan(&order.ID, &order.OrderType, &order.CustomerName, &total, &paid, &order.PaymentMethod, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("scan order %s: %w", id, err)
	}
	order.TotalAmount = domain.MoneyIDR(total)
	order.PaidAmount = domain.MoneyIDR(paid)
	if created.Valid {
		order.CreatedAt = created.Time
	}
	return &order, nil
}

// ListOrdersSince returns orders created at or after the given time.
func (r *OrderRepository) ListOrdersSince(ctx context.Context, since time.Time) ([]domain.Order, error) {
	const q = `SELECT id, order_type, COALESCE(customer_name, ''), total_amount, paid_amount, payment_method, created_at
		FROM orders WHERE created_at >= ? ORDER BY created_at ASC;`
	rows, err := r.db.QueryContext(ctx, q, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.Order
	for rows.Next() {
		var order domain.Order
		var total, paid int64
		var created sql.NullTime
		if err := rows.Scan(&order.ID, &order.OrderType, &order.CustomerName, &total, &paid, &order.PaymentMethod, &created); err != nil {
			return nil, fmt.Errorf("scan order row: %w", err)
		}
		order.TotalAmount = domain.MoneyIDR(total)
		order.PaidAmount = domain.MoneyIDR(paid)
		if created.Valid {
			order.CreatedAt = created.Time
		}
		out = append(out, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return out, nil
}

// ListDeductionsSince returns FIFO audit rows created at or after the given time.
func (r *OrderRepository) ListDeductionsSince(ctx context.Context, since time.Time) ([]domain.BatchDeduction, error) {
	const q = `SELECT id, order_id, roast_batch_id, deducted_mg FROM batch_deductions WHERE created_at >= ? ORDER BY id ASC;`
	rows, err := r.db.QueryContext(ctx, q, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("list deductions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.BatchDeduction
	for rows.Next() {
		var deduction domain.BatchDeduction
		var deducted int64
		if err := rows.Scan(&deduction.ID, &deduction.OrderID, &deduction.RoastBatchID, &deducted); err != nil {
			return nil, fmt.Errorf("scan deduction row: %w", err)
		}
		deduction.DeductedMg = domain.WeightMg(deducted)
		out = append(out, deduction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deductions: %w", err)
	}
	return out, nil
}

// deductFIFO consumes needed milligrams from the oldest batches first.
func deductFIFO(ctx context.Context, tx *sql.Tx, orderID string, beanID string, needed domain.WeightMg) error {
	if needed <= 0 {
		return nil
	}
	const q = `SELECT id, remaining_mg FROM roast_batches
		WHERE green_bean_id = ? AND remaining_mg > 0 ORDER BY roasted_at ASC, id ASC;`
	rows, err := tx.QueryContext(ctx, q, beanID)
	if err != nil {
		return fmt.Errorf("query fifo batches %s: %w", beanID, err)
	}
	type batchStock struct {
		// id is the batch identifier.
		id string
		// remaining is the available stock.
		remaining domain.WeightMg
	}
	var batches []batchStock
	for rows.Next() {
		var b batchStock
		var remaining int64
		if err := rows.Scan(&b.id, &remaining); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan fifo batch %s: %w", beanID, err)
		}
		b.remaining = domain.WeightMg(remaining)
		batches = append(batches, b)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate fifo batches %s: %w", beanID, err)
	}
	_ = rows.Close()
	remaining := needed
	for _, b := range batches {
		if remaining <= 0 {
			break
		}
		take := b.remaining
		take = min(take, remaining)
		if _, err := tx.ExecContext(ctx, `UPDATE roast_batches SET remaining_mg = remaining_mg - ? WHERE id = ?;`, int64(take), b.id); err != nil {
			return fmt.Errorf("deduct batch %s: %w", b.id, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO batch_deductions (order_id, roast_batch_id, deducted_mg) VALUES (?, ?, ?);`, orderID, b.id, int64(take)); err != nil {
			return fmt.Errorf("audit deduction %s: %w", b.id, err)
		}
		remaining -= take
	}
	if remaining > 0 {
		return domain.ErrInsufficientStock
	}
	return nil
}

// sortedBeanIDs returns deterministic bean iteration order for transactions.
func sortedBeanIDs(needs map[string]domain.WeightMg) []string {
	ids := make([]string, 0, len(needs))
	for id := range needs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// CountOrdersForDay returns the number of orders with the given ID prefix.
func (r *OrderRepository) CountOrdersForDay(ctx context.Context, prefix string) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE id LIKE ?;`, prefix+"%").Scan(&n); err != nil {
		return 0, fmt.Errorf("count orders %s: %w", prefix, err)
	}
	return n, nil
}
