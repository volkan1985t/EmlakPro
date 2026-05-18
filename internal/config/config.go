package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config ana yapısı — config.json ile birebir eşleşir
type Config struct {
	App      AppConfig      `json:"app"`
	Database DatabaseConfig `json:"database"`
	Auth     AuthConfig     `json:"auth"`
	Admin    AdminConfig    `json:"admin"`
	Telegram TelegramConfig `json:"telegram"`

	PropertyTypes  []string `json:"property_types"`
	ListingTypes   []string `json:"listing_types"`
	Districts      []string `json:"districts"`
	Neighborhoods  []string `json:"neighborhoods"`
	RoomOptions    []string `json:"room_options"`
	ZoningOptions  []string `json:"zoning_options"`
	HeatingOptions []string `json:"heating_options"`
	FloorOptions   []string `json:"floor_options"`

	ListingFields ListingFieldsConfig `json:"listing_fields"`
	RequestFields RequestFieldsConfig `json:"request_fields"`
}

type AppConfig struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	Port           string `json:"port"`
	Env            string `json:"env"`
	UploadDir      string `json:"upload_dir"`
	MaxImageWidth  int    `json:"max_image_width"`
	MaxImageHeight int    `json:"max_image_height"`
	ImageQuality   int    `json:"image_quality"`
	MaxGallery     int    `json:"max_gallery_images"`
}

type DatabaseConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Name         string `json:"name"`
	User         string `json:"user"`
	Password     string `json:"password"`
	SSLMode      string `json:"ssl_mode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

type AuthConfig struct {
	JWTSecret            string `json:"jwt_secret"`
	AccessTokenTTLMins   int    `json:"access_token_ttl_minutes"`
	RefreshTokenTTLDays  int    `json:"refresh_token_ttl_days"`
}

type AdminConfig struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type TelegramConfig struct {
	Enabled          bool   `json:"enabled"`
	BotToken         string `json:"bot_token"`
	ChannelID        string `json:"channel_id"`
	NotifyNewListing bool   `json:"notify_new_listing"`
	NotifyNewRequest bool   `json:"notify_new_request"`
}

type FieldDefinition struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Source     string `json:"source,omitempty"`
	Searchable bool   `json:"searchable"`
	AdminOnly  bool   `json:"admin_only,omitempty"`
}

type ListingFieldsConfig struct {
	AllFields     []FieldDefinition       `json:"all_fields"`
	CardFields    map[string][]string     `json:"card_fields"`
	SummaryFields []string                `json:"summary_fields"`
}

type RequestFieldsConfig struct {
	Common     []FieldDefinition       `json:"common"`
	ByProperty map[string][]string     `json:"by_property"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

// Load config.json dosyasını okur
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config dosyası açılamadı: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config parse hatası: %w", err)
	}

	// Zorunlu alanlar kontrolü
	if cfg.Auth.JWTSecret == "" || cfg.Auth.JWTSecret == "CHANGE_ME_JWT_SECRET_MIN_32_CHARS" {
		return nil, fmt.Errorf("auth.jwt_secret ayarlanmamış")
	}
	if cfg.Database.Password == "" || cfg.Database.Password == "CHANGE_ME_STRONG_PASSWORD" {
		return nil, fmt.Errorf("database.password ayarlanmamış")
	}

	// Varsayılanlar
	if cfg.App.MaxImageWidth == 0  { cfg.App.MaxImageWidth = 1920 }
	if cfg.App.MaxImageHeight == 0 { cfg.App.MaxImageHeight = 1080 }
	if cfg.App.ImageQuality == 0   { cfg.App.ImageQuality = 85 }
	if cfg.App.MaxGallery == 0     { cfg.App.MaxGallery = 12 }
	if cfg.Auth.AccessTokenTTLMins == 0  { cfg.Auth.AccessTokenTTLMins = 60 }
	if cfg.Auth.RefreshTokenTTLDays == 0 { cfg.Auth.RefreshTokenTTLDays = 30 }

	return &cfg, nil
}

// Save config'i belirtilen path'e yazar
func Save(path string, cfg *Config) error {
        f, err := os.Create(path)
        if err != nil { return fmt.Errorf("config dosyası yazılamadı: %w", err) }
        defer f.Close()
        enc := json.NewEncoder(f)
        enc.SetIndent("", "    ")
        return enc.Encode(cfg)
}

// FieldByKey belirtilen key'e sahip field tanımını döner
func (c *Config) FieldByKey(key string) *FieldDefinition {
	for _, f := range c.ListingFields.AllFields {
		if f.Key == key {
			return &f
		}
	}
	return nil
}

// NeighborhoodsFor returns the neighborhood list for a district.
// Currently returns all neighborhoods (no per-district mapping in config).
func (c *Config) NeighborhoodsFor(district string) []string {
	return c.Neighborhoods
}

// SourceValues bir field'ın source'una göre seçenek listesini döner
func (c *Config) SourceValues(source string) []string {
	switch source {
	case "property_types":  return c.PropertyTypes
	case "listing_types":   return c.ListingTypes
	case "districts":       return c.Districts
	case "neighborhoods":   return c.Neighborhoods
	case "room_options":    return c.RoomOptions
	case "zoning_options":  return c.ZoningOptions
	case "heating_options": return c.HeatingOptions
	case "floor_options":   return c.FloorOptions
	}
	return nil
}
