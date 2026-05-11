package handler

import (
	"encoding/json"
	"net/http"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
)

type ConfigHandler struct {
	cfg *config.Config
}

func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg}
}

// GET /api/config — frontend'e güvenli config gönderir (şifre yok)
func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	sources := map[string][]string{
		"property_types":  h.cfg.PropertyTypes,
		"listing_types":   h.cfg.ListingTypes,
		"districts":       h.cfg.Districts,
		"neighborhoods":   h.cfg.Neighborhoods,
		"room_options":    h.cfg.RoomOptions,
		"zoning_options":  h.cfg.ZoningOptions,
		"heating_options": h.cfg.HeatingOptions,
		"floor_options":   h.cfg.FloorOptions,
	}

	jsonOK(w, model.PublicConfig{
		AppName:       h.cfg.App.Name,
		PropertyTypes: h.cfg.PropertyTypes,
		ListingTypes:  h.cfg.ListingTypes,
		Districts:     h.cfg.Districts,
		Neighborhoods: h.cfg.Neighborhoods,
		ListingFields: h.cfg.ListingFields,
		RequestFields: h.cfg.RequestFields,
		FieldSources:  sources,
	})
}

// ── JSON yardımcıları (tüm handler'lar kullanır) ─────────────────────────────

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.OK(data))
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(model.Err(msg))
}
