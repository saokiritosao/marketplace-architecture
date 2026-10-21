package middleware

import "testing"

func TestMaskSensitive(t *testing.T) {
	input := map[string]any{
		"email":    "user@example.com",
		"password": "secret",
		"nested": map[string]any{
			"refresh_token": "token",
		},
	}

	masked := maskSensitive(input).(map[string]any)
	if masked["password"] != "***" {
		t.Fatalf("password was not masked")
	}
	nested := masked["nested"].(map[string]any)
	if nested["refresh_token"] != "***" {
		t.Fatalf("refresh token was not masked")
	}
	if masked["email"] != "user@example.com" {
		t.Fatalf("ordinary fields must stay unchanged")
	}
}
