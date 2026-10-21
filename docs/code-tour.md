# Что открывать, если просят показать код

## 1. OpenAPI

Открыть:

```text
api/openapi/marketplace.yaml
```

Показать на примере `POST /products`:

- `operationId` задаёт имя сгенерированного метода;
- `requestBody` ссылается на `ProductCreate`;
- ответы описаны отдельно;
- `security` берётся из общей JWT-схемы;
- ограничения `minLength`, `minimum`, `enum` находятся в контракте.

Потом показать `ProductCreate`, `ProductUpdate` и `ProductResponse`. Это три
разные модели: вход создания, вход полного обновления и ответ сервера.

## 2. Генерация

Открыть:

```text
Makefile
internal/api/generated/oapi-codegen.yaml
```

Команда `make generate` запускает `oapi-codegen`. Сгенерированный файл указан в
`.gitignore` и создаётся при Docker build.

## 3. Запуск

Открыть:

```text
cmd/api/main.go
```

Показать:

- загрузку конфигурации;
- соединение с PostgreSQL;
- создание service'ов;
- подключение сгенерированного router;
- порядок middleware;
- graceful shutdown.

Порядок middleware читается снизу вверх. Снаружи стоит ограничение размера body,
потом request ID и лог, recovery, JWT, OpenAPI validation и handler.

## 4. Обычный CRUD

Открыть по очереди:

```text
internal/httpapi/products.go
internal/service/products.go
internal/store/products.go
```

Так хорошо видно разделение:

- handler читает HTTP DTO;
- service проверяет роль и владельца;
- store выполняет SQL.

## 5. Транзакция заказа

Открыть:

```text
internal/service/orders.go
```

Найти `Create`. Показать:

1. `Begin`;
2. `defer Rollback`;
3. блокировку пользователя;
4. проверку лимита и активного заказа;
5. `LockProducts`;
6. проверку остатков;
7. сохранение заказа и позиций;
8. `Commit`.

Затем открыть:

```text
internal/store/products.go
```

и показать `FOR UPDATE` в `LockProducts`.

## 6. Обновление заказа

В `OrderService.Update` показать, что старые количества сначала прибавляются к
локальному состоянию товаров, потом резервируются новые. В базу всё записывается
только в рамках одной транзакции.

## 7. JWT

Открыть:

```text
internal/security/tokens.go
internal/middleware/auth.go
```

В token manager создаются два типа токенов. Middleware принимает только access
и кладёт пользователя в `context.Context`.

## 8. Логи

Открыть:

```text
internal/middleware/logging.go
```

Показать `responseRecorder`, генерацию request ID, измерение времени и
`maskSensitive`.

## 9. База и миграции

Открыть:

```text
migrations/V2__products.sql
migrations/V4__orders.sql
```

Показать индекс `idx_products_status`, trigger `set_updated_at`, частичный
уникальный индекс активного заказа и типы статусов.

## 10. Ошибки

Открыть:

```text
internal/apperror/error.go
internal/httpapi/json.go
```

Service возвращает понятную ошибку с code/status/details. HTTP-слой переводит
её в единый `ErrorResponse`.
