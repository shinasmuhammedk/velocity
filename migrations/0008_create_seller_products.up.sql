CREATE TABLE IF NOT EXISTS seller_products (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    status VARCHAR(30) DEFAULT 'Active' NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(seller_id, symbol)
);
