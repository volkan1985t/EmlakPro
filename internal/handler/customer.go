package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/volkan1985t/EmlakPro/internal/service"
	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	cfg          *config.Config
	customerRepo *repository.CustomerRepository
	listingRepo  *repository.ListingRepository
	imageSvc     *service.ImageService
}

func NewCustomerHandler(cfg *config.Config, customerRepo *repository.CustomerRepository, listingRepo *repository.ListingRepository, imageSvc *service.ImageService) *CustomerHandler {
	return &CustomerHandler{cfg: cfg, customerRepo: customerRepo, listingRepo: listingRepo, imageSvc: imageSvc}
}

// GET /api/customers
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	search    := r.URL.Query().Get("q")

	customers, err := h.customerRepo.List(userID, isAdmin, search)
	if err != nil { jsonErr(w, "Müşteriler yüklenemedi", http.StatusInternalServerError); return }
	if customers == nil { customers = []model.Customer{} }
	jsonOK(w, customers)
}

// POST /api/customers
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var req model.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest); return
	}
	if req.Name == "" { jsonErr(w, "Müşteri adı zorunludur", http.StatusBadRequest); return }

	c := &model.Customer{
		UserID: userID,
		Name:   req.Name,
		Phone:  req.Phone,
		Email:  req.Email,
		Source: req.Source,
		Notes:  req.Notes,
	}
	if err := h.customerRepo.Create(c); err != nil {
		jsonErr(w, "Müşteri oluşturulamadı: "+err.Error(), http.StatusInternalServerError); return
	}
	jsonOK(w, c)
}

// PUT /api/customers/{id}
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	if !isAdmin {
		ok, _ := h.customerRepo.IsOwner(id, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}

	var req model.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest); return
	}
	if req.Name == "" { jsonErr(w, "Müşteri adı zorunludur", http.StatusBadRequest); return }

	c := &model.Customer{ID: id, Name: req.Name, Phone: req.Phone, Email: req.Email, Source: req.Source, Notes: req.Notes}
	if err := h.customerRepo.Update(c); err != nil {
		jsonErr(w, "Müşteri güncellenemedi", http.StatusInternalServerError); return
	}
	updated, _ := h.customerRepo.GetByID(id)
	jsonOK(w, updated)
}

// PATCH /api/customers/{id}/toggle
func (h *CustomerHandler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	c, err := h.customerRepo.GetByID(id)
	if err != nil || c == nil { jsonErr(w, "Müşteri bulunamadı", http.StatusNotFound); return }
	if !isAdmin && c.UserID != userID { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }

	if err := h.customerRepo.SetActive(id, !c.IsActive); err != nil {
		jsonErr(w, "Durum değiştirilemedi", http.StatusInternalServerError); return
	}
	jsonOK(w, map[string]string{"message": "Güncellendi"})
}

// DELETE /api/customers/{id}
func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	if !isAdmin {
		ok, _ := h.customerRepo.IsOwner(id, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}
	if err := h.customerRepo.Delete(id); err != nil {
		jsonErr(w, "Müşteri silinemedi", http.StatusInternalServerError); return
	}
	jsonOK(w, map[string]string{"message": "Müşteri silindi"})
}

// GET /api/customers/{id}/listings
func (h *CustomerHandler) GetListings(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	if !isAdmin {
		ok, _ := h.customerRepo.IsOwner(id, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}

	listings, err := h.customerRepo.GetLinkedListings(id)
	if err != nil { jsonErr(w, "İlanlar yüklenemedi", http.StatusInternalServerError); return }
	if listings == nil { listings = []model.Listing{} }
	for i := range listings {
		listings[i].CoverImage = h.imageSvc.PathToURL(listings[i].CoverImage)
	}
	jsonOK(w, listings)
}

// POST /api/customers/{id}/listings
func (h *CustomerHandler) LinkListing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	if !isAdmin {
		ok, _ := h.customerRepo.IsOwner(id, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}

	var req model.LinkListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ListingID == 0 {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest); return
	}
	if err := h.customerRepo.LinkListing(id, req.ListingID, req.Note); err != nil {
		jsonErr(w, "İlan bağlanamadı", http.StatusInternalServerError); return
	}
	jsonOK(w, map[string]string{"message": "İlan bağlandı"})
}

// DELETE /api/customers/{id}/listings/{listingID}
func (h *CustomerHandler) UnlinkListing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }
	listingID, err := strconv.ParseInt(chi.URLParam(r, "listingID"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz İlan ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	if !isAdmin {
		ok, _ := h.customerRepo.IsOwner(id, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}
	if err := h.customerRepo.UnlinkListing(id, listingID); err != nil {
		jsonErr(w, "İlan bağlantısı kaldırılamadı", http.StatusInternalServerError); return
	}
	jsonOK(w, map[string]string{"message": "Bağlantı kaldırıldı"})
}
