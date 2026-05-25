package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
)

type ConfigHandler struct {
	cfg     *config.Config
	cfgPath string
}

func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg, cfgPath: "/opt/emlakpro/config/config.json"}
}

// GET /api/config
func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	sources := map[string][]string{
		"property_types":  h.cfg.PropertyTypes,
		"listing_types":   h.cfg.ListingTypes,
		"districts":       h.cfg.Districts,
		"neighborhoods":   h.cfg.Neighborhoods,
		"room_options":    h.cfg.RoomOptions,
		"zoning_options":  h.cfg.ZoningOptions,
		"heating_options": h.cfg.HeatingOptions,
	}
	for k, v := range h.cfg.CustomLists {
		sources[k] = v
	}
	jsonOK(w, model.PublicConfig{
		AppName:       h.cfg.App.Name,
		PropertyTypes: h.cfg.PropertyTypes,
		ListingTypes:  h.cfg.ListingTypes,
		Districts:     h.cfg.Districts,
		Neighborhoods: h.cfg.Neighborhoods,
		ListingFields: h.cfg.ListingFields,
		RequestFields: h.cfg.RequestFields,
		FieldSources:      sources,
		ListingChannels:   h.cfg.ListingChannels,
		AutoTaskTemplates: h.cfg.AutoTaskTemplates,
		CustomLists:       h.cfg.CustomLists,
	})
}

// GET /api/admin/settings
func (h *ConfigHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"property_types":  h.cfg.PropertyTypes,
		"listing_types":   h.cfg.ListingTypes,
		"districts":       h.cfg.Districts,
		"neighborhoods":   h.cfg.Neighborhoods,
		"room_options":    h.cfg.RoomOptions,
		"zoning_options":  h.cfg.ZoningOptions,
		"heating_options": h.cfg.HeatingOptions,
		"floor_options":   h.cfg.FloorOptions,
		"request_fields":  h.cfg.RequestFields,
		"card_fields":     h.cfg.ListingFields.CardFields,
		"summary_fields":  h.cfg.ListingFields.SummaryFields,
		"all_fields":       h.cfg.ListingFields.AllFields,
	})
}

// PUT /api/admin/settings
func (h *ConfigHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PropertyTypes  []string                 `json:"property_types"`
		ListingTypes   []string                 `json:"listing_types"`
		Districts      []string                 `json:"districts"`
		Neighborhoods  []string                 `json:"neighborhoods"`
		RoomOptions    []string                 `json:"room_options"`
		ZoningOptions  []string                 `json:"zoning_options"`
		HeatingOptions []string                 `json:"heating_options"`
		FloorOptions   []string                 `json:"floor_options"`
		CardFields     map[string][]string      `json:"card_fields"`
		SummaryFields  []string                 `json:"summary_fields"`
		AllFields      []config.FieldDefinition `json:"all_fields"`
		RequestCommon  []config.FieldDefinition `json:"request_common"`
		RequestByProp  map[string][]string      `json:"request_by_property"`
		AutoTaskTemplates []config.AutoTaskTemplate  `json:"auto_task_templates"`
		CustomLists       map[string][]string        `json:"custom_lists"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	if body.PropertyTypes  != nil { h.cfg.PropertyTypes  = body.PropertyTypes  }
	if body.ListingTypes   != nil { h.cfg.ListingTypes   = body.ListingTypes   }
	if body.Districts      != nil { h.cfg.Districts      = body.Districts      }
	if body.Neighborhoods  != nil { h.cfg.Neighborhoods  = body.Neighborhoods  }
	if body.RoomOptions    != nil { h.cfg.RoomOptions    = body.RoomOptions    }
	if body.ZoningOptions  != nil { h.cfg.ZoningOptions  = body.ZoningOptions  }
	if body.HeatingOptions != nil { h.cfg.HeatingOptions = body.HeatingOptions }
	if body.FloorOptions   != nil { h.cfg.FloorOptions   = body.FloorOptions   }
	if body.CardFields     != nil { h.cfg.ListingFields.CardFields    = body.CardFields    }
	if body.SummaryFields  != nil { h.cfg.ListingFields.SummaryFields = body.SummaryFields }
	if body.RequestCommon  != nil { h.cfg.RequestFields.Common        = body.RequestCommon }
	if body.RequestByProp  != nil { h.cfg.RequestFields.ByProperty    = body.RequestByProp }
	if body.AllFields      != nil { h.cfg.ListingFields.AllFields     = body.AllFields      }
	if body.AutoTaskTemplates != nil { h.cfg.AutoTaskTemplates = body.AutoTaskTemplates }
	if body.CustomLists       != nil { h.cfg.CustomLists       = body.CustomLists }
	if body.AutoTaskTemplates != nil { h.cfg.AutoTaskTemplates        = body.AutoTaskTemplates }

	f, err := os.Create(h.cfgPath)
	if err != nil {
		jsonErr(w, "Config açılamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	if err := enc.Encode(h.cfg); err != nil {
		jsonErr(w, "Config yazılamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "Ayarlar kaydedildi"})
}

// ── JSON yardımcıları ─────────────────────────────────────────────────────────
func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.OK(data))
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(model.Err(msg))
}
