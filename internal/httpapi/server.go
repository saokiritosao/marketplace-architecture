package httpapi

import (
	"context"
	"net/http"
	"time"

	"marketplace/internal/api/generated"
	"marketplace/internal/service"
	"marketplace/internal/store"
)

var _ generated.ServerInterface = (*Server)(nil)

type Server struct {
	store    *store.Store
	auth     *service.AuthService
	products *service.ProductService
	orders   *service.OrderService
	promos   *service.PromoService
}

func NewServer(
	store *store.Store,
	auth *service.AuthService,
	products *service.ProductService,
	orders *service.OrderService,
	promos *service.PromoService,
) *Server {
	return &Server{
		store:    store,
		auth:     auth,
		products: products,
		orders:   orders,
		promos:   promos,
	}
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		writeError(w, r, serviceUnavailable(err))
		return
	}
	writeJSON(w, http.StatusOK, generated.HealthResponse{Status: "ok"})
}
