package repository

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

// LookupRepository — gelen-arama / numara sorgusu için salt-okunur birleşik sorgular.
// Bir danışmanın (userID) yalnızca KENDİ kayıtlarında telefon numarasına göre arar.
type LookupRepository struct {
	db *sql.DB
}

func NewLookupRepository(db *sql.DB) *LookupRepository {
	return &LookupRepository{db: db}
}

// phoneLast10 — string içindeki rakamları alıp son 10 hanesini döner.
// +90 / 0 / boşluklu formatların hepsi aynı sonuca iner.
func phoneLast10(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			b.WriteByte(byte(ch))
		}
	}
	d := b.String()
	if len(d) > 10 {
		return d[len(d)-10:]
	}
	return d
}

// CustomerByPhone — numaraya eşleşen müşteriyi döner (yoksa nil, hata değil).
func (r *LookupRepository) CustomerByPhone(userID int64, phone string) (*model.Customer, error) {
	p := phoneLast10(phone)
	if p == "" {
		return nil, nil
	}
	var c model.Customer
	err := r.db.QueryRow(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, c.source,
		       COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at
		FROM customers c
		WHERE c.user_id = $1
		  AND RIGHT(regexp_replace(COALESCE(c.phone,''), '[^0-9]', '', 'g'), 10) = $2
		ORDER BY c.updated_at DESC
		LIMIT 1`, userID, p,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Email, &c.Source,
		&c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, nil // sql.ErrNoRows dahil → eşleşme yok
	}
	return &c, nil
}

// InterestsByPhone — numaraya (buyer_phone) eşleşen ilgileri döner, yeniden eskiye.
func (r *LookupRepository) InterestsByPhone(userID int64, phone string) ([]model.Interest, error) {
	p := phoneLast10(phone)
	if p == "" {
		return nil, nil
	}
	rows, err := r.db.Query(`
		SELECT i.id, i.user_id, COALESCE(i.listing_id,0), COALESCE(i.customer_id,0),
		       COALESCE(i.buyer_name,''), COALESCE(i.buyer_phone,''),
		       i.type, i.status, COALESCE(i.outcome,''), COALESCE(i.offer_amount,''),
		       COALESCE(i.next_step,''), COALESCE(TO_CHAR(i.next_date,'YYYY-MM-DD'),''),
		       COALESCE(i.notes,''), i.created_at, i.updated_at,
		       COALESCE(l.fields->>'title','')
		FROM interests i
		LEFT JOIN listings l ON l.id = i.listing_id
		WHERE i.user_id = $1
		  AND RIGHT(regexp_replace(COALESCE(i.buyer_phone,''), '[^0-9]', '', 'g'), 10) = $2
		ORDER BY i.created_at DESC`, userID, p)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Interest
	for rows.Next() {
		var it model.Interest
		if err := rows.Scan(
			&it.ID, &it.UserID, &it.ListingID, &it.CustomerID,
			&it.BuyerName, &it.BuyerPhone, &it.Type, &it.Status, &it.Outcome,
			&it.OfferAmount, &it.NextStep, &it.NextDate, &it.Notes,
			&it.CreatedAt, &it.UpdatedAt, &it.ListingTitle,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// RequestsByPhone — fields->>'client_phone' numaraya eşleşen talepleri döner.
func (r *LookupRepository) RequestsByPhone(userID int64, phone string) ([]model.Request, error) {
	p := phoneLast10(phone)
	if p == "" {
		return nil, nil
	}
	rows, err := r.db.Query(`
		SELECT r.id, r.user_id, r.is_active, r.notify_me,
		       r.fields, r.created_at, r.updated_at
		FROM requests r
		WHERE r.user_id = $1
		  AND RIGHT(regexp_replace(COALESCE(r.fields->>'client_phone',''), '[^0-9]', '', 'g'), 10) = $2
		ORDER BY r.created_at DESC`, userID, p)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Request
	for rows.Next() {
		var req model.Request
		var fieldsJSON []byte
		if err := rows.Scan(&req.ID, &req.UserID, &req.IsActive, &req.NotifyMe,
			&fieldsJSON, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(fieldsJSON, &req.Fields); err != nil {
			req.Fields = map[string]string{}
		}
		out = append(out, req)
	}
	return out, rows.Err()
}
