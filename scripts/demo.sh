#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
RUN_ID=$(date +%s)
SELLER_EMAIL="seller${RUN_ID}@example.com"
BUYER_EMAIL="buyer${RUN_ID}@example.com"
SECOND_BUYER_EMAIL="buyer2_${RUN_ID}@example.com"
PASSWORD="password123"
PROMO_CODE="SALE${RUN_ID}"

LAST_BODY=""
LAST_STATUS=""

wait_for_api() {
    for _ in $(seq 1 30); do
        if curl -fsS "$BASE_URL/health" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done

    echo "API did not become ready at $BASE_URL" >&2
    return 1
}

request() {
    local method=$1
    local path=$2
    local token=${3:-}
    local body=${4:-}
    local args=(-sS -X "$method" "$BASE_URL$path" -H "Content-Type: application/json")

    if [[ -n "$token" ]]; then
        args+=(-H "Authorization: Bearer $token")
    fi
    if [[ -n "$body" ]]; then
        args+=(-d "$body")
    fi

    local response
    response=$(curl "${args[@]}" -w $'\n%{http_code}')
    LAST_STATUS=${response##*$'\n'}
    LAST_BODY=${response%$'\n'*}

    printf '\n%s %s -> HTTP %s\n' "$method" "$path" "$LAST_STATUS"
    if [[ -n "$LAST_BODY" ]]; then
        python3 -m json.tool <<<"$LAST_BODY" 2>/dev/null || printf '%s\n' "$LAST_BODY"
    fi
}

json_value() {
    local key=$1
    python3 -c "import json,sys; print(json.load(sys.stdin)['$key'])" <<<"$LAST_BODY"
}

echo "== Waiting for API =="
wait_for_api

echo "== Health =="
request GET /health

echo "== Register seller and buyer =="
request POST /auth/register "" "{\"email\":\"$SELLER_EMAIL\",\"password\":\"$PASSWORD\",\"role\":\"SELLER\"}"
request POST /auth/register "" "{\"email\":\"$BUYER_EMAIL\",\"password\":\"$PASSWORD\",\"role\":\"USER\"}"

echo "== Login =="
request POST /auth/login "" "{\"email\":\"$SELLER_EMAIL\",\"password\":\"$PASSWORD\"}"
SELLER_TOKEN=$(json_value access_token)
request POST /auth/login "" "{\"email\":\"$BUYER_EMAIL\",\"password\":\"$PASSWORD\"}"
BUYER_TOKEN=$(json_value access_token)
request POST /auth/login "" '{"email":"admin@marketplace.local","password":"admin123"}'
ADMIN_TOKEN=$(json_value access_token)

echo "== Seller creates a product =="
request POST /products "$SELLER_TOKEN" '{
  "name":"Wireless keyboard",
  "description":"Simple keyboard for the demo",
  "price":2499.90,
  "stock":10,
  "category":"Electronics",
  "status":"ACTIVE"
}'
PRODUCT_ID=$(json_value id)

echo "== Pagination and filter =="
request GET "/products?page=0&size=5&status=ACTIVE&category=Electronics" "$BUYER_TOKEN"

echo "== Seller creates a promo code =="
VALID_FROM=$(python3 -c 'from datetime import datetime,timezone,timedelta; print((datetime.now(timezone.utc)-timedelta(minutes=1)).isoformat().replace("+00:00","Z"))')
VALID_UNTIL=$(python3 -c 'from datetime import datetime,timezone,timedelta; print((datetime.now(timezone.utc)+timedelta(days=7)).isoformat().replace("+00:00","Z"))')
request POST /promo-codes "$SELLER_TOKEN" "{
  \"code\":\"$PROMO_CODE\",
  \"discount_type\":\"PERCENTAGE\",
  \"discount_value\":15.00,
  \"min_order_amount\":1000.00,
  \"max_uses\":10,
  \"valid_from\":\"$VALID_FROM\",
  \"valid_until\":\"$VALID_UNTIL\",
  \"active\":true
}"

echo "== Buyer creates, updates and cancels an order =="
request POST /orders "$BUYER_TOKEN" "{
  \"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":2}],
  \"promo_code\":\"$PROMO_CODE\"
}"
ORDER_ID=$(json_value id)
request PUT "/orders/$ORDER_ID" "$BUYER_TOKEN" "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":3}]}"
request POST "/orders/$ORDER_ID/cancel" "$BUYER_TOKEN"

echo "== Expected errors =="
echo "Buyer cannot create products:"
request POST /products "$BUYER_TOKEN" '{"name":"No access","price":10.00,"stock":1,"category":"Test","status":"ACTIVE"}'

echo "OpenAPI validation rejects a negative price before service code:"
request POST /products "$SELLER_TOKEN" '{"name":"Bad price","price":-1,"stock":1,"category":"Test","status":"ACTIVE"}'

echo "Repeated order creation is rate limited:"
request POST /orders "$BUYER_TOKEN" "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}"

echo "== State machine with another buyer =="
request POST /auth/register "" "{\"email\":\"$SECOND_BUYER_EMAIL\",\"password\":\"$PASSWORD\",\"role\":\"USER\"}"
request POST /auth/login "" "{\"email\":\"$SECOND_BUYER_EMAIL\",\"password\":\"$PASSWORD\"}"
SECOND_BUYER_TOKEN=$(json_value access_token)
request POST /orders "$SECOND_BUYER_TOKEN" "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}"
SECOND_ORDER_ID=$(json_value id)

request PUT "/orders/$SECOND_ORDER_ID/status" "$ADMIN_TOKEN" '{"status":"PAYMENT_PENDING"}'
echo "Skipping PAID and going straight to SHIPPED must fail:"
request PUT "/orders/$SECOND_ORDER_ID/status" "$ADMIN_TOKEN" '{"status":"SHIPPED"}'
request PUT "/orders/$SECOND_ORDER_ID/status" "$ADMIN_TOKEN" '{"status":"PAID"}'
request PUT "/orders/$SECOND_ORDER_ID/status" "$ADMIN_TOKEN" '{"status":"SHIPPED"}'
request PUT "/orders/$SECOND_ORDER_ID/status" "$ADMIN_TOKEN" '{"status":"COMPLETED"}'

echo
echo "Demo finished. To inspect PostgreSQL run:"
echo "  make db"
