package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type ListingRepository struct {
	db *sql.DB
}

func NewListingRepository(db *sql.DB) *ListingRepository {
	return &ListingRepository{db: db}
}

type ListFilter struct {
	UserID       int64
	OnlyMine     bool
	ListingType  string
	PropertyType string
	District     string
	Rooms        string
	Search       string
	IsAdmin      bool
}

const listingSelectCols = `
	l.id, l.listing_no, l.user_id, l.share_token, l.is_active, l.is_listed,
	l.status, l.closing_price, l.cover_image, l.fields, l.created_at, l.updated_at,
	u.full_name as owner_name, COALESCE(l.customer_id, 0) as customer_id`

func scanListing(row interface {
	Scan(...interface{}) error
}) (*model.Listing, error) {
	var l model.Listing
	var fieldsJSON []byte
	var token string
	err := row.Scan(
		&l.ID, &l.ListingNo, &l.UserID, &token, &l.IsActive, &l.IsListed,
		&l.Status, &l.ClosingPrice, &l.CoverImage, &fieldsJSON,
		&l.CreatedAt, &l.UpdatedAt, &l.OwnerName, &l.CustomerID,
	)
	if err != nil { return nil, err }
	l.ShareToken = token
	if err := json.Unmarshal(fieldsJSON, &l.Fields); err != nil {
		l.Fields = map[string]string{}
	}
	return &l, nil
}

func (r *ListingRepository) List(f ListFilter) ([]model.Listing, error) {
	args  := []interface{}{}
	where := []string{}
	i := 1

	if f.OnlyMine && f.UserID > 0 {
		where = append(where, fmt.Sprintf("l.user_id = $%d", i))
		args = append(args, f.UserID); i++
	} else if f.IsAdmin {
		// Admin hepsini görür, kısıt yok
	} else {
		// Normal: aktif+listelenen ilanlar + sahibine ait tüm ilanlar
		if f.UserID > 0 {
			where = append(where, fmt.Sprintf("((l.is_active = true AND l.is_listed = true) OR l.user_id = $%d)", i))
			args = append(args, f.UserID); i++
		} else {
			where = append(where, "l.is_active = true AND l.is_listed = true")
		}
	}

	if f.ListingType != "" {
		where = append(where, fmt.Sprintf("l.fields->>'listing_type' = $%d", i))
		args = append(args, f.ListingType); i++
	}
	if f.PropertyType != "" {
		where = append(where, fmt.Sprintf("l.fields->>'property_type' = $%d", i))
		args = append(args, f.PropertyType); i++
	}
	if f.District != "" {
		where = append(where, fmt.Sprintf("l.fields->>'district' = $%d", i))
		args = append(args, f.District); i++
	}
	if f.Rooms != "" {
		where = append(where, fmt.Sprintf("l.fields->>'rooms' = $%d", i))
		args = append(args, f.Rooms); i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(l.fields->>'title' ILIKE $%d OR l.fields->>'district' ILIKE $%d OR l.listing_no::text ILIKE $%d)",
			i, i, i))
		args = append(args, "%"+f.Search+"%"); i++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM listings l
		JOIN users u ON u.id = l.user_id
		%s
		ORDER BY l.created_at DESC`, listingSelectCols, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var listings []model.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil { return nil, err }
		l.Images, _ = r.getImages(l.ID)
		listings = append(listings, *l)
	}
	return listings, nil
}

func (r *ListingRepository) GetByID(id int64) (*model.Listing, error) {
	row := r.db.QueryRow(fmt.Sprintf(`
		SELECT %s
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE l.id = $1`, listingSelectCols), id)
	l, err := scanListing(row)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	l.Images, _ = r.getImages(l.ID)
	return l, nil
}

func (r *ListingRepository) GetByShareToken(token string) (*model.Listing, error) {
	row := r.db.QueryRow(fmt.Sprintf(`
		SELECT %s
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE l.share_token = $1 AND l.is_active = true`, listingSelectCols), token)
	l, err := scanListing(row)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	l.Images, _ = r.getImages(l.ID)
	return l, nil
}

func (r *ListingRepository) Create(l *model.Listing) error {
	fieldsJSON, err := json.Marshal(l.Fields)
	if err != nil { return err }
	return r.db.QueryRow(`
		INSERT INTO listings (user_id, cover_image, fields, customer_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id, listing_no, share_token, is_listed, status, created_at, updated_at`,
		l.UserID, l.CoverImage, fieldsJSON, l.CustomerID,
	).Scan(&l.ID, &l.ListingNo, &l.ShareToken, &l.IsListed, &l.Status, &l.CreatedAt, &l.UpdatedAt)
}

func (r *ListingRepository) Update(l *model.Listing) error {
	fieldsJSON, err := json.Marshal(l.Fields)
	if err != nil { return err }
	if l.CustomerID > 0 {
		_, err = r.db.Exec(`UPDATE listings SET cover_image=$1, fields=$2, customer_id=$3, updated_at=NOW() WHERE id=$4`,
			l.CoverImage, fieldsJSON, l.CustomerID, l.ID)
	} else {
		_, err = r.db.Exec(`UPDATE listings SET cover_image=$1, fields=$2, updated_at=NOW() WHERE id=$3`,
			l.CoverImage, fieldsJSON, l.ID)
	}
	return err
}

// ToggleActive — status ve closing_price pasife alınırken set edilir, aktife alınırken reset
func (r *ListingRepository) ToggleActive(id, userID int64, isAdmin bool, status string, closingPrice *int64) error {
	var err error
	if isAdmin {
		_, err = r.db.Exec(`
			UPDATE listings
			SET is_active = NOT is_active, status = $2, closing_price = $3, updated_at = NOW()
			WHERE id = $1`, id, status, closingPrice)
	} else {
		_, err = r.db.Exec(`
			UPDATE listings
			SET is_active = NOT is_active, status = $2, closing_price = $3, updated_at = NOW()
			WHERE id = $1 AND user_id = $4`, id, status, closingPrice, userID)
	}
	return err
}

// ToggleListed — anasayfada görünürlük (is_listed)
func (r *ListingRepository) ToggleListed(id, userID int64, isAdmin bool) error {
	var err error
	if isAdmin {
		_, err = r.db.Exec(`
			UPDATE listings SET is_listed = NOT is_listed, updated_at = NOW() WHERE id = $1`, id)
	} else {
		_, err = r.db.Exec(`
			UPDATE listings SET is_listed = NOT is_listed, updated_at = NOW()
			WHERE id = $1 AND user_id = $2`, id, userID)
	}
	return err
}

func (r *ListingRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM listings WHERE id=$1`, id)
	return err
}

func (r *ListingRepository) IsOwner(listingID, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM listings WHERE id=$1 AND user_id=$2`, listingID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *ListingRepository) AddImage(listingID int64, path string, order int) (*model.ListingImage, error) {
	img := &model.ListingImage{}
	err := r.db.QueryRow(`
		INSERT INTO listing_images (listing_id, path, sort_order)
		VALUES ($1,$2,$3) RETURNING id, listing_id, path, sort_order, created_at`,
		listingID, path, order,
	).Scan(&img.ID, &img.ListingID, &img.Path, &img.SortOrder, &img.CreatedAt)
	return img, err
}

func (r *ListingRepository) DeleteImage(imageID, listingID int64) (string, error) {
	var path string
	err := r.db.QueryRow(
		`DELETE FROM listing_images WHERE id=$1 AND listing_id=$2 RETURNING path`,
		imageID, listingID,
	).Scan(&path)
	if err == sql.ErrNoRows { return "", fmt.Errorf("resim bulunamadı") }
	return path, err
}

func (r *ListingRepository) getImages(listingID int64) ([]model.ListingImage, error) {
	rows, err := r.db.Query(`
		SELECT id, listing_id, path, sort_order, created_at
		FROM listing_images WHERE listing_id=$1 ORDER BY sort_order`, listingID)
	if err != nil { return nil, err }
	defer rows.Close()
	var images []model.ListingImage
	for rows.Next() {
		var img model.ListingImage
		if err := rows.Scan(&img.ID, &img.ListingID, &img.Path, &img.SortOrder, &img.CreatedAt); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

// ── Tarihçe ──────────────────────────────────────────────────

func (r *ListingRepository) AddHistory(listingID, userID int64, action, status string, closingPrice *int64) {
	r.db.Exec(`
		INSERT INTO listing_history (listing_id, user_id, action, status, closing_price)
		VALUES ($1, $2, $3, $4, $5)`,
		listingID, userID, action, status, closingPrice)
}

func (r *ListingRepository) GetHistory(listingID int64) ([]model.ListingHistory, error) {
	rows, err := r.db.Query(`
		SELECT h.id, h.listing_id, h.user_id, h.action,
		       COALESCE(h.status,''), COALESCE(h.note,''), h.created_at,
		       u.full_name, h.closing_price
		FROM listing_history h
		JOIN users u ON u.id = h.user_id
		WHERE h.listing_id = $1
		ORDER BY h.created_at DESC`, listingID)
	if err != nil { return nil, err }
	defer rows.Close()
	var history []model.ListingHistory
	for rows.Next() {
		var h model.ListingHistory
		if err := rows.Scan(&h.ID, &h.ListingID, &h.UserID, &h.Action,
			&h.Status, &h.Note, &h.CreatedAt, &h.UserName, &h.ClosingPrice); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

func (r *ListingRepository) ExistsByCustomerID(customerID, excludeListingID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM listings WHERE customer_id=$1 AND is_active=true AND id!=$2`,
		customerID, excludeListingID,
	).Scan(&count)
	return count > 0, err
}
