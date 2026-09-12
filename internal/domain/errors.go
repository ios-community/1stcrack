package domain

import "errors"

// ErrInsufficientStock is returned when roast batches cannot fulfil the
// required quantity.
var ErrInsufficientStock = errors.New("insufficient coffee stock for this order")

// ErrInvalidRoastWeight is returned when roasted weight exceeds green
// weight or inputs are non-positive.
var ErrInvalidRoastWeight = errors.New("roasted weight must not exceed green weight")

// ErrProductNotFound is returned when the requested product does not exist
// or is inactive.
var ErrProductNotFound = errors.New("product not found")

// ErrEmptyCart is returned when checkout is attempted with no cart items.
var ErrEmptyCart = errors.New("shopping cart is empty")

// ErrNegativePayment is returned when the paid amount is less than the
// order total.
var ErrNegativePayment = errors.New("paid amount is less than the order total")
