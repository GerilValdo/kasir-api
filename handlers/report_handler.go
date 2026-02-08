package handlers

import (
	"encoding/json"
	"kasir-api/services"
	"net/http"
)

type ReportHandler struct {
	service *services.ReportService
}

func NewReportHandler(service *services.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) HandleReport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetSalesSummary(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ReportHandler) GetSalesSummary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	startDate := query.Get("start_date")
	endDate := query.Get("end_date")

	var start, end *string
	if startDate != "" && endDate != "" {
		start = &startDate
		end = &endDate
	}

	result, err := h.service.GetSalesSummary(start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
