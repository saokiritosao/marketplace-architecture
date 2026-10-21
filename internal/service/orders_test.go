package service

import (
	"testing"

	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/money"
)

func TestNormalizeLinesMergesDuplicates(t *testing.T) {
	got, err := normalizeLines([]domain.OrderLine{
		{ProductID: 8, Quantity: 2},
		{ProductID: 3, Quantity: 1},
		{ProductID: 8, Quantity: 4},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []domain.OrderLine{
		{ProductID: 3, Quantity: 1},
		{ProductID: 8, Quantity: 6},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNormalizeLinesRejectsTooLargeCombinedQuantity(t *testing.T) {
	_, err := normalizeLines([]domain.OrderLine{
		{ProductID: 1, Quantity: 600},
		{ProductID: 1, Quantity: 500},
	})
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("got %v, want VALIDATION_ERROR", err)
	}
}

func TestPercentageDiscountIsCappedAtSeventyPercent(t *testing.T) {
	promo := domain.PromoCode{
		DiscountType:  domain.DiscountPercentage,
		DiscountValue: money.Cents(9500), // 95.00%
	}
	got := calculateDiscount(promo, money.Cents(10000))
	if got != 7000 {
		t.Fatalf("got %d, want 7000", got)
	}
}

func TestAllowedStatusTransition(t *testing.T) {
	tests := []struct {
		from domain.OrderStatus
		to   domain.OrderStatus
		want bool
	}{
		{domain.OrderCreated, domain.OrderPaymentPending, true},
		{domain.OrderPaymentPending, domain.OrderPaid, true},
		{domain.OrderPaid, domain.OrderShipped, true},
		{domain.OrderShipped, domain.OrderCompleted, true},
		{domain.OrderCreated, domain.OrderPaid, false},
		{domain.OrderCompleted, domain.OrderCreated, false},
	}

	for _, tt := range tests {
		if got := allowedStatusTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("%s -> %s: got %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
