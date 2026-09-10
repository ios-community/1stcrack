package service

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
)

// BatchConsumption represents coffee consumed from one roast batch.
type BatchConsumption struct {
	// BatchID references the consumed roast batch.
	BatchID string
	// ConsumedMg is the total milligrams consumed in the period.
	ConsumedMg domain.WeightMg
}

// DailySummary aggregates sales and consumption for one calendar day.
type DailySummary struct {
	// Revenue is the total sales in Rupiah.
	Revenue domain.MoneyIDR
	// Count is the transaction count.
	Count int
	// ByMethod maps payment methods to revenue.
	ByMethod map[string]domain.MoneyIDR
	// ConsumedMg is the total roasted consumption in milligrams.
	ConsumedMg domain.WeightMg
	// PerBatch breaks consumption down by roast batch in identifier order.
	PerBatch []BatchConsumption
}

// Reporter aggregates daily sales and consumption from stored orders.
type Reporter struct {
	// orders reads orders and FIFO audit rows.
	orders *repository.OrderRepository
}

// NewReporter creates a Reporter using the given connection pool.
func NewReporter(db *sql.DB) *Reporter {
	return &Reporter{orders: repository.NewOrderRepository(db)}
}

// SummariseDay aggregates orders and deductions since local midnight of today.
func (r *Reporter) SummariseDay(ctx context.Context) (DailySummary, error) {
	return SummariseDay(ctx, r.orders, time.Now())
}

// SummariseDay aggregates orders and deductions since local midnight of now.
func SummariseDay(ctx context.Context, orders *repository.OrderRepository, now time.Time) (DailySummary, error) {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	summary := DailySummary{ByMethod: make(map[string]domain.MoneyIDR)}
	list, err := orders.ListOrdersSince(ctx, midnight)
	if err != nil {
		return DailySummary{}, err
	}
	for _, order := range list {
		summary.Revenue += order.TotalAmount
		summary.Count++
		summary.ByMethod[order.PaymentMethod] += order.TotalAmount
	}
	deductions, err := orders.ListDeductionsSince(ctx, midnight)
	if err != nil {
		return DailySummary{}, err
	}
	perBatch := make(map[string]domain.WeightMg)
	for _, deduction := range deductions {
		summary.ConsumedMg += deduction.DeductedMg
		perBatch[deduction.RoastBatchID] += deduction.DeductedMg
	}
	ids := make([]string, 0, len(perBatch))
	for id := range perBatch {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		summary.PerBatch = append(summary.PerBatch, BatchConsumption{BatchID: id, ConsumedMg: perBatch[id]})
	}
	return summary, nil
}
