CREATE TABLE IF NOT EXISTS green_beans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    origin TEXT NOT NULL,
    process TEXT NOT NULL,
    stock_mg INTEGER NOT NULL DEFAULT 0,
    cost_per_kg INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roast_batches (
    id TEXT PRIMARY KEY,
    green_bean_id TEXT NOT NULL,
    green_weight_mg INTEGER NOT NULL,
    roasted_weight_mg INTEGER NOT NULL,
    remaining_mg INTEGER NOT NULL,
    shrinkage_pct REAL NOT NULL,
    roast_level TEXT NOT NULL,
    roasted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price INTEGER NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS product_recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    green_bean_id TEXT,
    required_roasted_mg INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    order_type TEXT NOT NULL,
    customer_name TEXT,
    total_amount INTEGER NOT NULL,
    paid_amount INTEGER NOT NULL,
    payment_method TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    subtotal INTEGER NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE IF NOT EXISTS batch_deductions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    roast_batch_id TEXT NOT NULL,
    deducted_mg INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (roast_batch_id) REFERENCES roast_batches(id)
);

CREATE INDEX IF NOT EXISTS idx_roast_batches_bean_time ON roast_batches(green_bean_id, roasted_at ASC);
CREATE INDEX IF NOT EXISTS idx_product_recipes_product ON product_recipes(product_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_batch_deductions_order ON batch_deductions(order_id);
