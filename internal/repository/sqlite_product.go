package repository

import (
	"context"
	"database/sql"
	"fmt"

	"1stcrack/internal/domain"
)

// ProductRepository provides SQLite persistence for products and recipes.
type ProductRepository struct {
	// Shared SQLite connection pool.
	db *sql.DB
}

// NewProductRepository creates a ProductRepository using the given
// connection pool.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// ListActiveProducts returns all sellable products ordered by name.
func (r *ProductRepository) ListActiveProducts(ctx context.Context) ([]domain.Product, error) {
	const q = `SELECT id, name, category, price, is_active FROM products WHERE is_active = 1 ORDER BY name ASC;`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list active products: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.Product
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active product: %w", err)
		}
		out = append(out, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active products: %w", err)
	}
	return out, nil
}

// GetProduct returns the product with the given identifier, active or not.
func (r *ProductRepository) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	const q = `SELECT id, name, category, price, is_active FROM products WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, q, id)
	product, err := scanProduct(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan product %s: %w", id, err)
	}
	return product, nil
}

// GetRecipes returns all bill-of-materials lines for a product.
func (r *ProductRepository) GetRecipes(ctx context.Context, productID string) ([]domain.Recipe, error) {
	const q = `SELECT id, product_id, COALESCE(green_bean_id, ''), required_roasted_mg FROM product_recipes WHERE product_id = ? ORDER BY id ASC;`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("list recipes %s: %w", productID, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []domain.Recipe
	for rows.Next() {
		var recipe domain.Recipe
		var required int64
		if err := rows.Scan(&recipe.ID, &recipe.ProductID, &recipe.GreenBeanID, &required); err != nil {
			return nil, fmt.Errorf("scan recipe %s: %w", productID, err)
		}
		recipe.RequiredRoastedMg = domain.WeightMg(required)
		out = append(out, recipe)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes %s: %w", productID, err)
	}
	return out, nil
}

// productScanner abstracts row scanning for products.
type productScanner interface {
	// Scan copies column values into destinations.
	Scan(dest ...any) error
}

// scanProduct converts the current row into a Product.
func scanProduct(row productScanner) (*domain.Product, error) {
	var product domain.Product
	var price int64
	var active int
	if err := row.Scan(&product.ID, &product.Name, &product.Category, &price, &active); err != nil {
		return nil, err
	}
	product.Price = domain.MoneyIDR(price)
	product.IsActive = active == 1
	return &product, nil
}
