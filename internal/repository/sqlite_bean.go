package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"1stcrack/internal/domain"
)

// BeanRepository provides SQLite persistence for green beans and roast batches.
type BeanRepository struct {
	// db is the shared SQLite connection pool.
	db *sql.DB
}

// NewBeanRepository creates a BeanRepository using the given connection pool.
func NewBeanRepository(db *sql.DB) *BeanRepository {
	return &BeanRepository{db: db}
}

// CreateGreenBean inserts a new green bean record.
func (r *BeanRepository) CreateGreenBean(ctx context.Context, bean *domain.GreenBean) error {
	if r.db == nil {
		return fmt.Errorf("bean repository database must not be nil")
	}
	if bean == nil {
		return fmt.Errorf("green bean must not be nil")
	}
	const q = `INSERT INTO green_beans (id, name, origin, process, stock_mg, cost_per_kg) VALUES (?, ?, ?, ?, ?, ?);`
	if _, err := r.db.ExecContext(ctx, q, bean.ID, bean.Name, bean.Origin, bean.Process, int64(bean.StockMg), int64(bean.CostPerKg)); err != nil {
		return fmt.Errorf("insert green bean %s: %w", bean.ID, err)
	}
	return nil
}

// GetGreenBean returns the green bean with the given identifier.
func (r *BeanRepository) GetGreenBean(ctx context.Context, id string) (*domain.GreenBean, error) {
	const q = `SELECT id, name, origin, process, stock_mg, cost_per_kg, created_at FROM green_beans WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, q, id)
	var bean domain.GreenBean
	var stock, cost int64
	var created sql.NullTime
	if err := row.Scan(&bean.ID, &bean.Name, &bean.Origin, &bean.Process, &stock, &cost, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("scan green bean %s: %w", id, err)
	}
	bean.StockMg = domain.WeightMg(stock)
	bean.CostPerKg = domain.MoneyIDR(cost)
	if created.Valid {
		bean.CreatedAt = created.Time
	}
	return &bean, nil
}

// UpdateGreenStock adjusts raw stock by the given delta, which may be negative.
func (r *BeanRepository) UpdateGreenStock(ctx context.Context, id string, deltaMg domain.WeightMg) error {
	const q = `UPDATE green_beans SET stock_mg = stock_mg + ? WHERE id = ? AND stock_mg + ? >= 0;`
	res, err := r.db.ExecContext(ctx, q, int64(deltaMg), id, int64(deltaMg))
	if err != nil {
		return fmt.Errorf("update green stock %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count green stock rows %s: %w", id, err)
	}
	if n == 0 {
		if _, gerr := r.GetGreenBean(ctx, id); gerr != nil {
			return gerr
		}
		return domain.ErrInsufficientStock
	}
	return nil
}

// CreateRoastBatch inserts a new roast batch record.
func (r *BeanRepository) CreateRoastBatch(ctx context.Context, batch *domain.RoastBatch) error {
	if batch == nil {
		return fmt.Errorf("roast batch must not be nil")
	}
	roastedAt := batch.RoastedAt
	if roastedAt.IsZero() {
		roastedAt = time.Now().UTC()
	}
	const q = `INSERT INTO roast_batches (id, green_bean_id, green_weight_mg, roasted_weight_mg, remaining_mg, shrinkage_pct, roast_level, roasted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	if _, err := r.db.ExecContext(ctx, q, batch.ID, batch.GreenBeanID, int64(batch.GreenWeightMg), int64(batch.RoastedWeightMg), int64(batch.RemainingMg), batch.ShrinkagePct, batch.RoastLevel, roastedAt); err != nil {
		return fmt.Errorf("insert roast batch %s: %w", batch.ID, err)
	}
	return nil
}

// GetRoastBatch returns the roast batch with the given identifier.
func (r *BeanRepository) GetRoastBatch(ctx context.Context, id string) (*domain.RoastBatch, error) {
	const q = `SELECT id, green_bean_id, green_weight_mg, roasted_weight_mg, remaining_mg, shrinkage_pct, roast_level, roasted_at
		FROM roast_batches WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, q, id)
	batch, err := scanRoastBatch(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan roast batch %s: %w", id, err)
	}
	return batch, nil
}

// ListActiveBatches returns batches with remaining stock in FIFO order.
//
// The beanID parameter selects a single green bean lineage. Results are
// ordered by production time from oldest to newest.
func (r *BeanRepository) ListActiveBatches(ctx context.Context, beanID string) ([]domain.RoastBatch, error) {
	const q = `SELECT id, green_bean_id, green_weight_mg, roasted_weight_mg, remaining_mg, shrinkage_pct, roast_level, roasted_at
		FROM roast_batches WHERE green_bean_id = ? AND remaining_mg > 0 ORDER BY roasted_at ASC, id ASC;`
	rows, err := r.db.QueryContext(ctx, q, beanID)
	if err != nil {
		return nil, fmt.Errorf("list active batches %s: %w", beanID, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.RoastBatch
	for rows.Next() {
		batch, err := scanRoastBatch(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active batch %s: %w", beanID, err)
		}
		out = append(out, *batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active batches %s: %w", beanID, err)
	}
	return out, nil
}

// rowScanner abstracts sql.Row and sql.Rows for batch scanning.
type rowScanner interface {
	// Scan copies column values into destinations.
	Scan(dest ...any) error
}

// scanRoastBatch converts the current row into a RoastBatch.
func scanRoastBatch(row rowScanner) (*domain.RoastBatch, error) {
	var batch domain.RoastBatch
	var green, roasted, remaining int64
	var roastedAt sql.NullTime
	if err := row.Scan(&batch.ID, &batch.GreenBeanID, &green, &roasted, &remaining, &batch.ShrinkagePct, &batch.RoastLevel, &roastedAt); err != nil {
		return nil, err
	}
	batch.GreenWeightMg = domain.WeightMg(green)
	batch.RoastedWeightMg = domain.WeightMg(roasted)
	batch.RemainingMg = domain.WeightMg(remaining)
	if roastedAt.Valid {
		batch.RoastedAt = roastedAt.Time
	}
	return &batch, nil
}

// ListGreenBeans returns all green bean records ordered by name.
func (r *BeanRepository) ListGreenBeans(ctx context.Context) ([]domain.GreenBean, error) {
	const q = `SELECT id, name, origin, process, stock_mg, cost_per_kg, created_at FROM green_beans ORDER BY name ASC;`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list green beans: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.GreenBean
	for rows.Next() {
		var bean domain.GreenBean
		var stock, cost int64
		var created sql.NullTime
		if err := rows.Scan(&bean.ID, &bean.Name, &bean.Origin, &bean.Process, &stock, &cost, &created); err != nil {
			return nil, fmt.Errorf("scan green bean row: %w", err)
		}
		bean.StockMg = domain.WeightMg(stock)
		bean.CostPerKg = domain.MoneyIDR(cost)
		if created.Valid {
			bean.CreatedAt = created.Time
		}
		out = append(out, bean)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate green beans: %w", err)
	}
	return out, nil
}

// CountBatchesForDay returns the number of batches with the given ID prefix.
func (r *BeanRepository) CountBatchesForDay(ctx context.Context, prefix string) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roast_batches WHERE id LIKE ?;`, prefix+"%").Scan(&n); err != nil {
		return 0, fmt.Errorf("count batches %s: %w", prefix, err)
	}
	return n, nil
}

// DeductGreenAndCreateBatch deducts raw stock and inserts a roast batch atomically.
func (r *BeanRepository) DeductGreenAndCreateBatch(ctx context.Context, batch *domain.RoastBatch) error {
	if batch == nil {
		return fmt.Errorf("roast batch must not be nil")
	}
	roastedAt := batch.RoastedAt
	if roastedAt.IsZero() {
		roastedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin roast transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, `UPDATE green_beans SET stock_mg = stock_mg - ? WHERE id = ? AND stock_mg >= ?;`, int64(batch.GreenWeightMg), batch.GreenBeanID, int64(batch.GreenWeightMg))
	if err != nil {
		return fmt.Errorf("deduct green stock %s: %w", batch.GreenBeanID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deducted rows %s: %w", batch.GreenBeanID, err)
	}
	if n == 0 {
		return domain.ErrInsufficientStock
	}
	const q = `INSERT INTO roast_batches (id, green_bean_id, green_weight_mg, roasted_weight_mg, remaining_mg, shrinkage_pct, roast_level, roasted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	if _, err := tx.ExecContext(ctx, q, batch.ID, batch.GreenBeanID, int64(batch.GreenWeightMg), int64(batch.RoastedWeightMg), int64(batch.RemainingMg), batch.ShrinkagePct, batch.RoastLevel, roastedAt); err != nil {
		return fmt.Errorf("insert roast batch %s: %w", batch.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit roast batch %s: %w", batch.ID, err)
	}
	committed = true
	return nil
}
