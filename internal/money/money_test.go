package money

import (
	"errors"
	"testing"
)

func TestFromFloat(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		want    Cents
		wantErr bool
	}{
		{name: "whole number", value: 10, want: 1000},
		{name: "two decimals", value: 2499.90, want: 249990},
		{name: "one decimal", value: 15.5, want: 1550},
		{name: "too many decimals", value: 1.234, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromFloat(tt.value)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidAmount) {
					t.Fatalf("expected ErrInvalidAmount, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d cents, want %d", got, tt.want)
			}
		})
	}
}

func TestCentsMul(t *testing.T) {
	if got := Cents(1299).Mul(3); got != 3897 {
		t.Fatalf("got %d, want 3897", got)
	}
}
