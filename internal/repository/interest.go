package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type InterestRepository struct {
	db *sql.DB
}

func NewInterestRepository(db *sql.DB) *InterestRepository {
	return &InterestRepository{db: db}
}

// nullInt64 — 0 ise NULL döner (listing_id / customer_id için)
func nullInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// nullDate — "" ise NULL, dolu ise "2006-01-02" parse eder
func nullDate(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return nil
}

func (r *InterestRepository) Create(it *model.Interest) error {
	if it.Status == "" {
		it.Status = "yeni"
	}
	return r.db.QueryRow(`
		INSERT INTO interests
			(user_id, listing_id, customer_id, buyer_name, buyer_phone,
			 type, status, outcome, offer_amount, next_step, next_date, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`,
		it.UserID, nullInt64(it.ListingID), nullInt64(it.CustomerID),
		it.BuyerName, it.BuyerPhone, it.Type, it.Status, it.Outcome,
		it.OfferAmount, it.NextStep, nullDate(it.NextDate), it.Notes,
	).Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt)
}

// InterestFilter — listeleme filtreleri
type InterestFilter struct {
	UserID     int64  // sahibi (zorunlu — gizlilik)
	Status     string // opsiyonel
	ListingID  int64  // opsiyonel
	TodayOnly  bool   // sadece bugün/geçmiş aranacaklar
}

func (r *InterestRepository) List(f InterestFilter) ([]model.Interest, error) {
	where := []string{"i.user_id = $1"}
	args := []interface{}{f.UserID}
	n := 2
	if f.Status != "" {
		where = append(where, fmt.Sprintf("i.status = $%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.ListingID > 0 {
		where = append(where, fmt.Sprintf("i.listing_id = $%d", n))
		args = append(args, f.ListingID)
		n++
	}
	if f.TodayOnly {
		where = append(where, "i.next_date IS NOT NULL AND i.next_date <= CURRENT_DATE AND i.status <> 'sonuc'")
	}

	query := fmt.Sprintf(`
		SELECT i.id, i.user_id, COALESCE(i.listing_id,0), COALESCE(i.customer_id,0),
		       COALESCE(i.buyer_name,''), COALESCE(i.buyer_phone,''),
		       i.type, i.status, COALESCE(i.outcome,''), COALESCE(i.offer_amount,''),
		       COALESCE(i.next_step,''), COALESCE(TO_CHAR(i.next_date,'YYYY-MM-DD'),''),
		       COALESCE(i.notes,''), i.created_at, i.updated_at,
		       COALESCE(l.fields->>'title','')
		FROM interests i
		LEFT JOIN listings l ON l.id = i.listing_id
		WHERE %s
		ORDER BY (i.next_date IS NULL), i.next_date ASC, i.updated_at DESC`,
		strings.Join(where, " AND "))

	rows, err := r.db.Query(query, args...)
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

func (r *InterestRepository) GetByID(id int64) (*model.Interest, error) {
	var it model.Interest
	err := r.db.QueryRow(`
		SELECT i.id, i.user_id, COALESCE(i.listing_id,0), COALESCE(i.customer_id,0),
		       COALESCE(i.buyer_name,''), COALESCE(i.buyer_phone,''),
		       i.type, i.status, COALESCE(i.outcome,''), COALESCE(i.offer_amount,''),
		       COALESCE(i.next_step,''), COALESCE(TO_CHAR(i.next_date,'YYYY-MM-DD'),''),
		       COALESCE(i.notes,''), i.created_at, i.updated_at,
		       COALESCE(l.fields->>'title','')
		FROM interests i
		LEFT JOIN listings l ON l.id = i.listing_id
		WHERE i.id = $1`, id,
	).Scan(
		&it.ID, &it.UserID, &it.ListingID, &it.CustomerID,
		&it.BuyerName, &it.BuyerPhone, &it.Type, &it.Status, &it.Outcome,
		&it.OfferAmount, &it.NextStep, &it.NextDate, &it.Notes,
		&it.CreatedAt, &it.UpdatedAt, &it.ListingTitle,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// Update — tüm düzenlenebilir alanları günceller (sahiplik kontrolü çağıran katmanda)
func (r *InterestRepository) Update(it *model.Interest) error {
	_, err := r.db.Exec(`
		UPDATE interests SET
			status=$1, outcome=$2, offer_amount=$3, next_step=$4,
			next_date=$5, notes=$6, type=$7, updated_at=NOW()
		WHERE id=$8 AND user_id=$9`,
		it.Status, it.Outcome, it.OfferAmount, it.NextStep,
		nullDate(it.NextDate), it.Notes, it.Type, it.ID, it.UserID)
	return err
}

func (r *InterestRepository) Delete(id, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM interests WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// IsOwner — kayıt bu kullanıcıya mı ait
func (r *InterestRepository) IsOwner(id, userID int64) bool {
	var n int
	r.db.QueryRow(`SELECT COUNT(*) FROM interests WHERE id=$1 AND user_id=$2`, id, userID).Scan(&n)
	return n > 0
}

// CountByStatus — pipeline kolonları için sayılar
func (r *InterestRepository) CountByStatus(userID int64) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT status, COUNT(*) FROM interests WHERE user_id=$1 GROUP BY status`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var s string
		var c int
		if err := rows.Scan(&s, &c); err != nil {
			return nil, err
		}
		m[s] = c
	}
	return m, rows.Err()
}
