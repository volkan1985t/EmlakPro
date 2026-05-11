package model

import "time"

type Role string
const (
	RoleAdmin Role = "admin"
	RoleAgent Role = "agent"
)

type User struct {
	ID           int64     `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	Role         Role      `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Listing struct {
	ID         int64             `json:"id" db:"id"`
	ListingNo  int64             `json:"listing_no" db:"listing_no"`
	UserID     int64             `json:"user_id" db:"user_id"`
	ShareToken string            `json:"share_token" db:"share_token"`
	IsActive   bool              `json:"is_active" db:"is_active"`
	CoverImage string            `json:"cover_image" db:"cover_image"`
	Fields     map[string]string `json:"fields" db:"fields"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
	OwnerName  string            `json:"owner_name,omitempty" db:"owner_name"`
	Images     []ListingImage    `json:"images,omitempty"`
}

func (l *Listing) Field(key string) string {
	if l.Fields == nil { return "" }
	return l.Fields[key]
}

type ListingImage struct {
	ID        int64     `json:"id" db:"id"`
	ListingID int64     `json:"listing_id" db:"listing_id"`
	Path      string    `json:"path" db:"path"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Request struct {
	ID         int64             `json:"id" db:"id"`
	UserID     int64             `json:"user_id" db:"user_id"`
	IsActive   bool              `json:"is_active" db:"is_active"`
	NotifyMe   bool              `json:"notify_me" db:"notify_me"`
	Fields     map[string]string `json:"fields" db:"fields"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
	OwnerName  string            `json:"owner_name,omitempty" db:"owner_name"`
	MatchCount int               `json:"match_count,omitempty"`
}

func (r *Request) Field(key string) string {
	if r.Fields == nil { return "" }
	return r.Fields[key]
}

type RefreshToken struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ── API ──────────────────────────────────────────────────────
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     Role   `json:"role"`
}
type CreateListingRequest struct {
	Fields     map[string]string `json:"fields"`
	CoverImage string            `json:"cover_image"`
	Images     []string          `json:"images"`
}
type UpdateListingRequest struct {
	Fields       map[string]string `json:"fields"`
	CoverImage   string            `json:"cover_image"`
	Images       []string          `json:"images"`
	RemoveImages []int64           `json:"remove_images"`
}
type CreateRequestPayload struct {
	Fields   map[string]string `json:"fields"`
	NotifyMe bool              `json:"notify_me"`
}
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
func OK(data interface{}) APIResponse  { return APIResponse{Success: true, Data: data} }
func Err(msg string) APIResponse       { return APIResponse{Success: false, Error: msg} }

type PublicConfig struct {
	AppName       string              `json:"app_name"`
	PropertyTypes []string            `json:"property_types"`
	ListingTypes  []string            `json:"listing_types"`
	Districts     []string            `json:"districts"`
	Neighborhoods []string            `json:"neighborhoods"`
	ListingFields interface{}         `json:"listing_fields"`
	RequestFields interface{}         `json:"request_fields"`
	FieldSources  map[string][]string `json:"field_sources"`
}
