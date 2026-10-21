\pset pager off
\x off

SELECT 'USERS' AS section;
SELECT id, email, role, created_at
FROM users
ORDER BY id;

SELECT 'PRODUCTS' AS section;
SELECT id, seller_id, name, price, stock, category, status, created_at, updated_at
FROM products
ORDER BY id;

SELECT 'PROMO CODES' AS section;
SELECT id, seller_id, code, discount_type, discount_value,
       min_order_amount, max_uses, current_uses, active,
       valid_from, valid_until
FROM promo_codes
ORDER BY id;

SELECT 'ORDERS' AS section;
SELECT id, user_id, status, promo_code_id, subtotal_amount,
       discount_amount, total_amount, created_at, updated_at
FROM orders
ORDER BY id;

SELECT 'ORDER ITEMS' AS section;
SELECT id, order_id, product_id, quantity, price_at_order
FROM order_items
ORDER BY order_id, id;

SELECT 'USER OPERATIONS' AS section;
SELECT id, user_id, operation_type, created_at
FROM user_operations
ORDER BY id;
