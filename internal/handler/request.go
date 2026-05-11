package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/go-chi/chi/v5"
)

type RequestHandler struct {
	cfg         *config.Config
	requestRepo *repository.RequestRepository
}

func NewRequestHandler(cfg *config.Config, requestRepo *repository.RequestRepository) *RequestHandler {
	return &RequestHandler{cfg: cfg, requestRepo: requestRepo}
}

// GET /api/requests
func (h *RequestHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	f := repository.RequestFilter{
		ListingType:  r.URL.Query().Get("listing_type"),
		PropertyType: r.URL.Query().Get("property_type"),
		District:     r.URL.Query().Get("district"),
		Search:       r.URL.Query().Get("q"),
	}
	if !isAdmin {
		f.UserID = userID
	}

	list, err := h.requestRepo.List(f)
	if err != nil {
		jsonErr(w, "Talepler yüklenemedi", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.Request{}
	}
	jsonOK(w, list)
}

// POST /api/requests
func (h *RequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req model.CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	if req.Fields["client_name"] == "" {
		jsonErr(w, "Müşteri adı zorunludur", http.StatusBadRequest)
		return
	}
	if req.Fields["phone"] == "" {
		jsonErr(w, "Telefon zorunludur", http.StatusBadRequest)
		return
	}

	request := &model.Request{
		UserID:   userID,
		NotifyMe: req.NotifyMe,
		Fields:   req.Fields,
		IsActive: true,
	}
	if err := h.requestRepo.Create(request); err != nil {
		jsonErr(w, "Talep oluşturulamadı", http.StatusInternalServerError)
		return
	}
	jsonOK(w, request)
}

// PUT /api/requests/{id}
func (h *RequestHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	existing, err := h.requestRepo.GetByID(id)
	if err != nil || existing == nil {
		jsonErr(w, "Talep bulunamadı", http.StatusNotFound)
		return
	}
	if !isAdmin && existing.UserID != userID {
		jsonErr(w, "Bu talebi düzenleme yetkiniz yok", http.StatusForbidden)
		return
	}

	var req model.CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	existing.Fields = req.Fields
	existing.NotifyMe = req.NotifyMe
	if err := h.requestRepo.Update(existing); err != nil {
		jsonErr(w, "Talep güncellenemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, existing)
}

// PATCH /api/requests/{id}/toggle
func (h *RequestHandler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	if err := h.requestRepo.ToggleActive(id, userID, isAdmin); err != nil {
		jsonErr(w, "Durum değiştirilemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "Durum güncellendi"})
}

// PATCH /api/requests/{id}/notify
func (h *RequestHandler) ToggleNotify(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	userID, _ := middleware.GetUserID(r.Context())

	if err := h.requestRepo.ToggleNotify(id, userID); err != nil {
		jsonErr(w, "Bildirim ayarı değiştirilemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "Bildirim ayarı güncellendi"})
}
