package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type RequestRepository struct {
	db *sql.DB
}

func NewRequestRepository(db *sql.DB) *RequestRepository {
	return &RequestRepository{db: db}
}

type RequestFilter struct {
	UserID       int64
	ListingType  string
	PropertyType string
	District     string
	Search       string
	OnlyActive   bool
}

func (r *RequestRepository) List(f RequestFilter) ([]model.Request, error) {
	args := []interface{}{}
	where := []string{}
	i := 1

	if f.UserID > 0 {
		where = append(where, fmt.Sprintf("r.user_id = $%d", i))
		args = append(args, f.UserID)
		i++
	}
	if f.OnlyActive {
		where = append(where, "r.is_active = true")
	}
	if f.ListingType != "" {
		where = append(where, fmt.Sprintf("r.fields->>'listing_type' = $%d", i))
		args = append(args, f.ListingType)
		i++
	}
	if f.PropertyType != "" {
		where = append(where, fmt.Sprintf("r.fields->>'property_type' = $%d", i))
		args = append(args, f.PropertyType)
		i++
	}
	if f.District != "" {
		where = append(where, fmt.Sprintf("r.fields->>'district' = $%d", i))
		args = append(args, f.District)
		i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(r.fields->>'client_name' ILIKE $%d OR r.fields->>'district' ILIKE $%d)", i, i))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.is_active, r.notify_me,
		       r.fields, r.created_at, r.updated_at,
		       u.full_name as owner_name
		FROM requests r
		JOIN users u ON u.id = r.user_id
		%s
		ORDER BY r.created_at DESC`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Request
	for rows.Next() {
		var req model.Request
		var fieldsJSON []byte
		if err := rows.Scan(
			&req.ID, &req.UserID, &req.IsActive, &req.NotifyMe,
			&fieldsJSON, &req.CreatedAt, &req.UpdatedAt,
			&req.OwnerName,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(fieldsJSON, &req.Fields); err != nil {
			req.Fields = map[string]string{}
		}
		list = append(list, req)
	}
	return list, nil
}

func (r *RequestRepository) GetByID(id int64) (*model.Request, error) {
	req := &model.Request{}
	var fieldsJSON []byte
	err := r.db.QueryRow(`
		SELECT r.id, r.user_id, r.is_active, r.notify_me,
		       r.fields, r.created_at, r.updated_at,
		       u.full_name as owner_name
		FROM requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1`, id,
	).Scan(&req.ID, &req.UserID, &req.IsActive, &req.NotifyMe,
		&fieldsJSON, &req.CreatedAt, &req.UpdatedAt, &req.OwnerName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(fieldsJSON, &req.Fields)
	return req, nil
}

func (r *RequestRepository) Create(req *model.Request) error {
	fieldsJSON, err := json.Marshal(req.Fields)
	if err != nil {
		return err
	}
	return r.db.QueryRow(`
		INSERT INTO requests (user_id, notify_me, fields)
		VALUES ($1,$2,$3)
		RETURNING id, created_at, updated_at`,
		req.UserID, req.NotifyMe, fieldsJSON,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

func (r *RequestRepository) Update(req *model.Request) error {
	fieldsJSON, err := json.Marshal(req.Fields)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		UPDATE requests SET fields=$1, updated_at=NOW()
		WHERE id=$2 AND user_id=$3`,
		fieldsJSON, req.ID, req.UserID)
	return err
}

func (r *RequestRepository) ToggleActive(id, userID int64, isAdmin bool) error {
	var err error
	if isAdmin {
		_, err = r.db.Exec(
			`UPDATE requests SET is_active = NOT is_active, updated_at=NOW() WHERE id=$1`, id)
	} else {
		_, err = r.db.Exec(
			`UPDATE requests SET is_active = NOT is_active, updated_at=NOW()
			 WHERE id=$1 AND user_id=$2`, id, userID)
	}
	return err
}

func (r *RequestRepository) ToggleNotify(id, userID int64) error {
	_, err := r.db.Exec(`
		UPDATE requests SET notify_me = NOT notify_me, updated_at=NOW()
		WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *RequestRepository) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM requests WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("talep bulunamadı")
	}
	return nil
}

func (r *RequestRepository) IsOwner(requestID, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM requests WHERE id=$1 AND user_id=$2`, requestID, userID,
	).Scan(&count)
	return count > 0, err
}
