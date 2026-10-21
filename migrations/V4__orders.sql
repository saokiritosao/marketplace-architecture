CREATE TYPE order_status AS ENUM (
    'CREATED',
    'PAYMENT_PENDING',
    'PAID',
    'SHIPPED',
    'COMPLETED',
    'CANCELED'
);

CREATE TYPE operation_type AS ENUM ('CREATE_ORDER', 'UPDATE_ORDER');

CREATE TABLE orders (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id),
    status           order_status NOT NULL,
    promo_code_id    BIGINT REFERENCES promo_codes(id),
    subtotal_amount  NUMERIC(12, 2) NOT NULL CHECK (subtotal_amount >= 0),
    discount_amount  NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    total_amount     NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id             BIGSERIAL PRIMARY KEY,
    order_id       BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id     BIGINT NOT NULL REFERENCES products(id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    price_at_order NUMERIC(12, 2) NOT NULL CHECK (price_at_order > 0),
    UNIQUE (order_id, product_id)
);

CREATE TABLE user_operations (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id),
    operation_type operation_type NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_user_operations_lookup
    ON user_operations(user_id, operation_type, created_at DESC);

-- В один момент у пользователя может быть только один незавершённый заказ.
CREATE UNIQUE INDEX uq_orders_one_active_per_user
    ON orders(user_id)
    WHERE status IN ('CREATED', 'PAYMENT_PENDING');

CREATE TRIGGER orders_set_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
