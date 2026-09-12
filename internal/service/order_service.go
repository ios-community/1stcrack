package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
)

// CartItem represents a single product line in the shopping cart.
type CartItem struct {
	// ProductID references the sold product.
	ProductID string
	// Quantity is the number of units and must be positive.
	Quantity int
}

// CheckoutRequest represents a payment attempt for the current cart.
type CheckoutRequest struct {
	// OrderType is B2C for retail or B2B for wholesale. Empty defaults to B2C.
	OrderType string
	// CustomerName is the buyer name and is required for B2B wholesale.
	CustomerName string
	// Items holds one entry per product.
	Items []CartItem
	// PaidAmount is the amount tendered in Rupiah.
	PaidAmount domain.MoneyIDR
	// PaymentMethod is CASH, QRIS, or TRANSFER.
	PaymentMethod string
}

// OrderService validates carts and processes atomic checkouts.
type OrderService struct {
	// Catalogue and recipe source.
	products *repository.ProductRepository
	// Order persistence with FIFO deduction.
	orders *repository.OrderRepository
	// Order timestamp and identifier date source.
	now func() time.Time
}

// NewOrderService creates an OrderService using the given connection pool.
func NewOrderService(db *sql.DB) *OrderService {
	return &OrderService{products: repository.NewProductRepository(db), orders: repository.NewOrderRepository(db), now: time.Now}
}

// CalculateTotal returns the cart total without touching stock.
func (s *OrderService) CalculateTotal(ctx context.Context, items []CartItem) (domain.MoneyIDR, error) {
	if len(items) == 0 {
		return 0, domain.ErrEmptyCart
	}
	var total domain.MoneyIDR
	for _, item := range items {
		if item.Quantity <= 0 {
			return 0, fmt.Errorf("invalid quantity %d for product %s: must be positive", item.Quantity, item.ProductID)
		}
		product, err := s.products.GetProduct(ctx, item.ProductID)
		if err != nil {
			return 0, err
		}
		if !product.IsActive {
			return 0, domain.ErrProductNotFound
		}
		total += product.Price * domain.MoneyIDR(item.Quantity)
	}
	return total, nil
}

// ProcessCheckout validates payment, expands recipes, and records the order
// atomically.
func (s *OrderService) ProcessCheckout(ctx context.Context, req CheckoutRequest) (*domain.Order, error) {
	orderType := req.OrderType
	if orderType == "" {
		orderType = domain.OrderTypeB2C
	}
	if orderType != domain.OrderTypeB2C && orderType != domain.OrderTypeB2B {
		return nil, fmt.Errorf("invalid order type %q: want B2C or B2B", req.OrderType)
	}
	if orderType == domain.OrderTypeB2B && req.CustomerName == "" {
		return nil, fmt.Errorf("customer name is required for B2B wholesale")
	}
	if !validPaymentMethod(req.PaymentMethod) {
		return nil, fmt.Errorf("invalid payment method %q: want CASH, QRIS, or TRANSFER", req.PaymentMethod)
	}
	total, err := s.CalculateTotal(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	if req.PaidAmount < total {
		return nil, domain.ErrNegativePayment
	}
	items, needs, err := s.expandNeeds(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	prefix := "ORD-" + now.Format("20060102") + "-"
	count, err := s.orders.CountOrdersForDay(ctx, prefix)
	if err != nil {
		return nil, err
	}
	order := &domain.Order{
		ID:            fmt.Sprintf("%s%04d", prefix, count+1),
		OrderType:     orderType,
		CustomerName:  req.CustomerName,
		TotalAmount:   total,
		PaidAmount:    req.PaidAmount,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     now,
	}
	if err := s.orders.CreateOrder(ctx, order, items, needs); err != nil {
		return nil, err
	}
	return order, nil
}

// expandNeeds converts cart lines into order items and per-bean milligram
// needs.
func (s *OrderService) expandNeeds(ctx context.Context, cart []CartItem) ([]domain.OrderItem, map[string]domain.WeightMg, error) {
	needs := make(map[string]domain.WeightMg)
	items := make([]domain.OrderItem, 0, len(cart))
	for _, line := range cart {
		product, err := s.products.GetProduct(ctx, line.ProductID)
		if err != nil {
			return nil, nil, err
		}
		if !product.IsActive {
			return nil, nil, domain.ErrProductNotFound
		}
		items = append(items, domain.OrderItem{ProductID: product.ID, Quantity: line.Quantity, Subtotal: product.Price * domain.MoneyIDR(line.Quantity)})
		recipes, err := s.products.GetRecipes(ctx, product.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, recipe := range recipes {
			if recipe.GreenBeanID == "" {
				continue
			}
			needs[recipe.GreenBeanID] += recipe.RequiredRoastedMg * domain.WeightMg(line.Quantity)
		}
	}
	return items, needs, nil
}

// validPaymentMethod reports whether the method is a supported payment type.
func validPaymentMethod(method string) bool {
	return method == domain.PaymentCash || method == domain.PaymentQRIS || method == domain.PaymentTransfer
}
