INSERT INTO green_beans (id, name, origin, process, stock_mg, cost_per_kg) VALUES
    ('GB-GAYO-WASHED', 'Gayo Washed', 'Aceh Gayo', 'Washed', 5000000, 180000),
    ('GB-LINTONG-NATURAL', 'Lintong Natural', 'Lintong', 'Natural', 3000000, 150000);

INSERT INTO products (id, name, category, price, is_active) VALUES
    ('P-LATTE-HOT', 'Hot Latte', 'DRINK', 25000, 1),
    ('P-BEANS-250', 'Beans 250g', 'BEAN_RETAIL', 85000, 1),
    ('P-BEANS-1KG', 'Beans 1kg Wholesale', 'BEAN_WHOLESALE', 300000, 1);

INSERT INTO product_recipes (product_id, green_bean_id, required_roasted_mg) VALUES
    ('P-LATTE-HOT', 'GB-GAYO-WASHED', 18000),
    ('P-BEANS-250', 'GB-GAYO-WASHED', 250000),
    ('P-BEANS-1KG', 'GB-GAYO-WASHED', 1000000);
