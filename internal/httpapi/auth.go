package httpapi

import (
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/domain"
)

func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var body generated.RegisterRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	user, err := s.auth.Register(r.Context(), string(body.Email), body.Password, domain.Role(body.Role))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, userResponse(user))
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var body generated.LoginRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	pair, err := s.auth.Login(r.Context(), string(body.Email), body.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse(pair))
}

func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body generated.RefreshTokenRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	pair, err := s.auth.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse(pair))
}
