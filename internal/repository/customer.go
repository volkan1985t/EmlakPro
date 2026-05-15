package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) List(userID int64, isAdmin bool, search string) ([]model.Customer, error) {
	args  := []interface{}{}
	where := []string{}
	i := 1

	if !isAdmin {
		where = append(where, fmt.Sprintf("c.user_id = $%d", i))
		args = append(args, userID); i++
	}
	if search != "" {
		where = append(where, fmt.Sprintf(
			"(c.name ILIKE $%d OR c.phone ILIKE $%d OR c.email ILIKE $%d)", i, i, i))
		args = append(args, "%"+search+"%"); i++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, c.source,
		       COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at,
		       u.full_name as owner_name
		FROM customers c
		JOIN users u ON u.id = c.user_id
		%s
		ORDER BY c.created_at DESC`, whereClause), args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var c model.Customer
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Email, &c.Source,
			&c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.OwnerName); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func (r *CustomerRepository) GetByID(id int64) (*model.Customer, error) {
	var c model.Customer
	err := r.db.QueryRow(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, c.source,
		       COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at,
		       u.full_name as owner_name
		FROM customers c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Email, &c.Source,
		&c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.OwnerName)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	return &c, nil
}

func (r *CustomerRepository) Create(c *model.Customer) error {
	return r.db.QueryRow(`
		INSERT INTO customers (user_id, name, phone, email, source, notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		c.UserID, c.Name, c.Phone, c.Email, c.Source, c.Notes,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepository) Update(c *model.Customer) error {
	_, err := r.db.Exec(`
		UPDATE customers SET name=$1, phone=$2, email=$3, source=$4, notes=$5, updated_at=NOW()
		WHERE id=$6`,
		c.Name, c.Phone, c.Email, c.Source, c.Notes, c.ID)
	return err
}

func (r *CustomerRepository) SetActive(id int64, active bool) error {
	_, err := r.db.Exec(`UPDATE customers SET is_active=$1, updated_at=NOW() WHERE id=$2`, active, id)
	return err
}

func (r *CustomerRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM customers WHERE id=$1`, id)
	return err
}

func (r *CustomerRepository) IsOwner(customerID, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM customers WHERE id=$1 AND user_id=$2`, customerID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *CustomerRepository) LinkListing(customerID, listingID int64, note string) error {
	_, err := r.db.Exec(`
		INSERT INTO customer_listings (customer_id, listing_id, note)
		VALUES ($1,$2,$3)
		ON CONFLICT (customer_id, listing_id) DO UPDATE SET note=EXCLUDED.note`,
		customerID, listingID, note)
	return err
}

func (r *CustomerRepository) UnlinkListing(customerID, listingID int64) error {
	_, err := r.db.Exec(
		`DELETE FROM customer_listings WHERE customer_id=$1 AND listing_id=$2`,
		customerID, listingID)
	return err
}

func (r *CustomerRepository) GetLinkedListings(customerID int64) ([]model.Listing, error) {
	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT %s
		FROM listings l
		JOIN users u ON u.id = l.user_id
		JOIN customer_listings cl ON cl.listing_id = l.id
		WHERE cl.customer_id = $1
		ORDER BY cl.created_at DESC`, listingSelectCols), customerID)
	if err != nil { return nil, err }
	defer rows.Close()

	var listings []model.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil { return nil, err }
		listings = append(listings, *l)
	}
	return listings, nil
}
