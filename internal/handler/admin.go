package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	cfg         *config.Config
	userRepo    *repository.UserRepository
	listingRepo *repository.ListingRepository
	requestRepo *repository.RequestRepository
}

func NewAdminHandler(cfg *config.Config, userRepo *repository.UserRepository,
	listingRepo *repository.ListingRepository, requestRepo *repository.RequestRepository) *AdminHandler {
	return &AdminHandler{cfg: cfg, userRepo: userRepo, listingRepo: listingRepo, requestRepo: requestRepo}
}

// ── Kullanıcı Yönetimi ────────────────────────────────────────────────────────

// GET /api/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List()
	if err != nil {
		jsonErr(w, "Kullanıcılar yüklenemedi", http.StatusInternalServerError)
		return
	}
	// Şifre hash'lerini temizle
	for i := range users {
		users[i].PasswordHash = ""
	}
	jsonOK(w, users)
}

// POST /api/admin/users
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		jsonErr(w, "Kullanıcı adı ve şifre zorunludur", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		jsonErr(w, "Şifre en az 6 karakter olmalıdır", http.StatusBadRequest)
		return
	}

	// Duplicate kontrol
	exists, _ := h.userRepo.ExistsByUsername(req.Username)
	if exists {
		jsonErr(w, "Bu kullanıcı adı zaten kullanılıyor", http.StatusConflict)
		return
	}
	if req.Email != "" {
		exists, _ = h.userRepo.ExistsByEmail(req.Email)
		if exists {
			jsonErr(w, "Bu e-posta zaten kullanılıyor", http.StatusConflict)
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "Şifre işlenemedi", http.StatusInternalServerError)
		return
	}

	role := req.Role
	if role == "" {
		role = model.RoleAgent
	}
	// Güvenlik: API ile admin oluşturulamaz
	if role == model.RoleAdmin {
		jsonErr(w, "API ile admin kullanıcısı oluşturulamaz", http.StatusForbidden)
		return
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         role,
		IsActive:     true,
	}
	if err := h.userRepo.Create(user); err != nil {
		jsonErr(w, "Kullanıcı oluşturulamadı", http.StatusInternalServerError)
		return
	}
	user.PasswordHash = ""
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, user)
}

// PATCH /api/admin/users/{id}/toggle
func (h *AdminHandler) ToggleUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	if err := h.userRepo.SetActive(id, true); err != nil {
		jsonErr(w, "Durum değiştirilemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "Kullanıcı durumu güncellendi"})
}

// DELETE /api/admin/users/{id}
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	if err := h.userRepo.Delete(id); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"message": "Kullanıcı silindi"})
}

// ── İlan Yönetimi (Admin) ─────────────────────────────────────────────────────

// GET /api/admin/listings
func (h *AdminHandler) AllListings(w http.ResponseWriter, r *http.Request) {
	listings, err := h.listingRepo.List(repository.ListFilter{})
	if err != nil {
		jsonErr(w, "İlanlar yüklenemedi", http.StatusInternalServerError)
		return
	}
	if listings == nil {
		listings = []model.Listing{}
	}
	jsonOK(w, listings)
}

// DELETE /api/admin/listings/{id}
func (h *AdminHandler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	if err := h.listingRepo.Delete(id); err != nil {
		jsonErr(w, "İlan silinemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "İlan silindi"})
}

// ── Talep Yönetimi (Admin) ────────────────────────────────────────────────────

// GET /api/admin/requests
func (h *AdminHandler) AllRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.requestRepo.List(repository.RequestFilter{})
	if err != nil {
		jsonErr(w, "Talepler yüklenemedi", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.Request{}
	}
	jsonOK(w, list)
}

// DELETE /api/admin/requests/{id}
func (h *AdminHandler) DeleteRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	if err := h.requestRepo.Delete(id); err != nil {
		jsonErr(w, "Talep silinemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "Talep silindi"})
}
