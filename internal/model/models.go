package model

import "time"

type Role string
const (
	RoleAdmin Role = "admin"
	RoleAgent Role = "agent"
)

type User struct {
	ID             int64     `json:"id" db:"id"`
	Username       string    `json:"username" db:"username"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	FullName       string    `json:"full_name" db:"full_name"`
	Role           Role      `json:"role" db:"role"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	TelegramChatID   string    `json:"telegram_chat_id,omitempty" db:"telegram_chat_id"`
	TelegramUsername string    `json:"telegram_username,omitempty" db:"telegram_username"`
	NotifyTelegram bool      `json:"notify_telegram" db:"notify_telegram"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Listing struct {
	ID           int64             `json:"id" db:"id"`
	ListingNo    int64             `json:"listing_no" db:"listing_no"`
	UserID       int64             `json:"user_id" db:"user_id"`
	ShareToken   string            `json:"share_token" db:"share_token"`
	IsActive     bool              `json:"is_active" db:"is_active"`
	IsListed     bool              `json:"is_listed" db:"is_listed"`
	Status       string            `json:"status" db:"status"`
	PipelineStage string            `json:"pipeline_stage" db:"pipeline_stage"`
	ClosingPrice *int64            `json:"closing_price,omitempty" db:"closing_price"`
	CustomerID   int64             `json:"customer_id,omitempty" db:"customer_id"`
	CoverImage   string            `json:"cover_image" db:"cover_image"`
	Fields       map[string]string `json:"fields" db:"fields"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
	OwnerName    string            `json:"owner_name,omitempty" db:"owner_name"`
	Images       []ListingImage    `json:"images,omitempty"`
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

type ListingHistory struct {
	ID           int64     `json:"id"`
	ListingID    int64     `json:"listing_id"`
	UserID       int64     `json:"user_id"`
	Action       string    `json:"action"`
	Status       string    `json:"status"`
	ClosingPrice *int64    `json:"closing_price,omitempty"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	UserName     string    `json:"user_name"`
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

type Customer struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	Notes     string    `json:"notes"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	OwnerName string    `json:"owner_name,omitempty"`
	// Linked listings (populated on demand)
	Listings []Listing `json:"listings,omitempty"`
}

type CustomerListing struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	ListingID  int64     `json:"listing_id"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type RefreshToken struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ── Dashboard ────────────────────────────────────────────────
type DashboardStats struct {
	TotalListings    int                      `json:"total_listings"`
	ActiveListings   int                      `json:"active_listings"`
	PassiveListings  int                      `json:"passive_listings"`
	ListedListings   int                      `json:"listed_listings"`
	UnlistedListings int                      `json:"unlisted_listings"`
	ByStatus         map[string]int           `json:"by_status"`
	ByType           map[string]int           `json:"by_type"`
	ByDistrict       []DistrictCount          `json:"by_district"`
	MonthlyAdded     []MonthlyCount           `json:"monthly_added"`
	MonthlyClosed    []MonthlyCount           `json:"monthly_closed"`
	TopAgents        []AgentCount             `json:"top_agents"`
}

type DistrictCount struct {
	District string `json:"district"`
	Count    int    `json:"count"`
}

type MonthlyCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type AgentCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
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
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	FullName       string `json:"full_name"`
	Role           Role   `json:"role"`
	TelegramChatID string `json:"telegram_chat_id"`
}
type CreateListingRequest struct {
	Fields     map[string]string `json:"fields"`
	CoverImage string            `json:"cover_image"`
	Images     []string          `json:"images"`
	CustomerID  int64             `json:"customer_id"`
}
type UpdateListingRequest struct {
	Fields       map[string]string `json:"fields"`
	CoverImage   string            `json:"cover_image"`
	Images       []string          `json:"images"`
	RemoveImages []int64           `json:"remove_images"`
	CustomerID   int64             `json:"customer_id"`
}
type ToggleActiveRequest struct {
	Status       string `json:"status"`
	ClosingPrice *int64 `json:"closing_price"`
}
type CreateRequestPayload struct {
	Fields   map[string]string `json:"fields"`
	NotifyMe bool              `json:"notify_me"`
}
type CreateCustomerRequest struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
	Source string `json:"source"`
	Notes  string `json:"notes"`
}
type LinkListingRequest struct {
	ListingID int64  `json:"listing_id"`
	Note      string `json:"note"`
}
// ── Tasks ────────────────────────────────────────────────────
type Task struct {
	ID          int64      `json:"id"`
	ParentID    *int64     `json:"parent_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedBy   int64      `json:"created_by"`
	CreatorName string     `json:"creator_name,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Assignees   []TaskUser    `json:"assignees,omitempty"`
	Comments    []TaskComment `json:"comments,omitempty"`
	Images      []TaskImage   `json:"images,omitempty"`
	Subtasks    []Task        `json:"subtasks,omitempty"`
}

type TaskUser struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskImage struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	Path       string    `json:"path"`
	UploadedBy int64     `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateTaskRequest struct {
	ParentID    *int64     `json:"parent_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	Assignees   []int64    `json:"assignees"`
}

type UpdateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	Assignees   []int64    `json:"assignees"`
}

type TaskFilter struct {
	Status   string
	Priority string
	UserID   int64
	ParentID *int64
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
	ListingChannels   interface{}         `json:"listing_channels"`
	AutoTaskTemplates interface{}         `json:"auto_task_templates"`
}
