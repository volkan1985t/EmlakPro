package handler

import (
	"net/http"
	"strconv"
	"strings"

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

// LookupResult — gelen-arama popup'i icin birlesik salt-okunur sonuc.
// name/summary/count, MacroDroid'in tek alan gosterebilmesi icin hazir gelir.
type LookupResult struct {
	Found     bool             `json:"found"`
	Phone     string           `json:"phone"`
	Name      string           `json:"name"`
	Summary   string           `json:"summary"`
	Count     int              `json:"count"`
	Customer  *model.Customer  `json:"customer"`
	Interests []model.Interest `json:"interests"`
	Requests  []model.Request  `json:"requests"`
}

// GET /api/(sorgu/)lookup?phone=...
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

	name := buildName(cust, interests)

	res := LookupResult{
		Found:     cust != nil || len(interests) > 0 || len(requests) > 0,
		Phone:     phone,
		Name:      name,
		Summary:   buildSummary(name, cust, interests, requests),
		Count:     len(interests) + len(requests),
		Customer:  cust,
		Interests: interests,
		Requests:  requests,
	}
	jsonOK(w, res)
}

func buildName(c *model.Customer, interests []model.Interest) string {
	if c != nil && strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	if len(interests) > 0 && strings.TrimSpace(interests[0].BuyerName) != "" {
		return interests[0].BuyerName
	}
	return ""
}

// buildSummary — popup icin tek satirlik ozet. En yeni ilgi one cikar.
func buildSummary(name string, c *model.Customer, interests []model.Interest, requests []model.Request) string {
	if name == "" && len(interests) == 0 && len(requests) == 0 {
		return ""
	}
	var parts []string
	if name != "" {
		parts = append(parts, name)
	}
	if len(interests) > 0 {
		it := interests[0] // created_at DESC -> en yeni
		seg := it.Type
		if strings.TrimSpace(it.OfferAmount) != "" {
			seg += " " + it.OfferAmount
		}
		if strings.TrimSpace(seg) != "" {
			parts = append(parts, seg)
		}
		if strings.TrimSpace(it.ListingTitle) != "" {
			parts = append(parts, it.ListingTitle)
		}
		if strings.TrimSpace(it.Status) != "" {
			parts = append(parts, it.Status)
		}
	} else if c != nil {
		parts = append(parts, "kayitli musteri")
	}
	if total := len(interests) + len(requests); total > 1 {
		parts = append(parts, "(+"+strconv.Itoa(total-1)+" kayit daha)")
	}
	return strings.Join(parts, " \u00b7 ")
}
