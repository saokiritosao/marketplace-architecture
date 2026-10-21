package httpapi

import (
	"context"

	openapiTypes "github.com/oapi-codegen/runtime/types"

	"marketplace/internal/api/generated"
	"marketplace/internal/domain"
	"marketplace/internal/requestctx"
	"marketplace/internal/security"
)

func mustActor(ctx context.Context) requestctx.Actor {
	actor, _ := requestctx.ActorFrom(ctx)
	return actor
}

func tokenResponse(pair security.Pair) generated.TokenPairResponse {
	return generated.TokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    generated.TokenPairResponseTokenType("Bearer"),
		ExpiresIn:    pair.ExpiresIn,
	}
}

func userResponse(user domain.User) generated.UserResponse {
	return generated.UserResponse{
		ID:        user.ID,
		Email:     openapiTypes.Email(user.Email),
		Role:      generated.UserRole(user.Role),
		CreatedAt: user.CreatedAt,
	}
}

func productResponse(product domain.Product) generated.ProductResponse {
	return generated.ProductResponse{
		ID:          product.ID,
		SellerID:    product.SellerID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price.Float64(),
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      generated.ProductStatus(product.Status),
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}

func productPageResponse(page domain.ProductPage) generated.ProductPage {
	items := make([]generated.ProductResponse, 0, len(page.Items))
	for _, product := range page.Items {
		items = append(items, productResponse(product))
	}
	return generated.ProductPage{
		Items:         items,
		TotalElements: page.Total,
		Page:          page.Page,
		Size:          page.Size,
	}
}

func orderResponse(order domain.Order) generated.OrderResponse {
	items := make([]generated.OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, generated.OrderItemResponse{
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder.Float64(),
			LineAmount:   item.PriceAtOrder.Mul(item.Quantity).Float64(),
		})
	}
	return generated.OrderResponse{
		ID:             order.ID,
		UserID:         order.UserID,
		Status:         generated.OrderStatus(order.Status),
		PromoCode:      order.PromoCode,
		Items:          items,
		SubtotalAmount: order.SubtotalAmount.Float64(),
		DiscountAmount: order.DiscountAmount.Float64(),
		TotalAmount:    order.TotalAmount.Float64(),
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}

func promoResponse(promo domain.PromoCode) generated.PromoCodeResponse {
	return generated.PromoCodeResponse{
		ID:             promo.ID,
		SellerID:       promo.SellerID,
		Code:           promo.Code,
		DiscountType:   generated.PromoDiscountType(promo.DiscountType),
		DiscountValue:  promo.DiscountValue.Float64(),
		MinOrderAmount: promo.MinOrderAmount.Float64(),
		MaxUses:        promo.MaxUses,
		CurrentUses:    promo.CurrentUses,
		ValidFrom:      promo.ValidFrom,
		ValidUntil:     promo.ValidUntil,
		Active:         promo.Active,
	}
}
