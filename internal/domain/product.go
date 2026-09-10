package domain

// Product category identifiers.
const (
	// CategoryDrink represents a brewed beverage consuming roasted grams per cup.
	CategoryDrink = "DRINK"
	// CategoryBeanRetail represents a retail bean pack, for example 250g or 500g.
	CategoryBeanRetail = "BEAN_RETAIL"
	// CategoryBeanWholesale represents a wholesale kilo pack for business buyers.
	CategoryBeanWholesale = "BEAN_WHOLESALE"
)

// Product represents a sellable catalogue item.
//
// A drink consumes roasted coffee through its recipes. A retail or wholesale
// pack maps to a fixed roasted gram amount.
type Product struct {
	// ID is the unique identifier of the product.
	ID string
	// Name is the display name of the product.
	Name string
	// Category is one of DRINK, BEAN_RETAIL, or BEAN_WHOLESALE.
	Category string
	// Price is the selling price in Rupiah.
	Price MoneyIDR
	// IsActive indicates whether the product is currently sellable.
	IsActive bool
}

// Recipe represents a bill-of-materials line for a product.
//
// It maps a product to the roasted grams required from a specific green bean
// lineage. A product may have multiple recipe lines.
type Recipe struct {
	// ID is the surrogate identifier of the recipe line.
	ID int64
	// ProductID references the owning [Product].
	ProductID string
	// GreenBeanID references the required [GreenBean] lineage, if specific.
	GreenBeanID string
	// RequiredRoastedMg is the roasted coffee consumed per product unit in milligrams.
	RequiredRoastedMg WeightMg
}
