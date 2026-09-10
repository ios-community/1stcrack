package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
)

// RoastInput represents a single roast production request.
type RoastInput struct {
	// GreenBeanID references the source green bean.
	GreenBeanID string
	// GreenWeightMg is the raw input weight in milligrams and must be positive.
	GreenWeightMg domain.WeightMg
	// RoastedWeightMg is the roasted output weight in milligrams.
	RoastedWeightMg domain.WeightMg
	// RoastLevel is Light, Medium, or Dark.
	RoastLevel string
}

// RoastingService executes roast batches with shrinkage calculation.
type RoastingService struct {
	// beans persists green beans and roast batches.
	beans *repository.BeanRepository
	// now supplies production timestamps and batch ID dates.
	now func() time.Time
}

// NewRoastingService creates a RoastingService using the given connection pool.
func NewRoastingService(db *sql.DB) *RoastingService {
	return &RoastingService{beans: repository.NewBeanRepository(db), now: time.Now}
}

// ExecuteBatch validates input, calculates shrinkage, and records a roast batch atomically.
func (s *RoastingService) ExecuteBatch(ctx context.Context, input RoastInput) (*domain.RoastBatch, error) {
	if input.GreenBeanID == "" {
		return nil, fmt.Errorf("green bean id must not be empty")
	}
	if input.GreenWeightMg <= 0 || input.RoastedWeightMg <= 0 || input.RoastedWeightMg > input.GreenWeightMg {
		return nil, domain.ErrInvalidRoastWeight
	}
	if !validRoastLevel(input.RoastLevel) {
		return nil, fmt.Errorf("invalid roast level %q: want Light, Medium, or Dark", input.RoastLevel)
	}
	if _, err := s.beans.GetGreenBean(ctx, input.GreenBeanID); err != nil {
		return nil, err
	}
	shrinkage, err := domain.CalcShrinkage(input.GreenWeightMg, input.RoastedWeightMg)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	prefix := "BATCH-" + now.Format("20060102") + "-"
	count, err := s.beans.CountBatchesForDay(ctx, prefix)
	if err != nil {
		return nil, err
	}
	batch := &domain.RoastBatch{
		ID:              fmt.Sprintf("%s%03d", prefix, count+1),
		GreenBeanID:     input.GreenBeanID,
		GreenWeightMg:   input.GreenWeightMg,
		RoastedWeightMg: input.RoastedWeightMg,
		RemainingMg:     input.RoastedWeightMg,
		ShrinkagePct:    shrinkage,
		RoastLevel:      input.RoastLevel,
		RoastedAt:       now,
	}
	if err := s.beans.DeductGreenAndCreateBatch(ctx, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// validRoastLevel reports whether the level is a supported roast profile.
func validRoastLevel(level string) bool {
	return level == domain.RoastLight || level == domain.RoastMedium || level == domain.RoastDark
}
