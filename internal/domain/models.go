package domain

import (
	"time"

	"marketplace/internal/money"
)

type Role string

const (
	RoleUser   Role = "USER"
	RoleSeller Role = "SELLER"
	RoleAdmin  Role = "ADMIN"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

type ProductStatus string

const (
	ProductActive   ProductStatus = "ACTIVE"
	ProductInactive ProductStatus = "INACTIVE"
	ProductArchived ProductStatus = "ARCHIVED"
)

type Product struct {
	ID          int64
	SellerID    int64
	Name        string
	Description *string
	Price       money.Cents
	Stock       int
	Category    string
	Status      ProductStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProductDraft struct {
	Name        string
	Description *string
	Price       money.Cents
	Stock       int
	Category    string
	Status      ProductStatus
}

type ProductFilter struct {
	Page     int
	Size     int
	Status   *ProductStatus
	Category *string
}

type ProductPage struct {
	Items []Product
	Total int64
	Page  int
	Size  int
}

type DiscountType string

const (
	DiscountPercentage DiscountType = "PERCENTAGE"
	DiscountFixed      DiscountType = "FIXED_AMOUNT"
)

type PromoCode struct {
	ID             int64
	SellerID       int64
	Code           string
	DiscountType   DiscountType
	DiscountValue  money.Cents
	MinOrderAmount money.Cents
	MaxUses        int
	CurrentUses    int
	ValidFrom      time.Time
	ValidUntil     time.Time
	Active         bool
	CreatedAt      time.Time
}

type PromoDraft struct {
	Code           string
	DiscountType   DiscountType
	DiscountValue  money.Cents
	MinOrderAmount money.Cents
	MaxUses        int
	ValidFrom      time.Time
	ValidUntil     time.Time
	Active         bool
}

type OrderStatus string

const (
	OrderCreated        OrderStatus = "CREATED"
	OrderPaymentPending OrderStatus = "PAYMENT_PENDING"
	OrderPaid           OrderStatus = "PAID"
	OrderShipped        OrderStatus = "SHIPPED"
	OrderCompleted      OrderStatus = "COMPLETED"
	OrderCanceled       OrderStatus = "CANCELED"
)

type OrderItem struct {
	ID           int64
	OrderID      int64
	ProductID    int64
	Quantity     int
	PriceAtOrder money.Cents
}

type OrderLine struct {
	ProductID int64
	Quantity  int
}

type Order struct {
	ID             int64
	UserID         int64
	Status         OrderStatus
	PromoCodeID    *int64
	PromoCode      *string
	SubtotalAmount money.Cents
	DiscountAmount money.Cents
	TotalAmount    money.Cents
	Items          []OrderItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OperationType string

const (
	OperationCreateOrder OperationType = "CREATE_ORDER"
	OperationUpdateOrder OperationType = "UPDATE_ORDER"
)
