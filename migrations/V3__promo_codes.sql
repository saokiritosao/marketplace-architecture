CREATE TYPE promo_discount_type AS ENUM ('PERCENTAGE', 'FIXED_AMOUNT');

CREATE TABLE promo_codes (
    id               BIGSERIAL PRIMARY KEY,
    seller_id        BIGINT NOT NULL REFERENCES users(id),
    code             VARCHAR(20) NOT NULL UNIQUE,
    discount_type    promo_discount_type NOT NULL,
    discount_value   NUMERIC(12, 2) NOT NULL CHECK (discount_value > 0),
    min_order_amount NUMERIC(12, 2) NOT NULL CHECK (min_order_amount >= 0),
    max_uses         INTEGER NOT NULL CHECK (max_uses > 0),
    current_uses     INTEGER NOT NULL DEFAULT 0 CHECK (current_uses >= 0),
    valid_from       TIMESTAMPTZ NOT NULL,
    valid_until      TIMESTAMPTZ NOT NULL,
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (valid_until > valid_from),
    CHECK (current_uses <= max_uses)
);

CREATE INDEX idx_promo_codes_code ON promo_codes(code);
