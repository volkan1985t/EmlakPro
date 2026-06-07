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

type InterestHandler struct {
	cfg          *config.Config
	interestRepo *repository.InterestRepository
}

func NewInterestHandler(cfg *config.Config, interestRepo *repository.InterestRepository) *InterestHandler {
	return &InterestHandler{cfg: cfg, interestRepo: interestRepo}
}

// GET /api/interests?status=&listing_id=&today=1
func (h *InterestHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	f := repository.InterestFilter{UserID: userID}
	f.Status = r.URL.Query().Get("status")
	if lid := r.URL.Query().Get("listing_id"); lid != "" {
		f.ListingID, _ = strconv.ParseInt(lid, 10, 64)
	}
	if r.URL.Query().Get("today") == "1" {
		f.TodayOnly = true
	}
	items, err := h.interestRepo.List(f)
	if err != nil {
		jsonErr(w, "İlgiler yüklenemedi", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []model.Interest{}
	}
	jsonOK(w, items)
}

// GET /api/interests/counts
func (h *InterestHandler) Counts(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	m, err := h.interestRepo.CountByStatus(userID)
	if err != nil {
		jsonErr(w, "Sayılar yüklenemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}

// POST /api/interests
func (h *InterestHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var it model.Interest
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	if it.Type == "" {
		jsonErr(w, "İlgi tipi zorunludur", http.StatusBadRequest)
		return
	}
	it.UserID = userID
	if err := h.interestRepo.Create(&it); err != nil {
		jsonErr(w, "İlgi kaydedilemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, it)
}

// PUT /api/interests/{id}
func (h *InterestHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if !h.interestRepo.IsOwner(id, userID) {
		jsonErr(w, "Bu kayıt size ait değil", http.StatusForbidden)
		return
	}
	var it model.Interest
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	it.ID = id
	it.UserID = userID
	if err := h.interestRepo.Update(&it); err != nil {
		jsonErr(w, "Güncellenemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, it)
}

// DELETE /api/interests/{id}
func (h *InterestHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if !h.interestRepo.IsOwner(id, userID) {
		jsonErr(w, "Bu kayıt size ait değil", http.StatusForbidden)
		return
	}
	if err := h.interestRepo.Delete(id, userID); err != nil {
		jsonErr(w, "Silinemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"deleted": true})
}
