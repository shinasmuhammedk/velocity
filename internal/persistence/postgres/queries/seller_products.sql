-- name: CreateSellerProduct :one
INSERT INTO seller_products (seller_id, name, symbol, status) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListSellerProducts :many
SELECT * FROM seller_products WHERE seller_id = $1;

-- name: GetAllSellerProducts :many
SELECT * FROM seller_products;

-- name: GetSellerStats :one
SELECT 
    COALESCE(SUM(t.price * t.quantity), 0)::BIGINT as total_revenue,
    COALESCE(SUM(t.quantity), 0)::BIGINT as total_products_sold,
    (SELECT COUNT(*) FROM orders o WHERE o.user_id = $1 AND o.side = 'SELL' AND o.status IN ('OPEN', 'PARTIALLY_FILLED'))::BIGINT as active_listings,
    (SELECT COALESCE(SUM(o.price * o.remaining), 0) FROM orders o WHERE o.user_id = $1 AND o.side = 'SELL' AND o.status IN ('OPEN', 'PARTIALLY_FILLED'))::BIGINT as locked_inventory_value
FROM trades t
WHERE t.seller_id = $1;

-- name: GetSellerActivity :many
SELECT 
    t.id::TEXT as id,
    t.executed_at as time,
    'Sold'::TEXT as action,
    COALESCE(sp.name, t.symbol)::TEXT as product,
    t.quantity as amount,
    t.price as price
FROM trades t
LEFT JOIN seller_products sp ON sp.symbol = t.symbol AND sp.seller_id = t.seller_id
WHERE t.seller_id = $1
ORDER BY t.executed_at DESC
LIMIT 10;
