package httpapi

import (
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/domain"
)

func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var body generated.OrderCreate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	order, err := s.orders.Create(r.Context(), mustActor(r.Context()), orderLines(body.Items), body.PromoCode)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, orderResponse(order))
}

func (s *Server) GetOrder(w http.ResponseWriter, r *http.Request, id int64) {
	order, err := s.orders.Get(r.Context(), mustActor(r.Context()), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderResponse(order))
}

func (s *Server) UpdateOrder(w http.ResponseWriter, r *http.Request, id int64) {
	var body generated.OrderUpdate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	order, err := s.orders.Update(r.Context(), mustActor(r.Context()), id, orderLines(body.Items))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderResponse(order))
}

func (s *Server) CancelOrder(w http.ResponseWriter, r *http.Request, id int64) {
	order, err := s.orders.Cancel(r.Context(), mustActor(r.Context()), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderResponse(order))
}

func (s *Server) UpdateOrderStatus(w http.ResponseWriter, r *http.Request, id int64) {
	var body generated.OrderStatusUpdate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	order, err := s.orders.ChangeStatus(
		r.Context(),
		mustActor(r.Context()),
		id,
		domain.OrderStatus(body.Status),
	)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderResponse(order))
}

func orderLines(items []generated.OrderItemInput) []domain.OrderLine {
	lines := make([]domain.OrderLine, 0, len(items))
	for _, item := range items {
		lines = append(lines, domain.OrderLine{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}
	return lines
}
