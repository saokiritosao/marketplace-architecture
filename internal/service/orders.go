package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/money"
	"marketplace/internal/requestctx"
	"marketplace/internal/store"
)

type OrderService struct {
	store     *store.Store
	rateLimit time.Duration
	now       func() time.Time
}

func NewOrderService(store *store.Store, rateLimit time.Duration) *OrderService {
	return &OrderService{
		store:     store,
		rateLimit: rateLimit,
		now:       time.Now,
	}
}

func (s *OrderService) Create(
	ctx context.Context,
	actor requestctx.Actor,
	lines []domain.OrderLine,
	promoCode *string,
) (domain.Order, error) {
	if actor.Role != domain.RoleUser && actor.Role != domain.RoleAdmin {
		return domain.Order{}, apperror.AccessDenied()
	}

	lines, err := normalizeLines(lines)
	if err != nil {
		return domain.Order{}, err
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	defer tx.Rollback(ctx) // после успешного Commit этот Rollback ничего не меняет

	q := s.store.WithTx(tx)
	if err := q.LockUser(ctx, actor.ID); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := s.checkRateLimit(ctx, q, actor.ID, domain.OperationCreateOrder); err != nil {
		return domain.Order{}, err
	}

	hasActive, err := q.HasActiveOrder(ctx, actor.ID)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if hasActive {
		return domain.Order{}, apperror.OrderHasActive()
	}

	products, err := q.LockProducts(ctx, productIDs(lines))
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	items, subtotal, err := reserveProducts(lines, products)
	if err != nil {
		return domain.Order{}, err
	}
	if err := saveStocks(ctx, q, products); err != nil {
		return domain.Order{}, err
	}

	var promo *domain.PromoCode
	var discount money.Cents
	if promoCode != nil {
		code := strings.ToUpper(strings.TrimSpace(*promoCode))
		promo, discount, err = s.applyNewPromo(ctx, q, code, subtotal)
		if err != nil {
			return domain.Order{}, err
		}
	}

	order := domain.Order{
		UserID:         actor.ID,
		Status:         domain.OrderCreated,
		SubtotalAmount: subtotal,
		DiscountAmount: discount,
		TotalAmount:    subtotal - discount,
	}
	if promo != nil {
		order.PromoCodeID = &promo.ID
		order.PromoCode = &promo.Code
	}

	order, err = q.CreateOrder(ctx, order)
	if err != nil {
		if store.IsUniqueViolation(err) {
			return domain.Order{}, apperror.OrderHasActive()
		}
		return domain.Order{}, apperror.Internal(err)
	}

	order.Items = make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		item.OrderID = order.ID
		stored, err := q.AddOrderItem(ctx, item)
		if err != nil {
			return domain.Order{}, apperror.Internal(err)
		}
		order.Items = append(order.Items, stored)
	}

	if err := q.AddOperation(ctx, actor.ID, domain.OperationCreateOrder); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	return order, nil
}

func (s *OrderService) Get(ctx context.Context, actor requestctx.Actor, id int64) (domain.Order, error) {
	if actor.Role == domain.RoleSeller {
		return domain.Order{}, apperror.AccessDenied()
	}

	order, err := s.store.Queries().OrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, apperror.OrderNotFound(id)
	}
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if actor.Role != domain.RoleAdmin && order.UserID != actor.ID {
		return domain.Order{}, apperror.OrderOwnershipViolation()
	}
	return order, nil
}

func (s *OrderService) Update(
	ctx context.Context,
	actor requestctx.Actor,
	id int64,
	lines []domain.OrderLine,
) (domain.Order, error) {
	if actor.Role == domain.RoleSeller {
		return domain.Order{}, apperror.AccessDenied()
	}

	lines, err := normalizeLines(lines)
	if err != nil {
		return domain.Order{}, err
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	q := s.store.WithTx(tx)
	preview, err := q.OrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, apperror.OrderNotFound(id)
	}
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if actor.Role != domain.RoleAdmin && preview.UserID != actor.ID {
		return domain.Order{}, apperror.OrderOwnershipViolation()
	}

	if err := q.LockUser(ctx, preview.UserID); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	order, err := q.LockOrderByID(ctx, id)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if order.Status != domain.OrderCreated {
		return domain.Order{}, apperror.InvalidStateTransition(string(order.Status))
	}
	if err := s.checkRateLimit(ctx, q, order.UserID, domain.OperationUpdateOrder); err != nil {
		return domain.Order{}, err
	}

	allProductIDs := mergeProductIDs(order.Items, lines)
	products, err := q.LockProducts(ctx, allProductIDs)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if len(products) != len(allProductIDs) {
		return domain.Order{}, apperror.ProductNotFound(firstMissingID(allProductIDs, products))
	}

	// Сначала мысленно возвращаем старую бронь, затем проверяем новый состав заказа.
	for _, item := range order.Items {
		product := products[item.ProductID]
		product.Stock += item.Quantity
		products[item.ProductID] = product
	}

	items, subtotal, err := reserveProducts(lines, products)
	if err != nil {
		return domain.Order{}, err
	}
	if err := saveStocks(ctx, q, products); err != nil {
		return domain.Order{}, err
	}

	promoID := order.PromoCodeID
	promoCode := order.PromoCode
	discount := money.Cents(0)
	if promoID != nil {
		promo, err := q.LockPromoByID(ctx, *promoID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, apperror.PromoInvalid()
		}
		if err != nil {
			return domain.Order{}, apperror.Internal(err)
		}

		if subtotal < promo.MinOrderAmount {
			if err := q.ChangePromoUses(ctx, promo.ID, -1); err != nil {
				return domain.Order{}, apperror.Internal(err)
			}
			promoID = nil
			promoCode = nil
		} else {
			if err := validateExistingPromo(promo, s.now()); err != nil {
				return domain.Order{}, err
			}
			discount = calculateDiscount(promo, subtotal)
		}
	}

	storedItems, err := q.ReplaceOrderItems(ctx, order.ID, items)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	updatedAt, err := q.UpdateOrderAmounts(
		ctx,
		order.ID,
		promoID,
		int64(subtotal),
		int64(discount),
		int64(subtotal-discount),
	)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := q.AddOperation(ctx, order.UserID, domain.OperationUpdateOrder); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}

	order.Items = storedItems
	order.PromoCodeID = promoID
	order.PromoCode = promoCode
	order.SubtotalAmount = subtotal
	order.DiscountAmount = discount
	order.TotalAmount = subtotal - discount
	order.UpdatedAt = updatedAt
	return order, nil
}

func (s *OrderService) Cancel(ctx context.Context, actor requestctx.Actor, id int64) (domain.Order, error) {
	if actor.Role == domain.RoleSeller {
		return domain.Order{}, apperror.AccessDenied()
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	q := s.store.WithTx(tx)
	preview, err := q.OrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, apperror.OrderNotFound(id)
	}
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if actor.Role != domain.RoleAdmin && preview.UserID != actor.ID {
		return domain.Order{}, apperror.OrderOwnershipViolation()
	}

	if err := q.LockUser(ctx, preview.UserID); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	order, err := q.LockOrderByID(ctx, id)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if order.Status != domain.OrderCreated && order.Status != domain.OrderPaymentPending {
		return domain.Order{}, apperror.InvalidStateTransition(string(order.Status))
	}

	ids := make([]int64, 0, len(order.Items))
	for _, item := range order.Items {
		ids = append(ids, item.ProductID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	products, err := q.LockProducts(ctx, ids)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if len(products) != len(ids) {
		return domain.Order{}, apperror.Internal(errors.New("an ordered product no longer exists"))
	}
	for _, item := range order.Items {
		product := products[item.ProductID]
		product.Stock += item.Quantity
		products[item.ProductID] = product
	}
	if err := saveStocks(ctx, q, products); err != nil {
		return domain.Order{}, err
	}

	if order.PromoCodeID != nil {
		if _, err := q.LockPromoByID(ctx, *order.PromoCodeID); err != nil {
			return domain.Order{}, apperror.Internal(err)
		}
		if err := q.ChangePromoUses(ctx, *order.PromoCodeID, -1); err != nil {
			return domain.Order{}, apperror.Internal(err)
		}
	}

	updatedAt, err := q.CancelOrder(ctx, order.ID)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}

	order.Status = domain.OrderCanceled
	order.UpdatedAt = updatedAt
	return order, nil
}

func (s *OrderService) applyNewPromo(
	ctx context.Context,
	q *store.Queries,
	code string,
	subtotal money.Cents,
) (*domain.PromoCode, money.Cents, error) {
	promo, err := q.LockPromoByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, apperror.PromoInvalid()
	}
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	if err := validateNewPromo(promo, subtotal, s.now()); err != nil {
		return nil, 0, err
	}
	if err := q.ChangePromoUses(ctx, promo.ID, 1); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	promo.CurrentUses++
	return &promo, calculateDiscount(promo, subtotal), nil
}

func (s *OrderService) checkRateLimit(
	ctx context.Context,
	q *store.Queries,
	userID int64,
	operation domain.OperationType,
) error {
	last, err := q.LastOperation(ctx, userID, operation)
	if err != nil {
		return apperror.Internal(err)
	}
	if last == nil {
		return nil
	}

	remaining := s.rateLimit - s.now().Sub(*last)
	if remaining > 0 {
		return apperror.OrderLimitExceeded(int64(math.Ceil(remaining.Seconds())))
	}
	return nil
}

func normalizeLines(lines []domain.OrderLine) ([]domain.OrderLine, error) {
	if len(lines) == 0 {
		return nil, apperror.Validation(map[string]any{"items": "must contain at least one item"})
	}

	quantities := make(map[int64]int, len(lines))
	for _, line := range lines {
		if line.ProductID <= 0 {
			return nil, apperror.Validation(map[string]any{"product_id": "must be greater than zero"})
		}
		if line.Quantity <= 0 || line.Quantity > 999 {
			return nil, apperror.Validation(map[string]any{"quantity": "must be between 1 and 999"})
		}

		quantities[line.ProductID] += line.Quantity
		if quantities[line.ProductID] > 999 {
			return nil, apperror.Validation(map[string]any{
				"items": "total quantity for one product cannot be greater than 999",
			})
		}
	}

	normalized := make([]domain.OrderLine, 0, len(quantities))
	for productID, quantity := range quantities {
		normalized = append(normalized, domain.OrderLine{ProductID: productID, Quantity: quantity})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].ProductID < normalized[j].ProductID
	})
	return normalized, nil
}

func productIDs(lines []domain.OrderLine) []int64 {
	ids := make([]int64, len(lines))
	for i, line := range lines {
		ids[i] = line.ProductID
	}
	return ids
}

func mergeProductIDs(oldItems []domain.OrderItem, newLines []domain.OrderLine) []int64 {
	set := make(map[int64]struct{}, len(oldItems)+len(newLines))
	for _, item := range oldItems {
		set[item.ProductID] = struct{}{}
	}
	for _, line := range newLines {
		set[line.ProductID] = struct{}{}
	}

	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func reserveProducts(
	lines []domain.OrderLine,
	products map[int64]domain.Product,
) ([]domain.OrderItem, money.Cents, error) {
	for _, line := range lines {
		product, ok := products[line.ProductID]
		if !ok {
			return nil, 0, apperror.ProductNotFound(line.ProductID)
		}
		if product.Status != domain.ProductActive {
			return nil, 0, apperror.ProductInactive(product.ID)
		}
	}

	shortage := make([]map[string]any, 0)
	for _, line := range lines {
		product := products[line.ProductID]
		if product.Stock < line.Quantity {
			shortage = append(shortage, map[string]any{
				"product_id":         product.ID,
				"requested_quantity": line.Quantity,
				"available_stock":    product.Stock,
			})
		}
	}
	if len(shortage) > 0 {
		return nil, 0, apperror.InsufficientStock(shortage)
	}

	items := make([]domain.OrderItem, 0, len(lines))
	var subtotal money.Cents
	for _, line := range lines {
		product := products[line.ProductID]
		product.Stock -= line.Quantity
		products[line.ProductID] = product

		item := domain.OrderItem{
			ProductID:    product.ID,
			Quantity:     line.Quantity,
			PriceAtOrder: product.Price,
		}
		items = append(items, item)
		subtotal += product.Price.Mul(line.Quantity)
	}
	return items, subtotal, nil
}

func saveStocks(ctx context.Context, q *store.Queries, products map[int64]domain.Product) error {
	ids := make([]int64, 0, len(products))
	for id := range products {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		if err := q.SetProductStock(ctx, id, products[id].Stock); err != nil {
			return apperror.Internal(err)
		}
	}
	return nil
}

func firstMissingID(ids []int64, products map[int64]domain.Product) int64 {
	for _, id := range ids {
		if _, ok := products[id]; !ok {
			return id
		}
	}
	return 0
}

func validateNewPromo(promo domain.PromoCode, subtotal money.Cents, now time.Time) error {
	if !promo.Active || promo.CurrentUses >= promo.MaxUses || now.Before(promo.ValidFrom) || now.After(promo.ValidUntil) {
		return apperror.PromoInvalid()
	}
	if subtotal < promo.MinOrderAmount {
		return apperror.PromoMinAmount(promo.MinOrderAmount.Float64(), subtotal.Float64())
	}
	return nil
}

func validateExistingPromo(promo domain.PromoCode, now time.Time) error {
	if !promo.Active || promo.CurrentUses > promo.MaxUses || now.Before(promo.ValidFrom) || now.After(promo.ValidUntil) {
		return apperror.PromoInvalid()
	}
	return nil
}

func calculateDiscount(promo domain.PromoCode, subtotal money.Cents) money.Cents {
	if promo.DiscountType == domain.DiscountFixed {
		return money.Min(promo.DiscountValue, subtotal)
	}

	// Для процентов DiscountValue хранит сотые доли процента: 15.00% == 1500.
	discount := money.Cents(int64(subtotal) * int64(promo.DiscountValue) / 10000)
	maxDiscount := money.Cents(int64(subtotal) * 70 / 100)
	return money.Min(discount, maxDiscount)
}

func (s *OrderService) ChangeStatus(
	ctx context.Context,
	actor requestctx.Actor,
	id int64,
	target domain.OrderStatus,
) (domain.Order, error) {
	if actor.Role != domain.RoleAdmin {
		return domain.Order{}, apperror.AccessDenied()
	}
	if target == domain.OrderCanceled {
		return s.Cancel(ctx, actor, id)
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	q := s.store.WithTx(tx)
	preview, err := q.OrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, apperror.OrderNotFound(id)
	}
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := q.LockUser(ctx, preview.UserID); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}

	order, err := q.LockOrderByID(ctx, id)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if !allowedStatusTransition(order.Status, target) {
		return domain.Order{}, apperror.InvalidStateTransition(string(order.Status))
	}

	updatedAt, err := q.UpdateOrderStatus(ctx, order.ID, target)
	if err != nil {
		return domain.Order{}, apperror.Internal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, apperror.Internal(err)
	}

	order.Status = target
	order.UpdatedAt = updatedAt
	return order, nil
}

func allowedStatusTransition(from, to domain.OrderStatus) bool {
	switch from {
	case domain.OrderCreated:
		return to == domain.OrderPaymentPending
	case domain.OrderPaymentPending:
		return to == domain.OrderPaid
	case domain.OrderPaid:
		return to == domain.OrderShipped
	case domain.OrderShipped:
		return to == domain.OrderCompleted
	default:
		return false
	}
}
