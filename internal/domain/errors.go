package domain

import "errors"

// ErrInsufficientStock is returned when roast batches cannot fulfil the required quantity.
var ErrInsufficientStock = errors.New("stok kopi tidak mencukupi untuk pesanan ini")

// ErrInvalidRoastWeight is returned when roasted weight exceeds green weight or inputs are non-positive.
var ErrInvalidRoastWeight = errors.New("berat matang tidak boleh lebih besar dari berat mentah")

// ErrProductNotFound is returned when the requested product does not exist or is inactive.
var ErrProductNotFound = errors.New("produk tidak ditemukan")

// ErrEmptyCart is returned when checkout is attempted with no cart items.
var ErrEmptyCart = errors.New("keranjang belanja masih kosong")

// ErrNegativePayment is returned when the paid amount is less than the order total.
var ErrNegativePayment = errors.New("nominal pembayaran kurang dari total belanja")
