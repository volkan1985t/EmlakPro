package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/volkan1985t/EmlakPro/internal/service"
	"github.com/go-chi/chi/v5"
)

type ListingHandler struct {
	cfg         *config.Config
	listingRepo *repository.ListingRepository
	imageSvc    *service.ImageService
}

func NewListingHandler(cfg *config.Config, repo *repository.ListingRepository, imageSvc *service.ImageService) *ListingHandler {
	return &ListingHandler{cfg: cfg, listingRepo: repo, imageSvc: imageSvc}
}

// GET /api/listings
func (h *ListingHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	onlyMine  := r.URL.Query().Get("only_mine") == "1"

	f := repository.ListFilter{
		UserID:       userID,
		OnlyMine:     onlyMine,
		IsAdmin:      isAdmin,
		ListingType:  r.URL.Query().Get("listing_type"),
		PropertyType: r.URL.Query().Get("property_type"),
		District:     r.URL.Query().Get("district"),
		Rooms:        r.URL.Query().Get("rooms"),
		Search:       r.URL.Query().Get("q"),
	}

	listings, err := h.listingRepo.List(f)
	if err != nil {
		log.Printf("List error: %v", err)
		jsonErr(w, "İlanlar yüklenemedi", http.StatusInternalServerError)
		return
	}
	for i := range listings {
		listings[i].CoverImage = h.imageSvc.PathToURL(listings[i].CoverImage)
		for j := range listings[i].Images {
			listings[i].Images[j].Path = h.imageSvc.PathToURL(listings[i].Images[j].Path)
		}
	}
	if listings == nil { listings = []model.Listing{} }
	jsonOK(w, listings)
}

// GET /api/listings/{id}
func (h *ListingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	listing, err := h.listingRepo.GetByID(id)
	if err != nil || listing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	if !isAdmin && listing.UserID != userID {
		jsonErr(w, "Bu ilana erişim yetkiniz yok", http.StatusForbidden)
		return
	}
	listing.CoverImage = h.imageSvc.PathToURL(listing.CoverImage)
	for i := range listing.Images {
		listing.Images[i].Path = h.imageSvc.PathToURL(listing.Images[i].Path)
	}
	jsonOK(w, listing)
}

// GET /api/listings/share/{token}
func (h *ListingHandler) GetByShareToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	listing, err := h.listingRepo.GetByShareToken(token)
	if err != nil || listing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }
	listing.CoverImage = h.imageSvc.PathToURL(listing.CoverImage)
	for i := range listing.Images {
		listing.Images[i].Path = h.imageSvc.PathToURL(listing.Images[i].Path)
	}
	jsonOK(w, listing)
}

// POST /api/listings
func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var req model.CreateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest); return
	}
	if req.Fields["title"] == "" { jsonErr(w, "Başlık zorunludur", http.StatusBadRequest); return }
	if req.Fields["price"] == ""  { jsonErr(w, "Fiyat zorunludur",  http.StatusBadRequest); return }

	listing := &model.Listing{
		UserID:     userID,
		CoverImage: req.CoverImage,
		Fields:     req.Fields,
		IsActive:   true,
		IsListed:   true,
		Status:     "aktif",
	}
	if err := h.listingRepo.Create(listing); err != nil {
		log.Printf("Create listing error: %v", err)
		jsonErr(w, "İlan oluşturulamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for i, imgPath := range req.Images {
		h.listingRepo.AddImage(listing.ID, imgPath, i)
	}
	h.listingRepo.AddHistory(listing.ID, userID, "created", "aktif", nil)

	full, _ := h.listingRepo.GetByID(listing.ID)
	if full != nil {
		full.CoverImage = h.imageSvc.PathToURL(full.CoverImage)
		for i := range full.Images { full.Images[i].Path = h.imageSvc.PathToURL(full.Images[i].Path) }
		jsonOK(w, full)
	} else {
		jsonOK(w, listing)
	}
}

// PUT /api/listings/{id}
func (h *ListingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	existing, err := h.listingRepo.GetByID(id)
	if err != nil || existing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }
	if !isAdmin && existing.UserID != userID {
		jsonErr(w, "Bu ilanı düzenleme yetkiniz yok", http.StatusForbidden); return
	}

	var req model.UpdateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest); return
	}
	existing.Fields = req.Fields
	if req.CoverImage != "" { existing.CoverImage = req.CoverImage }
	if err := h.listingRepo.Update(existing); err != nil {
		jsonErr(w, "İlan güncellenemedi", http.StatusInternalServerError); return
	}
	for _, imgID := range req.RemoveImages {
		path, err := h.listingRepo.DeleteImage(imgID, id)
		if err == nil { h.imageSvc.DeleteImage(path) }
	}
	for i, imgPath := range req.Images {
		h.listingRepo.AddImage(id, imgPath, i+100)
	}
	h.listingRepo.AddHistory(id, userID, "updated", existing.Status, nil)

	full, _ := h.listingRepo.GetByID(id)
	if full != nil {
		full.CoverImage = h.imageSvc.PathToURL(full.CoverImage)
		for i := range full.Images { full.Images[i].Path = h.imageSvc.PathToURL(full.Images[i].Path) }
		jsonOK(w, full)
	}
}

// PATCH /api/listings/{id}/toggle — aktif/pasif + durum sorusu
func (h *ListingHandler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	existing, err := h.listingRepo.GetByID(id)
	if err != nil || existing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }
	if !isAdmin && existing.UserID != userID {
		jsonErr(w, "Yetkisiz", http.StatusForbidden); return
	}

	var status string
	var closingPrice *int64
	var action string

	if existing.IsActive {
		// Aktif → Pasif: body'den durum oku
		var body model.ToggleActiveRequest
		json.NewDecoder(r.Body).Decode(&body)
		status = body.Status
		if status == "" { status = "bekliyor" }
		closingPrice = body.ClosingPrice
		action = "deactivated"
	} else {
		// Pasif → Aktif: durumu sıfırla
		status = "aktif"
		closingPrice = nil
		action = "activated"
	}

	if err := h.listingRepo.ToggleActive(id, userID, isAdmin, status, closingPrice); err != nil {
		jsonErr(w, "Durum değiştirilemedi", http.StatusInternalServerError); return
	}
	h.listingRepo.AddHistory(id, userID, action, status, closingPrice)
	jsonOK(w, map[string]string{"message": "Durum güncellendi", "status": status})
}

// PATCH /api/listings/{id}/listed — listeleme toggle
func (h *ListingHandler) ToggleListed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	existing, err := h.listingRepo.GetByID(id)
	if err != nil || existing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }
	if !isAdmin && existing.UserID != userID {
		jsonErr(w, "Yetkisiz", http.StatusForbidden); return
	}

	if err := h.listingRepo.ToggleListed(id, userID, isAdmin); err != nil {
		jsonErr(w, "Listeleme durumu değiştirilemedi", http.StatusInternalServerError); return
	}
	action := "listed"
	if existing.IsListed { action = "unlisted" }
	h.listingRepo.AddHistory(id, userID, action, existing.Status, nil)
	jsonOK(w, map[string]string{"message": "Listeleme güncellendi"})
}

// GET /api/listings/{id}/history
func (h *ListingHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil { jsonErr(w, "Geçersiz ID", http.StatusBadRequest); return }

	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	listing, err := h.listingRepo.GetByID(id)
	if err != nil || listing == nil { jsonErr(w, "İlan bulunamadı", http.StatusNotFound); return }
	if !isAdmin && listing.UserID != userID {
		jsonErr(w, "Yetkisiz", http.StatusForbidden); return
	}

	history, err := h.listingRepo.GetHistory(id)
	if err != nil {
		jsonErr(w, "Tarihçe yüklenemedi", http.StatusInternalServerError); return
	}
	if history == nil { history = []model.ListingHistory{} }
	jsonOK(w, history)
}

// DELETE /api/listings/{id}/images/{imgID}
func (h *ListingHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	listingID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	imgID, _     := strconv.ParseInt(chi.URLParam(r, "imgID"), 10, 64)
	userID, _    := middleware.GetUserID(r.Context())
	isAdmin      := middleware.IsAdmin(r.Context())
	if !isAdmin {
		ok, _ := h.listingRepo.IsOwner(listingID, userID)
		if !ok { jsonErr(w, "Yetkisiz", http.StatusForbidden); return }
	}
	path, err := h.listingRepo.DeleteImage(imgID, listingID)
	if err != nil { jsonErr(w, "Resim silinemedi", http.StatusNotFound); return }
	h.imageSvc.DeleteImage(path)
	jsonOK(w, map[string]string{"message": "Resim silindi"})
}
