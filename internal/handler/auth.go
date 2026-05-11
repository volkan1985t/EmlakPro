package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/auth"
	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	cfg      *config.Config
	userRepo *repository.UserRepository
	tokenSvc *auth.TokenService
}

func NewAuthHandler(cfg *config.Config, userRepo *repository.UserRepository, tokenSvc *auth.TokenService) *AuthHandler {
	return &AuthHandler{cfg: cfg, userRepo: userRepo, tokenSvc: tokenSvc}
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil || user == nil {
		jsonErr(w, "Kullanıcı adı veya şifre hatalı", http.StatusUnauthorized)
		return
	}
	if !user.IsActive {
		jsonErr(w, "Hesabınız devre dışı", http.StatusForbidden)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		jsonErr(w, "Kullanıcı adı veya şifre hatalı", http.StatusUnauthorized)
		return
	}

	// Access token
	accessToken, err := h.tokenSvc.GenerateAccessToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		jsonErr(w, "Token oluşturulamadı", http.StatusInternalServerError)
		return
	}

	// Refresh token
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(h.tokenSvc.RefreshTokenTTL())
	if err := h.userRepo.SaveRefreshToken(user.ID, refreshToken, expiresAt); err != nil {
		jsonErr(w, "Oturum kaydedilemedi", http.StatusInternalServerError)
		return
	}

	// Refresh token cookie olarak da set et
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/api/auth",
		SameSite: http.SameSiteStrictMode,
	})

	jsonOK(w, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	})
}

// POST /api/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Cookie'den de bakabilir
	if req.RefreshToken == "" {
		if c, err := r.Cookie("refresh_token"); err == nil {
			req.RefreshToken = c.Value
		}
	}
	if req.RefreshToken == "" {
		jsonErr(w, "Refresh token gerekli", http.StatusBadRequest)
		return
	}

	rt, err := h.userRepo.GetRefreshToken(req.RefreshToken)
	if err != nil || rt == nil {
		jsonErr(w, "Geçersiz veya süresi dolmuş token", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByID(rt.UserID)
	if err != nil || user == nil || !user.IsActive {
		jsonErr(w, "Kullanıcı bulunamadı", http.StatusUnauthorized)
		return
	}

	// Eski token'ı sil
	h.userRepo.DeleteRefreshToken(req.RefreshToken)

	// Yeni tokenlar
	accessToken, _ := h.tokenSvc.GenerateAccessToken(user.ID, user.Username, string(user.Role))
	newRefresh := uuid.New().String()
	expiresAt := time.Now().Add(h.tokenSvc.RefreshTokenTTL())
	h.userRepo.SaveRefreshToken(user.ID, newRefresh, expiresAt)

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefresh,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/api/auth",
		SameSite: http.SameSiteStrictMode,
	})

	jsonOK(w, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		User:         *user,
	})
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Cookie'den refresh token al
	if c, err := r.Cookie("refresh_token"); err == nil {
		h.userRepo.DeleteRefreshToken(c.Value)
	}
	// Body'den de bakabilir
	var req model.RefreshRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		h.userRepo.DeleteRefreshToken(req.RefreshToken)
	}

	// Cookie'yi temizle
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/api/auth",
	})
	jsonOK(w, map[string]string{"message": "Çıkış yapıldı"})
}

// GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		jsonErr(w, "Yetkisiz", http.StatusUnauthorized)
		return
	}
	user, err := h.userRepo.GetByID(userID)
	if err != nil || user == nil {
		jsonErr(w, "Kullanıcı bulunamadı", http.StatusNotFound)
		return
	}
	user.PasswordHash = ""
	jsonOK(w, user)
}
