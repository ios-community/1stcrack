package service

import (
	"context"
	"database/sql"
	"fmt"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
)

// AlertLevel represents a stock threshold state.
type AlertLevel string

// Supported stock alert states.
const (
	// AlertOK indicates stock at or above threshold.
	AlertOK AlertLevel = "ok"
	// AlertLow indicates stock below threshold but above half threshold.
	AlertLow AlertLevel = "low"
	// AlertCritical indicates depleted stock or stock below half threshold.
	AlertCritical AlertLevel = "critical"
)

// StockAlert represents a green bean stock state against a threshold.
type StockAlert struct {
	// BeanID references the monitored green bean.
	BeanID string
	// BeanName is the display name of the green bean.
	BeanName string
	// StockMg is the remaining raw stock in milligrams.
	StockMg domain.WeightMg
	// RoastRemainingMg is the total remaining roasted stock in milligrams.
	RoastRemainingMg domain.WeightMg
	// ThresholdMg is the monitored minimum in milligrams.
	ThresholdMg domain.WeightMg
	// Level is the computed alert state.
	Level AlertLevel
}

// BatchAlert represents a single roast batch stock state.
type BatchAlert struct {
	// BatchID references the monitored roast batch.
	BatchID string
	// BeanID references the source green bean.
	BeanID string
	// RemainingMg is the remaining roasted stock in milligrams.
	RemainingMg domain.WeightMg
	// Level is the computed alert state.
	Level AlertLevel
}

// InventoryService reports stock states against configurable thresholds.
type InventoryService struct {
	// Reads green beans and roast batches.
	beans *repository.BeanRepository
}

// NewInventoryService creates an InventoryService using the given
// connection pool.
func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{beans: repository.NewBeanRepository(db)}
}

// AlertLevelFor computes the alert state for a stock level.
//
// A non-positive threshold disables monitoring and always returns [AlertOK].
// Depleted stock returns [AlertCritical]. Stock below half the threshold
// returns [AlertCritical], stock below threshold returns [AlertLow], and
// anything at or above threshold returns [AlertOK].
func AlertLevelFor(remaining domain.WeightMg, threshold domain.WeightMg) AlertLevel {
	if threshold <= 0 {
		return AlertOK
	}
	if remaining <= 0 {
		return AlertCritical
	}
	if remaining*2 < threshold {
		return AlertCritical
	}
	if remaining < threshold {
		return AlertLow
	}
	return AlertOK
}

// ListGreenBeanAlerts returns one alert per green bean for the given threshold.
func (s *InventoryService) ListGreenBeanAlerts(ctx context.Context, threshold domain.WeightMg) ([]StockAlert, error) {
	beans, err := s.beans.ListGreenBeans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]StockAlert, 0, len(beans))
	for _, bean := range beans {
		batches, err := s.beans.ListActiveBatches(ctx, bean.ID)
		if err != nil {
			return nil, fmt.Errorf("list batches %s: %w", bean.ID, err)
		}
		var roastTotal domain.WeightMg
		for _, b := range batches {
			roastTotal += b.RemainingMg
		}
		out = append(out, StockAlert{BeanID: bean.ID, BeanName: bean.Name, StockMg: bean.StockMg, RoastRemainingMg: roastTotal, ThresholdMg: threshold, Level: AlertLevelFor(bean.StockMg, threshold)})
	}
	return out, nil
}

// ListRoastBatchAlerts returns one alert per active roast batch for the
// given threshold.
func (s *InventoryService) ListRoastBatchAlerts(ctx context.Context, beanID string, threshold domain.WeightMg) ([]BatchAlert, error) {
	batches, err := s.beans.ListActiveBatches(ctx, beanID)
	if err != nil {
		return nil, err
	}
	out := make([]BatchAlert, 0, len(batches))
	for _, b := range batches {
		out = append(out, BatchAlert{BatchID: b.ID, BeanID: b.GreenBeanID, RemainingMg: b.RemainingMg, Level: AlertLevelFor(b.RemainingMg, threshold)})
	}
	return out, nil
}
