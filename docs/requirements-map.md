# Где закрыт каждый пункт задания

Этот файл нужен скорее как личная проверка перед сдачей. Его можно открыть, если
на защите спросят, где именно реализован конкретный критерий.

## 1. OpenAPI CRUD для Product

- контракт: `api/openapi/marketplace.yaml`;
- методы: `POST/GET/PUT/DELETE /products`;
- пагинация и фильтры: `ListProductsParams` и `store.ListProducts`;
- мягкое удаление: `ProductService.Archive` и `store.ArchiveProduct`.

## 2. Схемы данных

В OpenAPI отдельно описаны:

- `ProductCreate`;
- `ProductUpdate`;
- `ProductResponse`;
- `ProductPage`;
- enum `ProductStatus`;
- форматы денежных и временных полей.

## 3. Кодогенерация

- конфигурация: `internal/api/generated/oapi-codegen.yaml`;
- команда: `make generate`;
- generated-файл находится в `.gitignore`;
- Docker запускает генерацию до сборки;
- handler реализует generated `ServerInterface`;
- request и response используют generated-модели.

## 4. PostgreSQL и CRUD

- PostgreSQL: `docker-compose.yml`;
- Flyway: сервис `flyway` и папка `migrations`;
- индекс статуса: `migrations/V2__products.sql`;
- автоматические timestamps: default и trigger;
- SQL CRUD: `internal/store/products.go`.

## 5. Единые ошибки

- схема `ErrorResponse` и список `ErrorCode` находятся в OpenAPI;
- внутренний тип ошибки: `internal/apperror/error.go`;
- преобразование в HTTP: `internal/httpapi/json.go` и
  `internal/middleware/response.go`.

## 6. Контрактная валидация

- ограничения находятся в OpenAPI;
- middleware: `internal/middleware/validation.go`;
- неправильный запрос не попадает в handler/service;
- несколько нарушений собираются в `details`.

## 7. Заказы

- создание, обновление, отмена и переходы: `internal/service/orders.go`;
- SQL заказов: `internal/store/orders.go`;
- блокировка товаров: `store.LockProducts` с `FOR UPDATE`;
- вся сложная операция проходит в одной транзакции;
- цена сделки сохраняется в `price_at_order`;
- остатки и использование промокода возвращаются при отмене.

## 8. JSON-логи

- `internal/middleware/logging.go`;
- request ID в логе и `X-Request-Id`;
- status и duration берутся через `responseRecorder`;
- пароли и токены маскируются;
- тест маскирования: `logging_test.go`.

## 9. JWT

- создание и проверка токенов: `internal/security/tokens.go`;
- `access` и `refresh` различаются claim `token_type`;
- middleware access token: `internal/middleware/auth.go`;
- endpoints регистрации, входа и refresh: `internal/httpapi/auth.go`.

## 10. Роли

- роли: `internal/domain/models.go`;
- роль записывается в JWT;
- проверки находятся в service, потому что там известен владелец ресурса;
- `seller_id` есть в таблице `products`;
- ограничения ролей также написаны в описаниях endpoint'ов OpenAPI.
