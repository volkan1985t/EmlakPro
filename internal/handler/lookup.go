package handler

import (
	"net/http"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
)

type LookupHandler struct {
	cfg        *config.Config
	lookupRepo *repository.LookupRepository
}

func NewLookupHandler(cfg *config.Config, lookupRepo *repository.LookupRepository) *LookupHandler {
	return &LookupHandler{cfg: cfg, lookupRepo: lookupRepo}
}

// LookupResult — gelen-arama popup'ı için birleşik salt-okunur sonuç.
type LookupResult struct {
	Found     bool             `json:"found"`
	Phone     string           `json:"phone"`
	Customer  *model.Customer  `json:"customer"`
	Interests []model.Interest `json:"interests"`
	Requests  []model.Request  `json:"requests"`
}

// GET /api/lookup?phone=...
// Yalnızca giriş yapan danışmanın kendi kayıtlarını döner (user_id kapsamlı).
func (h *LookupHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		jsonErr(w, "phone parametresi zorunludur", http.StatusBadRequest)
		return
	}

	cust, err := h.lookupRepo.CustomerByPhone(userID, phone)
	if err != nil {
		jsonErr(w, "Musteri sorgulanamadi", http.StatusInternalServerError)
		return
	}
	interests, err := h.lookupRepo.InterestsByPhone(userID, phone)
	if err != nil {
		jsonErr(w, "Ilgiler sorgulanamadi", http.StatusInternalServerError)
		return
	}
	requests, err := h.lookupRepo.RequestsByPhone(userID, phone)
	if err != nil {
		jsonErr(w, "Talepler sorgulanamadi", http.StatusInternalServerError)
		return
	}

	if interests == nil {
		interests = []model.Interest{}
	}
	if requests == nil {
		requests = []model.Request{}
	}

	res := LookupResult{
		Found:     cust != nil || len(interests) > 0 || len(requests) > 0,
		Phone:     phone,
		Customer:  cust,
		Interests: interests,
		Requests:  requests,
	}
	jsonOK(w, res)
}
